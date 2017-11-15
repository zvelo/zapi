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
	"zvelo.io/httpsig"
	"zvelo.io/msg"
	"zvelo.io/msg/mock"
)

type reqData struct {
	url          string
	start        time.Time
	redirectFrom string
}

var (
	keyGetter              httpsig.KeyGetter
	callbackURL            string
	callbackNoValidate     bool
	callbackNoKeyCache     bool
	queryListen            string
	queryNoPoll            bool
	queryNoFollowRedirects bool
	queryRedirectLimit     int
	queryURLs              []string
	queryURLContent        []*msg.URLContent
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

	reqIDDataLock sync.RWMutex
	reqIDData     = map[string]reqData{}
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
	var keyCache callback.KeyCache

	if !callbackNoKeyCache {
		keyCache = callback.FileKeyCache(name)
	}

	if !callbackNoValidate {
		keyGetter = callback.KeyGetter(keyCache)
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
			queryURLContent = append(queryURLContent, &msg.URLContent{
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
			queryURLContent = append(queryURLContent, &msg.URLContent{
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

		queryURLContent = append(queryURLContent, &msg.URLContent{
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
			fmt.Fprintf(os.Stderr, "listening for callbacks at %s\n", queryListen)
			_ = http.ListenAndServe(
				queryListen,
				callback.Middleware(keyGetter, callbackHandler(ctx)),
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
		// wait for the wait group to complete or the context to timeout
		go func() {
			resultWg.Wait()
			cancel()
		}()

		<-ctx.Done()
		return ctx.Err()
	}

	return nil
}

func query(ctx context.Context, queryReq *msg.QueryRequests) ([]string, error) {
	start := time.Now()

	if !queryNoPoll || callbackURL != "" {
		resultWg.Add(len(queryReq.Url) + len(queryReq.Content))
	}

	if rest {
		return queryREST(ctx, start, queryReq)
	}

	return queryGRPC(ctx, start, queryReq)
}

func queryREST(ctx context.Context, start time.Time, queryReq *msg.QueryRequests) ([]string, error) {
	var resp *http.Response
	replies, err := restV1Client.Query(ctx, queryReq, zapi.Response(&resp))
	if err != nil {
		return nil, err
	}

	return queryComplete(ctx, start, queryReq, resp.Header.Get("uber-trace-id"), replies.Reply), nil
}

func queryGRPC(ctx context.Context, start time.Time, queryReq *msg.QueryRequests) ([]string, error) {
	var header metadata.MD
	replies, err := grpcV1Client.Query(ctx, queryReq, grpc.Header(&header))
	if err != nil {
		return nil, err
	}

	var traceID string
	if tids, ok := header["uber-trace-id"]; ok && len(tids) > 0 {
		traceID = tids[0]
	}

	return queryComplete(ctx, start, queryReq, traceID, replies.Reply), nil
}

func getReqIDData(reqID, defURL string) reqData {
	reqIDDataLock.RLock()

	if data, ok := reqIDData[reqID]; ok {
		reqIDDataLock.RUnlock()
		if data.url == "" {
			data.url = defURL
		}
		return data
	}

	reqIDDataLock.RUnlock()

	return reqData{url: defURL}
}

func setReqIDData(reqID, url string, start time.Time) {
	reqIDDataLock.Lock()
	reqIDData[reqID] = reqData{
		url:   url,
		start: start,
	}
	reqIDDataLock.Unlock()
}

func printTraceID(traceID string) string {
	return traceID[:strings.Index(traceID, ":")]
}

func queryComplete(ctx context.Context, start time.Time, queryReq *msg.QueryRequests, traceID string, replies []*msg.QueryReply) []string {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)

	defer func() {
		_ = w.Flush()
		printf := printfFunc(color.FgCyan, os.Stderr)
		printf(buf.String())
	}()

	if traceID != "" {
		fmt.Fprintf(w, "Trace ID:\t%s\n", printTraceID(traceID))
	}

	fmt.Fprintf(w, "Query Duration:\t%s\n", time.Since(start))

	var ret []string

	for i, reply := range replies {
		var u string
		if i < len(queryReq.Url) {
			u = queryReq.Url[i]
		} else if j := i - len(queryReq.Url); j >= 0 && j < len(queryReq.Content) {
			u = queryReq.Content[j].Url

			p, err := url.Parse(u)
			if err != nil {
				errorf("error parsing url (%s): %s\n", u, err)
				continue
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
			errorf("got unexpected reply: %d => %#v\n", i, reply)
			continue
		}

		ret = append(ret, reply.RequestId)

		setReqIDData(reply.RequestId, u, start)
		fmt.Fprintf(w, "%s:\t%s\n", u, reply.RequestId)
	}

	return ret
}

func countRedirects(reqID string) int {
	reqIDDataLock.RLock()
	defer reqIDDataLock.RUnlock()

	for ret := 0; ; ret++ {
		data, ok := reqIDData[reqID]
		if !ok || data.redirectFrom == "" {
			return ret
		}
		reqID = data.redirectFrom
	}
}

func followRedirect(ctx context.Context, result *msg.QueryResult) []string {
	url := getReqIDData(result.RequestId, "").url

	if queryNoFollowRedirects || !isComplete(result) {
		return nil
	}

	qs := result.QueryStatus

	if qs.Location == "" ||
		qs.FetchCode < 300 ||
		qs.FetchCode > 399 {
		return nil
	}

	if qs.Location == url {
		errorf("\nnot redirecting to the same url\n")
		return nil
	}

	num := countRedirects(result.RequestId) + 1
	location := qs.Location

	if num >= queryRedirectLimit {
		errorf("\ntoo many redirects (%d): %s → %s\n", num, url, location)
		return nil
	}

	printf := printfFunc(color.FgYellow, os.Stderr)
	printf("\nfollowing redirect #%d: %s → %s\n", num, url, location)

	reqIDs, err := query(ctx, &msg.QueryRequests{
		Callback: callbackURL,
		Dataset:  datasets,
		Url:      []string{location},
	})

	if err != nil {
		errorf("query error: %s\n", err)
		return nil
	}

	if len(reqIDs) == 0 {
		return nil
	}

	// There should be at most 1 reqID
	reqIDDataLock.Lock()
	data := reqIDData[reqIDs[0]]
	data.redirectFrom = result.RequestId
	reqIDData[reqIDs[0]] = data
	reqIDDataLock.Unlock()

	return reqIDs
}

func callbackHandler(ctx context.Context) callback.Handler {
	return callback.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request, result *msg.QueryResult) {
		followRedirect(ctx, result)
		resultCh <- queryResult{QueryResult: result}
	})
}
