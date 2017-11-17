package query

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
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
	"zvelo.io/zapi/clients"
	"zvelo.io/zapi/internal/zvelo"
	"zvelo.io/zapi/poller"
	"zvelo.io/zapi/results"
	"zvelo.io/zapi/tokensourcer"
)

type reqData struct {
	start        time.Time
	redirectFrom string
}

func defaultDatasets() []string {
	return []string{msg.CATEGORIZATION.String()}
}

type queryCmd struct {
	appName            string
	wg                 sync.WaitGroup
	datasets           []msg.DataSetType
	datasetStrings     cli.StringSlice
	debug, rest        bool
	timeout            time.Duration
	clients            clients.Clients
	poller             poller.Poller
	keyGetter          httpsig.KeyGetter
	callbackURL        string
	callbackNoValidate bool
	callbackNoKeyCache bool
	listen             string
	noPoll             bool
	noFollowRedirects  bool
	redirectLimit      int
	urls               []string
	urlContent         []*msg.URLContent
	mockCategories     cli.StringSlice
	mockMalicious      string
	mockMaliciousClean bool
	mockCompleteAfter  time.Duration
	mockFetchCode      int
	mockLocation       string
	mockErrorCode      int
	mockErrorMessage   string
	mockContextOpts    []mock.ContextOption
	contents           cli.StringSlice

	reqIDDataLock sync.RWMutex
	reqIDData     map[string]reqData
}

func (c *queryCmd) Flags() []cli.Flag {
	flags := append(c.clients.Flags(), c.poller.Flags()...)
	return append(flags,
		cli.BoolFlag{
			Name:        "debug",
			EnvVar:      "ZVELO_DEBUG",
			Usage:       "enable debug logging",
			Destination: &c.debug,
		},
		cli.BoolFlag{
			Name:        "rest",
			EnvVar:      "ZVELO_REST",
			Usage:       "Use REST instead of gRPC for api requests",
			Destination: &c.rest,
		},
		cli.DurationFlag{
			Name:        "timeout",
			EnvVar:      "ZVELO_TIMEOUT",
			Usage:       "maximum amount of time to wait for results to complete",
			Value:       15 * time.Minute,
			Destination: &c.timeout,
		},
		cli.StringFlag{
			Name:        "listen",
			EnvVar:      "ZVELO_QUERY_LISTEN_ADDRESS",
			Usage:       "address and port to listen for callbacks",
			Value:       ":8080",
			Destination: &c.listen,
		},
		cli.StringFlag{
			Name:        "callback",
			EnvVar:      "ZVELO_QUERY_CALLBACK_URL",
			Usage:       "publicly accessible base URL that routes to the address used by -listen flag, disables polling",
			Destination: &c.callbackURL,
		},
		cli.BoolFlag{
			Name:        "no-validate-callback",
			EnvVar:      "ZVELO_NO_VALIDATE_CALLBACK",
			Usage:       "do not validate callback signatures",
			Destination: &c.callbackNoValidate,
		},
		cli.BoolFlag{
			Name:        "no-key-cache",
			EnvVar:      "ZVELO_NO_KEY_CACHE",
			Usage:       "do not cache public keys when validating http signatures in callbacks",
			Destination: &c.callbackNoKeyCache,
		},
		cli.BoolFlag{
			Name:        "no-poll",
			EnvVar:      "ZVELO_QUERY_NO_POLL",
			Usage:       "don't poll for resutls",
			Destination: &c.noPoll,
		},
		cli.BoolFlag{
			Name:        "no-follow-redirects",
			EnvVar:      "ZVELO_NO_FOLLOW_REDIRECTS",
			Usage:       "if poll or callback is enabled, follow redirect responses",
			Destination: &c.noFollowRedirects,
		},
		cli.IntFlag{
			Name:        "redirect-limit",
			EnvVar:      "REDIRECT_LIMIT",
			Usage:       "maximum number of redirects to follow for a single request",
			Value:       10,
			Destination: &c.redirectLimit,
		},
		cli.StringSliceFlag{
			Name:  "mock-category",
			Usage: "when querying against the mock server, expect these categories in the categorization response (category id or category short name, may be repeated)",
			Value: &c.mockCategories,
		},
		cli.StringFlag{
			Name:        "mock-malicious-category",
			Usage:       "when querying against the mock server, expect this category in the malicious response and for the verdict to be MALICIOUS (category id or category short name)",
			Destination: &c.mockMalicious,
		},
		cli.BoolFlag{
			Name:        "mock-malicious-clean",
			Usage:       "when querying against the mock server, expect the malicious dataset to return CLEAN with UNKNOWN_CATEGORY",
			Destination: &c.mockMaliciousClean,
		},
		cli.DurationFlag{
			Name:        "mock-complete-after",
			Usage:       "when querying against the mock server, results will not be marked complete until this much time has passed since the query",
			Destination: &c.mockCompleteAfter,
		},
		cli.IntFlag{
			Name:        "mock-fetch-code",
			Usage:       "when querying against the mock server, expect this query status fetch code",
			Destination: &c.mockFetchCode,
		},
		cli.StringFlag{
			Name:        "mock-location",
			Usage:       "when querying against the mock server, expect this query status location",
			Destination: &c.mockLocation,
		},
		cli.IntFlag{
			Name:        "mock-error-code",
			Usage:       "when querying against the mock server, expect this query status error code",
			Destination: &c.mockErrorCode,
		},
		cli.StringFlag{
			Name:        "mock-error-message",
			Usage:       "when querying against the mock server, expect this query status error message",
			Destination: &c.mockErrorMessage,
		},
		cli.StringSliceFlag{
			Name: "content",
			Usage: "get datasets for the content provided (as opposed to fetching a URL and getting datasets for the content received)." +
				" if you start the content with the letter @, the rest should be a file name to read the data from, or - if you want zapi to read the data from stdin." +
				" (may be repeated)",
			Value: &c.contents,
		},
		cli.StringSliceFlag{
			Name:   "dataset",
			EnvVar: "ZVELO_DATASETS",
			Usage:  "list of datasets to retrieve, may be repeated (available options: " + strings.Join(availableDS(), ", ") + ", default: " + strings.Join(defaultDatasets(), ", ") + ")",
			Value:  &c.datasetStrings,
		},
	)
}

func availableDS() []string {
	var ds []string
	for dst, name := range msg.DataSetType_name {
		if dst == int32(msg.ECHO) {
			continue
		}

		ds = append(ds, name)
	}
	return ds
}

func Command(appName string) cli.Command {
	c := queryCmd{
		appName:   appName,
		reqIDData: map[string]reqData{},
	}

	tokenSourcer := tokensourcer.New(appName, &c.debug)
	c.clients = clients.New(tokenSourcer, &c.debug)
	c.poller = poller.New(&c.debug, &c.rest, c.clients)

	return cli.Command{
		Name:      "query",
		Usage:     "query for a URL",
		ArgsUsage: "url [url...]",
		Before:    c.setup,
		Action:    c.action,
		Flags:     c.Flags(),
	}
}

func (c *queryCmd) setupMock() error {
	var cats []msg.Category
	for _, catName := range c.mockCategories {
		cat := msg.ParseCategory(catName)
		if cat == msg.UNKNOWN_CATEGORY {
			return errors.Errorf("invalid category: %s", catName)
		}
		cats = append(cats, cat)
	}

	if len(cats) > 0 {
		c.mockContextOpts = append(c.mockContextOpts, mock.WithCategories(cats...))
	}

	if c.mockMaliciousClean {
		c.mockContextOpts = append(c.mockContextOpts, mock.WithMalicious(msg.VERDICT_CLEAN, msg.UNKNOWN_CATEGORY))
	}

	if c.mockMalicious != "" {
		malcat := msg.ParseCategory(c.mockMalicious)
		if malcat == msg.UNKNOWN_CATEGORY {
			return errors.Errorf("invalid category: %s", c.mockMalicious)
		}
		c.mockContextOpts = append(c.mockContextOpts, mock.WithMalicious(msg.VERDICT_MALICIOUS, msg.Category(malcat)))
	}

	if c.mockCompleteAfter > 0 {
		c.mockContextOpts = append(c.mockContextOpts, mock.WithCompleteAfter(c.mockCompleteAfter))
	}

	if c.mockFetchCode != 0 {
		c.mockContextOpts = append(c.mockContextOpts, mock.WithFetchCode(int32(c.mockFetchCode)))
	}

	if c.mockLocation != "" {
		c.mockContextOpts = append(c.mockContextOpts, mock.WithLocation(c.mockLocation))
	}

	if c.mockErrorCode != 0 || c.mockErrorMessage != "" {
		c.mockContextOpts = append(c.mockContextOpts, mock.WithError(codes.Code(c.mockErrorCode), c.mockErrorMessage))
	}

	return nil
}

func (c *queryCmd) setupContents() error {
	for _, content := range c.contents {
		if len(content) == 0 || content == "@" {
			continue
		}

		// no '@' implies the data is provided directly
		if content[0] != '@' {
			c.urlContent = append(c.urlContent, &msg.URLContent{
				Content: content,
			})
			continue
		}

		// '@-' means we need to read from stdin
		if content == "@-" {
			var buf bytes.Buffer
			if _, err := buf.ReadFrom(os.Stdin); err != nil {
				return err
			}
			c.urlContent = append(c.urlContent, &msg.URLContent{
				Content: buf.String(),
			})
			continue
		}

		// anything else beginning with '@' implies that the value following the
		// '@' is a filename that should be read for the content
		data, err := ioutil.ReadFile(content[1:])
		if err != nil {
			return err
		}

		c.urlContent = append(c.urlContent, &msg.URLContent{
			Content: string(data),
		})
	}

	return nil
}

func (c *queryCmd) setupDataSets() error {
	if len(c.datasetStrings) == 0 {
		c.datasetStrings = defaultDatasets()
	}

	for _, dsName := range c.datasetStrings {
		dsName = strings.TrimSpace(dsName)

		dst, err := msg.NewDataSetType(dsName)
		if err != nil {
			zvelo.Errorf("invalid dataset type: %s\n", dsName)
			continue
		}

		c.datasets = append(c.datasets, dst)
	}

	if len(c.datasets) == 0 {
		return errors.New("at least one valid dataset is required")
	}

	return nil
}

func (c *queryCmd) setup(cli *cli.Context) error {
	if err := c.setupDataSets(); err != nil {
		return err
	}

	var keyCache callback.KeyCache

	if !c.callbackNoKeyCache {
		keyCache = callback.FileKeyCache(c.appName)
	}

	if !c.callbackNoValidate {
		c.keyGetter = callback.KeyGetter(keyCache)
	}

	if c.callbackURL != "" {
		c.noPoll = true
	}

	if err := c.setupMock(); err != nil {
		return err
	}

	if len(cli.Args()) == 0 && len(c.contents) == 0 {
		return errors.New("at least one url or content is required")
	}

	if err := c.setupContents(); err != nil {
		return err
	}

	for _, u := range cli.Args() {
		if u == "" {
			continue
		}

		if !strings.Contains(u, "://") {
			u = "http://" + u
		}

		c.urls = append(c.urls, u)
	}

	if c.callbackURL != "" {
		if !strings.Contains(c.callbackURL, "://") {
			c.callbackURL = "http://" + c.callbackURL
		}
	}

	return nil
}

func (c *queryCmd) action(_ *cli.Context) error {
	ctx := mock.QueryContext(context.Background(), c.mockContextOpts...)
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	if c.callbackURL != "" {
		go func() {
			fmt.Fprintf(os.Stderr, "listening for callbacks at %s\n", c.listen)
			_ = http.ListenAndServe(
				c.listen,
				callback.Middleware(c.keyGetter, c.callbackHandler(ctx)),
			)
		}()
	}

	queryReq := msg.QueryRequests{
		Callback: c.callbackURL,
		Dataset:  c.datasets,
		Url:      c.urls,
		Content:  c.urlContent,
	}

	requests, err := c.query(ctx, &queryReq)
	if err != nil {
		return err
	}

	if !c.noPoll {
		go c.poller.Poll(ctx, requests, c)
	}

	if !c.noPoll || c.callbackURL != "" {
		// wait for the wait group to complete or the context to timeout
		go func() {
			c.wg.Wait()
			cancel()
		}()

		<-ctx.Done()
		return ctx.Err()
	}

	return nil
}

func (c *queryCmd) query(ctx context.Context, queryReq *msg.QueryRequests) (poller.Requests, error) {
	start := time.Now()

	if !c.noPoll || c.callbackURL != "" {
		c.wg.Add(len(queryReq.Url) + len(queryReq.Content))
	}

	if c.rest {
		return c.queryREST(ctx, start, queryReq)
	}

	return c.queryGRPC(ctx, start, queryReq)
}

func (c *queryCmd) queryREST(ctx context.Context, start time.Time, queryReq *msg.QueryRequests) (poller.Requests, error) {
	var resp *http.Response
	replies, err := c.clients.RESTv1().Query(ctx, queryReq, zapi.Response(&resp))
	if err != nil {
		return nil, err
	}

	return c.queryComplete(ctx, start, queryReq, resp.Header.Get("uber-trace-id"), replies.Reply), nil
}

func (c *queryCmd) queryGRPC(ctx context.Context, start time.Time, queryReq *msg.QueryRequests) (poller.Requests, error) {
	client, err := c.clients.GRPCv1(ctx)
	if err != nil {
		return nil, err
	}

	var header metadata.MD
	replies, err := client.Query(ctx, queryReq, grpc.Header(&header))
	if err != nil {
		return nil, err
	}

	var traceID string
	if tids, ok := header["uber-trace-id"]; ok && len(tids) > 0 {
		traceID = tids[0]
	}

	return c.queryComplete(ctx, start, queryReq, traceID, replies.Reply), nil
}

func (c *queryCmd) getReqIDStart(reqID string) time.Time {
	c.reqIDDataLock.RLock()
	defer c.reqIDDataLock.RUnlock()
	return c.reqIDData[reqID].start
}

func (c *queryCmd) setReqIDStart(reqID string, start time.Time) {
	c.reqIDDataLock.Lock()
	c.reqIDData[reqID] = reqData{start: start}
	c.reqIDDataLock.Unlock()
}

func (c *queryCmd) setReqIDRedirectFrom(reqID, fromReqID string) {
	c.reqIDDataLock.Lock()
	data := c.reqIDData[reqID]
	data.redirectFrom = fromReqID
	c.reqIDData[reqID] = data
	c.reqIDDataLock.Unlock()
}

func (c *queryCmd) queryComplete(ctx context.Context, start time.Time, queryReq *msg.QueryRequests, traceID string, replies []*msg.QueryReply) poller.Requests {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)

	defer func() {
		_ = w.Flush()
		printf := zvelo.PrintfFunc(color.FgCyan, os.Stderr)
		printf(buf.String())
	}()

	if traceID != "" {
		fmt.Fprintf(w, "Trace ID:\t%s\n", zvelo.TraceIDString(traceID))
	}

	fmt.Fprintf(w, "Query Duration:\t%s\n", time.Since(start))

	ret := poller.Requests{}

	for i, reply := range replies {
		var u string
		if i < len(queryReq.Url) {
			u = queryReq.Url[i]
		} else if j := i - len(queryReq.Url); j >= 0 && j < len(queryReq.Content) {
			u = queryReq.Content[j].Url

			p, err := url.Parse(u)
			if err != nil {
				zvelo.Errorf("error parsing url (%s): %s\n", u, err)
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
			zvelo.Errorf("got unexpected reply: %d => %#v\n", i, reply)
			continue
		}

		ret[reply.RequestId] = u

		c.setReqIDStart(reply.RequestId, start)
		fmt.Fprintf(w, "%s:\t%s\n", u, reply.RequestId)
	}

	return ret
}

func (c *queryCmd) countRedirects(reqID string) int {
	c.reqIDDataLock.RLock()
	defer c.reqIDDataLock.RUnlock()

	for ret := 0; ; ret++ {
		data, ok := c.reqIDData[reqID]
		if !ok || data.redirectFrom == "" {
			return ret
		}
		reqID = data.redirectFrom
	}
}

func (c *queryCmd) Result(ctx context.Context, result *results.Result) poller.Requests {
	if zvelo.IsComplete(result.QueryResult) {
		defer c.wg.Done()
	}

	result.Start = c.getReqIDStart(result.RequestId)

	results.Print(result)

	if c.noFollowRedirects || !zvelo.IsComplete(result.QueryResult) {
		return nil
	}

	qs := result.QueryStatus

	if qs.Location == "" ||
		qs.FetchCode < 300 ||
		qs.FetchCode > 399 {
		return nil
	}

	if qs.Location == result.Url {
		zvelo.Errorf("\nnot redirecting to the same url\n")
		return nil
	}

	num := c.countRedirects(result.RequestId) + 1
	location := qs.Location

	if num >= c.redirectLimit {
		zvelo.Errorf("\ntoo many redirects (%d): %s → %s\n", num, result.Url, location)
		return nil
	}

	printf := zvelo.PrintfFunc(color.FgYellow, os.Stderr)
	printf("\nfollowing redirect #%d: %s → %s\n", num, result.Url, location)

	requests, err := c.query(ctx, &msg.QueryRequests{
		Callback: c.callbackURL,
		Dataset:  c.datasets,
		Url:      []string{location},
	})

	if err != nil {
		zvelo.Errorf("query error: %s\n", err)
		return nil
	}

	// There should be at most 1 reqID
	for reqID := range requests {
		c.setReqIDRedirectFrom(reqID, result.RequestId)
	}

	return requests
}

func (c *queryCmd) callbackHandler(ctx context.Context) callback.Handler {
	return callback.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request, result *msg.QueryResult) {
		c.Result(ctx, &results.Result{QueryResult: result})
	})
}