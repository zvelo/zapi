package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"zvelo.io/go-zapi"
	"zvelo.io/go-zapi/callback"
	"zvelo.io/msg"
	"zvelo.io/msg/mock"
)

var (
	callbackOpts           []callback.Option
	callbackURL            string
	callbackNoValidate     bool
	callbackNoKeyCache     bool
	queryListen            string
	queryNoPoll            bool
	queryNoFollowRedirects bool
	queryRedirectLimit     int
	queryURLs              []string
	queryURLContent        []*msg.QueryRequests_URLContent
	mockCategories         cli.StringSlice
	mockMalicious          string
	mockMaliciousClean     bool
	mockCompleteAfter      time.Duration
	mockFetchCode          int
	mockLocation           string
	mockErrorCode          int
	mockErrorMessage       string
	mockContextOpts        []mock.ContextOption
	contents               cli.StringSlice

	reqIDtoURLLock sync.RWMutex
	reqIDtoURL     = map[string]string{}

	redirectsLock sync.RWMutex
	redirects     = map[string]string{}
)

func init() {
	cmd := cli.Command{
		Name:      "query",
		Usage:     "query for a URL",
		ArgsUsage: "url [url...]",
		Before:    setupQuery,
		Action:    runQuery,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:        "listen",
				EnvVar:      "ZVELO_QUERY_LISTEN_ADDRESS",
				Usage:       "address and port to listen for callbacks",
				Value:       ":8080",
				Destination: &queryListen,
			},
			cli.StringFlag{
				Name:        "callback",
				EnvVar:      "ZVELO_QUERY_CALLBACK_URL",
				Usage:       "publicly accessible base URL that routes to the address used by -listen flag, disables polling",
				Destination: &callbackURL,
			},
			cli.BoolFlag{
				Name:        "no-validate-callback",
				EnvVar:      "ZVELO_NO_VALIDATE_CALLBACK",
				Usage:       "do not validate callback signatures",
				Destination: &callbackNoValidate,
			},
			cli.BoolFlag{
				Name:        "no-key-cache",
				EnvVar:      "ZVELO_NO_KEY_CACHE",
				Usage:       "do not cache public keys when validating http signatures in callbacks",
				Destination: &callbackNoKeyCache,
			},
			cli.BoolFlag{
				Name:        "no-poll",
				EnvVar:      "ZVELO_QUERY_NO_POLL",
				Usage:       "don't poll for resutls",
				Destination: &queryNoPoll,
			},
			cli.BoolFlag{
				Name:        "no-follow-redirects",
				EnvVar:      "ZVELO_NO_FOLLOW_REDIRECTS",
				Usage:       "if poll or callback is enabled, follow redirect responses",
				Destination: &queryNoFollowRedirects,
			},
			cli.IntFlag{
				Name:        "redirect-limit",
				EnvVar:      "REDIRECT_LIMIT",
				Usage:       "maximum number of redirects to follow for a single request",
				Value:       10,
				Destination: &queryRedirectLimit,
			},
			cli.StringSliceFlag{
				Name:  "mock-category",
				Usage: "when querying against the mock server, expect these categories in the categorization response (category id or category short name, may be repeated)",
				Value: &mockCategories,
			},
			cli.StringFlag{
				Name:        "mock-malicious-category",
				Usage:       "when querying against the mock server, expect this category in the malicious response and for the verdict to be MALICIOUS (category id or category short name)",
				Destination: &mockMalicious,
			},
			cli.BoolFlag{
				Name:        "mock-malicious-clean",
				Usage:       "when querying against the mock server, expect the malicious dataset to return CLEAN with UNKNOWN_CATEGORY",
				Destination: &mockMaliciousClean,
			},
			cli.DurationFlag{
				Name:        "mock-complete-after",
				Usage:       "when querying against the mock server, results will not be marked complete until this much time has passed since the query",
				Destination: &mockCompleteAfter,
			},
			cli.IntFlag{
				Name:        "mock-fetch-code",
				Usage:       "when querying against the mock server, expect this query status fetch code",
				Destination: &mockFetchCode,
			},
			cli.StringFlag{
				Name:        "mock-location",
				Usage:       "when querying against the mock server, expect this query status location",
				Destination: &mockLocation,
			},
			cli.IntFlag{
				Name:        "mock-error-code",
				Usage:       "when querying against the mock server, expect this query status error code",
				Destination: &mockErrorCode,
			},
			cli.StringFlag{
				Name:        "mock-error-message",
				Usage:       "when querying against the mock server, expect this query status error message",
				Destination: &mockErrorMessage,
			},
			cli.StringSliceFlag{
				Name: "content",
				Usage: "get datasets for the content provided (as opposed to fetching a URL and getting datasets for the content received)." +
					" if you start the content with the letter @, the rest should be a file name to read the data from, or - if you want zapi to read the data from stdin." +
					" (may be repeated)",
				Value: &contents,
			},
		},
	}
	cmd.BashComplete = bashCommandComplete(cmd)
	app.Commands = append(app.Commands, cmd)
}

func parseCategory(name string) (msg.Category, error) {
	if cid, err := strconv.Atoi(name); err == nil {
		if _, ok := msg.Category_name[int32(cid)]; ok {
			return msg.Category(cid), nil
		}
	}

	name = strings.ToUpper(name)

	if cid, ok := msg.Category_value[name]; ok {
		return msg.Category(cid), nil
	}

	if cid, ok := msg.Category_value[name+"_4"]; ok {
		return msg.Category(cid), nil
	}

	return msg.UNKNOWN_CATEGORY, errors.Errorf("invalid category: %s", name)
}

func setupQuery(c *cli.Context) error {
	if debug {
		callbackOpts = append(callbackOpts, callback.WithDebug(os.Stderr))
	}

	if callbackNoValidate {
		callbackOpts = append(callbackOpts, callback.WithoutValidation())
	}

	if callbackNoKeyCache {
		callbackOpts = append(callbackOpts, callback.WithoutCache())
	}

	if callbackURL != "" {
		queryNoPoll = true
	}

	var cats []msg.Category
	for _, c := range mockCategories {
		cat, err := parseCategory(c)
		if err != nil {
			return err
		}
		cats = append(cats, cat)
	}

	if len(cats) > 0 {
		mockContextOpts = append(mockContextOpts, mock.WithCategories(cats...))
	}

	if mockMaliciousClean {
		mockContextOpts = append(mockContextOpts, mock.WithMalicious(msg.VERDICT_CLEAN, msg.UNKNOWN_CATEGORY))
	}

	if mockMalicious != "" {
		malcat, err := parseCategory(mockMalicious)
		if err != nil {
			return err
		}
		mockContextOpts = append(mockContextOpts, mock.WithMalicious(msg.VERDICT_MALICIOUS, msg.Category(malcat)))
	}

	if mockCompleteAfter > 0 {
		mockContextOpts = append(mockContextOpts, mock.WithCompleteAfter(mockCompleteAfter))
	}

	if mockFetchCode != 0 {
		mockContextOpts = append(mockContextOpts, mock.WithFetchCode(int32(mockFetchCode)))
	}

	if mockLocation != "" {
		mockContextOpts = append(mockContextOpts, mock.WithLocation(mockLocation))
	}

	if mockErrorCode != 0 || mockErrorMessage != "" {
		mockContextOpts = append(mockContextOpts, mock.WithError(codes.Code(mockErrorCode), mockErrorMessage))
	}

	if len(c.Args()) == 0 && len(contents) == 0 {
		return errors.New("at least one url or content is required")
	}

	for _, c := range contents {
		if len(c) == 0 || c == "@" {
			continue
		}

		// no '@' implies the data is provided directly
		if c[0] != '@' {
			queryURLContent = append(queryURLContent, &msg.QueryRequests_URLContent{
				Content: c,
			})
			continue
		}

		// '@-' means we need to read from stdin
		if c == "@-" {
			var buf bytes.Buffer
			if _, err := buf.ReadFrom(os.Stdin); err != nil {
				return err
			}
			queryURLContent = append(queryURLContent, &msg.QueryRequests_URLContent{
				Content: buf.String(),
			})
			continue
		}

		// anything else beginning with '@' implies that the value following the
		// '@' is a filename that should be read for the content
		data, err := ioutil.ReadFile(c[1:])
		if err != nil {
			return err
		}

		queryURLContent = append(queryURLContent, &msg.QueryRequests_URLContent{
			Content: string(data),
		})
	}

	for _, u := range c.Args() {
		if u == "" {
			continue
		}

		if !strings.Contains(u, "://") {
			u = "http://" + u
		}

		queryURLs = append(queryURLs, u)
	}

	if callbackURL != "" {
		if !strings.Contains(callbackURL, "://") {
			callbackURL = "http://" + callbackURL
		}
	}

	return nil
}

func runQuery(_ *cli.Context) error {
	ctx := mock.QueryContext(context.Background(), mockContextOpts...)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if !queryNoPoll || callbackURL != "" {
		go resultHandler()
	}

	if callbackURL != "" {
		go func() {
			_ = http.ListenAndServe(
				queryListen,
				callback.HTTPHandler(name, callbackHandler(ctx), callbackOpts...),
			)
		}()
	}

	queryReq := msg.QueryRequests{
		Callback: callbackURL,
		Dataset:  datasets,
		Url:      queryURLs,
		Content:  queryURLContent,
	}

	var err error
	if pollRequestIDs, err = query(ctx, &queryReq); err != nil {
		return err
	}

	if !queryNoPoll {
		go pollHandler(ctx, followRedirect)
	}

	if !queryNoPoll || callbackURL != "" {
		resultWg.Wait()
	}

	return nil
}

func query(ctx context.Context, queryReq *msg.QueryRequests) ([]string, error) {
	if rest {
		return queryREST(ctx, queryReq)
	}

	return queryGRPC(ctx, queryReq)
}

func queryREST(ctx context.Context, queryReq *msg.QueryRequests) ([]string, error) {
	var resp *http.Response
	replies, err := restV1Client.Query(ctx, queryReq, zapi.Response(&resp))
	if err != nil {
		return nil, err
	}

	return queryComplete(ctx, queryReq, resp.Header.Get("uber-trace-id"), replies.Reply)
}

func queryGRPC(ctx context.Context, queryReq *msg.QueryRequests) ([]string, error) {
	var header metadata.MD
	replies, err := grpcV1Client.Query(ctx, queryReq, grpc.Header(&header))
	if err != nil {
		return nil, err
	}

	var traceID string
	if tids, ok := header["uber-trace-id"]; ok && len(tids) > 0 {
		traceID = tids[0]
	}

	return queryComplete(ctx, queryReq, traceID, replies.Reply)
}

func urlFromReqID(reqID, def string) string {
	reqIDtoURLLock.RLock()

	if u, ok := reqIDtoURL[reqID]; ok {
		reqIDtoURLLock.RUnlock()
		return u
	}

	reqIDtoURLLock.RUnlock()

	return def
}

func setReqIDURL(reqID, url string) {
	reqIDtoURLLock.Lock()
	reqIDtoURL[reqID] = url
	reqIDtoURLLock.Unlock()
}

func queryComplete(ctx context.Context, queryReq *msg.QueryRequests, traceID string, replies []*msg.QueryReply) ([]string, error) {
	color.Set(color.FgCyan)

	w := tabwriter.NewWriter(os.Stderr, 0, 0, 1, ' ', 0)

	if traceID != "" {
		fmt.Fprintf(w, "Trace ID:\t%s\n", traceID[:strings.Index(traceID, ":")])
	}

	var ret []string

	for i, reply := range replies {
		var u string
		if i < len(queryReq.Url) {
			u = queryReq.Url[i]
		} else if j := i - len(queryReq.Url); j >= 0 && j < len(queryReq.Content) {
			u = queryReq.Content[j].Url

			p, err := url.Parse(u)
			if err != nil {
				return nil, err
			}

			if p.Host == "" {
				c := queryReq.Content[j].Content
				l := len(c)
				if l > 23 {
					u = c[:23] + "..."
				} else if l > 0 {
					u = c
				} else {
					u = "<CONTENT_REQUEST>"
				}
			}
		} else {
			fmt.Fprintf(os.Stderr, "got unexpected reply: %d => %#v\n", i, reply)
			continue
		}

		if !queryNoPoll || callbackURL != "" {
			resultWg.Add(1)
		}

		ret = append(ret, reply.RequestId)

		setReqIDURL(reply.RequestId, u)
		fmt.Fprintf(w, "%s:\t%s\n", u, reply.RequestId)
	}

	if err := w.Flush(); err != nil {
		color.Unset()
		return nil, err
	}

	color.Unset()

	return ret, nil
}

func countRedirects(reqID string) int {
	redirectsLock.RLock()
	defer redirectsLock.RUnlock()

	var ok bool
	for ret := 0; ; ret++ {
		if reqID, ok = redirects[reqID]; !ok {
			return ret
		}
	}
}

func followRedirect(ctx context.Context, result *msg.QueryResult) ([]string, error) {
	url := urlFromReqID(result.RequestId, "")

	if queryNoFollowRedirects ||
		result == nil ||
		result.QueryStatus == nil ||
		!result.QueryStatus.Complete ||
		result.QueryStatus.Location == "" ||
		result.QueryStatus.Location == url ||
		result.QueryStatus.Error != nil ||
		result.QueryStatus.FetchCode < 300 ||
		result.QueryStatus.FetchCode >= 400 {
		return nil, nil
	}

	num := countRedirects(result.RequestId) + 1
	location := result.QueryStatus.Location

	if num >= queryRedirectLimit {
		fmt.Fprintf(os.Stderr, "\ntoo many redirects (%d): %s → %s\n", num, url, location)
		return nil, nil
	}

	fmt.Fprintf(os.Stderr, "\nfollowing redirect #%d: %s → %s\n", num, url, location)

	reqIDs, err := query(ctx, &msg.QueryRequests{
		Callback: callbackURL,
		Dataset:  datasets,
		Url:      []string{location},
	})

	if err != nil || len(reqIDs) == 0 {
		return nil, err
	}

	// There should be at most 1 reqID
	redirectsLock.Lock()
	redirects[reqIDs[0]] = result.RequestId
	redirectsLock.Unlock()

	return reqIDs, nil
}

func callbackHandler(ctx context.Context) callback.Handler {
	return callback.HandlerFunc(func(result *msg.QueryResult) {
		if _, err := followRedirect(ctx, result); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}

		resultCh <- queryResult{result: result}
	})
}
