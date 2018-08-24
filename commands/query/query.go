package query

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"zvelo.io/go-zapi"
	"zvelo.io/go-zapi/callback"
	"zvelo.io/httpsig"
	"zvelo.io/msg/mock"
	msg "zvelo.io/msg/msgpb"
	"zvelo.io/zapi/clients"
	"zvelo.io/zapi/internal/zvelo"
	"zvelo.io/zapi/poller"
	"zvelo.io/zapi/results"
	"zvelo.io/zapi/tokensourcer"
)

var jsonMarshaler = jsonpb.Marshaler{OrigName: true}

func defaultDatasets() []string {
	return []string{msg.CATEGORIZATION.String()}
}

type cmd struct {
	appName                  string
	datasets                 []msg.DatasetType
	datasetStrings           cli.StringSlice
	debug, trace, rest, json bool
	insecureSkipVerify       bool
	timeout                  time.Duration
	clients                  clients.Clients
	poller                   poller.Poller
	keyGetter                httpsig.KeyGetter
	callbackURL              string
	noListen                 bool
	callbackNoValidate       bool
	callbackNoKeyCache       bool
	listen                   string
	noPoll                   bool
	noFollowRedirects        bool
	redirectLimit            int
	urls                     []string
	urlContent               []*msg.URLContent
	mockCategories           cli.StringSlice
	mockMalicious            cli.StringSlice
	mockCompleteAfter        time.Duration
	mockFetchCode            int
	mockLocation             string
	mockErrorCode            int
	mockErrorMessage         string
	mockContextOpts          []mock.ContextOption
	contents                 cli.StringSlice

	queries queries
}

type queryData struct {
	key          string
	reqID        string
	redirectFrom *queryData
	done         bool
}

type queries struct {
	sync.RWMutex
	internal map[string]*queryData
	reqs     map[string]*queryData
	wg       sync.WaitGroup
}

func (q *queries) Add(key string) {
	q.Lock()
	defer q.Unlock()

	if q.internal == nil {
		q.internal = map[string]*queryData{}
	}

	q.internal[key] = &queryData{
		key: key,
	}

	q.wg.Add(1)
}

func (q *queries) SetReqID(key, reqID string) {
	q.Lock()
	defer q.Unlock()

	d, ok := q.internal[key]
	if !ok {
		panic(fmt.Errorf("couldn't set reqID for key %q", key))
	}

	d.reqID = reqID

	if q.reqs == nil {
		q.reqs = map[string]*queryData{}
	}

	q.reqs[reqID] = d
}

func (q *queries) SetRedirect(reqID, fromReqID string) {
	q.Lock()
	defer q.Unlock()

	f, ok := q.reqs[fromReqID]
	if !ok {
		panic(fmt.Errorf("couldn't find 'from' query for reqID %q", fromReqID))
	}

	d, ok := q.reqs[reqID]
	if !ok {
		panic(fmt.Errorf("couldn't find query for redirect %q", reqID))
	}

	d.redirectFrom = f
}

func (q *queries) NumRedirects(reqID string) int {
	q.RLock()
	defer q.RUnlock()

	for ret := 0; ; ret++ {
		d, ok := q.reqs[reqID]
		if !ok {
			panic(fmt.Errorf("couldn't count redirect %q", reqID))
		}

		if d.redirectFrom == nil {
			return ret
		}

		reqID = d.redirectFrom.reqID
	}
}

func (q *queries) Done(reqID string) {
	q.Lock()
	defer q.Unlock()

	d, ok := q.reqs[reqID]
	if !ok {
		panic(fmt.Errorf("couldn't complete reqID %q", reqID))
	}

	if !d.done {
		d.done = true
		q.wg.Done()
	}
}

func (q *queries) Wait() {
	q.wg.Wait()
}

func (c *cmd) Flags() []cli.Flag {
	flags := append(c.clients.Flags(), c.poller.Flags()...)
	return append(flags,
		cli.BoolFlag{
			Name:        "debug",
			EnvVar:      "ZVELO_DEBUG",
			Usage:       "enable debug logging",
			Destination: &c.debug,
		},
		cli.BoolFlag{
			Name:        "insecure-skip-verify",
			Usage:       "accept any certificate presented by the server and any host name in that certificate. only for testing.",
			Destination: &c.insecureSkipVerify,
		},
		cli.BoolFlag{
			Name:        "trace",
			EnvVar:      "ZVELO_TRACE",
			Usage:       "request a trace to be generated for each request",
			Destination: &c.trace,
		},
		cli.BoolFlag{
			Name:        "rest",
			EnvVar:      "ZVELO_REST",
			Usage:       "Use REST instead of gRPC for api requests",
			Destination: &c.rest,
		},
		cli.BoolFlag{
			Name:        "json",
			EnvVar:      "ZVELO_JSON",
			Usage:       "Print raw JSON response",
			Destination: &c.json,
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
		cli.BoolFlag{
			Name:        "no-listen",
			EnvVar:      "ZVELO_QUERY_NO_LISTEN",
			Usage:       "do not listen for results when using --callback",
			Destination: &c.noListen,
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
		cli.StringSliceFlag{
			Name:  "mock-malicious-category",
			Usage: "when querying against the mock server, expect this category in the malicious response and for the verdict to be MALICIOUS (category id or category short name, may be repeated)",
			Value: &c.mockMalicious,
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
	for dst, name := range msg.DatasetType_name {
		if dst == int32(msg.ECHO) {
			continue
		}

		ds = append(ds, name)
	}
	return ds
}

func Command(appName string) cli.Command {
	c := cmd{appName: appName}

	tokenSourcer := tokensourcer.New(appName, &c.debug, &c.insecureSkipVerify, strings.Fields(zapi.DefaultScopes)...)
	c.clients = clients.New(tokenSourcer, &c.debug, &c.insecureSkipVerify)
	c.poller = poller.New(&c.debug, &c.rest, &c.trace, c.clients)

	return cli.Command{
		Name:      "query",
		Usage:     "query for a URL",
		ArgsUsage: "url [url...]",
		Before:    c.setup,
		Action:    c.action,
		Flags:     c.Flags(),
	}
}

func (c *cmd) setupMock() error {
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

	var malcats []msg.Category
	for _, catName := range c.mockMalicious {
		cat := msg.ParseCategory(catName)
		if cat == msg.UNKNOWN_CATEGORY {
			return errors.Errorf("invalid category: %s", catName)
		}
		malcats = append(malcats, cat)
	}

	if len(c.mockMalicious) > 0 {
		c.mockContextOpts = append(c.mockContextOpts, mock.WithMalicious(malcats...))
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

func (c *cmd) setupContents() error {
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

func (c *cmd) setupDatasets() error {
	if len(c.datasetStrings) == 0 {
		c.datasetStrings = defaultDatasets()
	}

	for _, dsName := range c.datasetStrings {
		dsName = strings.TrimSpace(dsName)

		dst, err := msg.NewDatasetType(dsName)
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

func (c *cmd) setup(cli *cli.Context) error {
	if err := c.setupDatasets(); err != nil {
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

func (c *cmd) action(_ *cli.Context) error {
	ctx := mock.QueryContext(context.Background(), c.mockContextOpts...)
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	if c.callbackURL != "" && !c.noListen {
		go func() {
			debugWriter := io.Writer(nil)

			if c.debug {
				debugWriter = os.Stderr
			}

			fmt.Fprintf(os.Stderr, "listening for callbacks at %s\n", c.listen) // #nosec
			_ = http.ListenAndServe(
				c.listen,
				callback.Middleware(c.keyGetter, c.callbackHandler(ctx), debugWriter),
			) // #nosec
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
		// c satisfies the poll.Handler interface due to the Result() method
		go c.poller.Poll(ctx, requests, c)
	}

	if !c.noPoll || (c.callbackURL != "" && !c.noListen) {
		// wait for the wait group to complete or the context to timeout
		go func() {
			c.queries.Wait()
			cancel()
		}()

		<-ctx.Done()
		return ctx.Err()
	}

	return nil
}

func (c *cmd) query(ctx context.Context, queryReq *msg.QueryRequests) (poller.Requests, error) {
	var replies *msg.QueryReplies
	var err error

	if c.rest {
		replies, err = c.queryREST(ctx, queryReq)
	} else {
		replies, err = c.queryGRPC(ctx, queryReq)
	}

	if err != nil {
		return nil, errors.Wrap(err, "query error")
	}

	if !c.noPoll || (c.callbackURL != "" && !c.noListen) {
		for _, u := range queryReq.Url {
			c.queries.Add(u)
		}

		for _, u := range queryReq.Content {
			c.queries.Add(u.Url + u.Content)
		}
	}

	return c.queryComplete(ctx, queryReq, replies), nil
}

func (c *cmd) queryREST(ctx context.Context, queryReq *msg.QueryRequests) (*msg.QueryReplies, error) {
	var opts []zapi.CallOption

	if c.trace {
		opts = append(opts, zapi.WithHeader("x-client-trace-id", results.TracingTag().String()))
	}

	replies, err := c.clients.RESTv1().Query(ctx, queryReq, opts...)
	if err != nil {
		return nil, err
	}

	return replies, nil
}

func (c *cmd) queryGRPC(ctx context.Context, queryReq *msg.QueryRequests) (*msg.QueryReplies, error) {
	if c.trace {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-client-trace-id", results.TracingTag().String())
	}

	client, err := c.clients.GRPCv1(ctx)
	if err != nil {
		return nil, err
	}

	replies, err := client.Query(ctx, queryReq)
	if err != nil {
		return nil, err
	}

	return replies, nil
}

func (c *cmd) queryComplete(ctx context.Context, queryReq *msg.QueryRequests, reply *msg.QueryReplies) poller.Requests {
	replies := reply.Reply

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)

	defer func() {
		_ = w.Flush() // #nosec
		printf := zvelo.PrintfFunc(color.FgCyan, os.Stderr)
		printf(buf.String())

		if c.json {
			if err := jsonMarshaler.Marshal(os.Stdout, reply); err != nil {
				zvelo.Errorf("marshal error: %s\n", err)
			}
			fmt.Fprintln(os.Stdout) // #nosec
		}
	}()

	ret := poller.Requests{}

	for i, reply := range replies {
		var u, key string
		if i < len(queryReq.Url) {
			u = queryReq.Url[i]
			key = u
		} else if j := i - len(queryReq.Url); j >= 0 && j < len(queryReq.Content) {
			u = queryReq.Content[j].Url
			key = queryReq.Content[j].Url + queryReq.Content[j].Content

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

		if !c.noPoll || (c.callbackURL != "" && !c.noListen) {
			c.queries.SetReqID(key, reply.RequestId)
		}

		if !c.json {
			fmt.Fprintf(w, "%s:\t%s\n", u, reply.RequestId) // #nosec
		}
	}

	return ret
}

func (c *cmd) Result(ctx context.Context, result *msg.QueryResult) poller.Requests {
	complete := zvelo.IsComplete(result)

	if complete || c.poller.Once() {
		defer c.queries.Done(result.RequestId)
	}

	qs := result.QueryStatus

	isRedirect := qs.Location != "" && qs.FetchCode >= 300 && qs.FetchCode < 400

	if c.debug || c.noFollowRedirects || (complete && !isRedirect) {
		results.Print(result, c.json)
	}

	if c.noFollowRedirects || !complete {
		return nil
	}

	if !isRedirect {
		return nil
	}

	location := qs.Location

	if location[0] == '/' {
		// make relative redirects absolute
		if u, err := url.Parse(result.Url); err == nil {
			u.Path = location
			location = u.String()
		}
	}

	if location == result.Url {
		zvelo.Errorf("\nnot redirecting to the same url\n")
		return nil
	}

	num := c.queries.NumRedirects(result.RequestId) + 1

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
		c.queries.SetRedirect(reqID, result.RequestId)
	}

	return requests
}

func (c *cmd) callbackHandler(ctx context.Context) callback.Handler {
	return callback.HandlerFunc(func(w http.ResponseWriter, _ *http.Request, result *msg.QueryResult) {
		w.WriteHeader(http.StatusOK)
		c.Result(ctx, result)
	})
}
