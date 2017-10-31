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
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	zapi "zvelo.io/go-zapi"
	"zvelo.io/msg"
	"zvelo.io/msg/mock"
)

var (
	queryReq           msg.QueryRequests
	queryListen        string
	queryPoll          bool
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

	queryCh = make(chan *msg.QueryResult)
	reqIDs  = map[string]string{}
)

func init() {
	cmd := cli.Command{
		Name:      "query",
		Usage:     "query for a URL",
		ArgsUsage: "url [url...]",
		Before:    setupQuery,
		Action:    query,
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
				Usage:       "publicly accessible base URL that routes to the address used by -listen flag",
				Destination: &queryReq.Callback,
			},
			cli.BoolFlag{
				Name:        "poll",
				EnvVar:      "ZVELO_QUERY_POLL",
				Usage:       "poll for resutls",
				Destination: &queryPoll,
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
	if queryPoll && queryReq.Callback != "" {
		return errors.New("poll and callback can't both be enabled")
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
			queryReq.Content = append(queryReq.Content, &msg.QueryRequests_URLContent{
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
			queryReq.Content = append(queryReq.Content, &msg.QueryRequests_URLContent{
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

		queryReq.Content = append(queryReq.Content, &msg.QueryRequests_URLContent{
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

		queryReq.Url = append(queryReq.Url, u)
	}

	queryReq.Dataset = datasets

	if queryReq.Callback != "" {
		if !strings.Contains(queryReq.Callback, "://") {
			queryReq.Callback = "http://" + queryReq.Callback
		}

		go func() {
			_ = http.ListenAndServe(
				queryListen,
				zapi.CallbackHandler(callbackHandler(), zapiOpts...),
			)
		}()
	}

	return nil
}

func callbackHandler() zapi.Handler {
	return zapi.HandlerFunc(func(in *msg.QueryResult) {
		queryCh <- in
	})
}

func query(_ *cli.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ctx = mock.QueryContext(ctx, mockContextOpts...)

	if rest {
		return queryREST(ctx)
	}

	return queryGRPC(ctx)
}

func queryREST(ctx context.Context) error {
	var resp *http.Response
	replies, err := restV1Client.Query(ctx, &queryReq, zapi.Response(&resp))
	if err != nil {
		return err
	}

	return queryWait(ctx, resp.Header.Get("uber-trace-id"), replies.Reply)
}

func queryGRPC(ctx context.Context) error {
	var header metadata.MD
	replies, err := grpcV1Client.Query(ctx, &queryReq, grpc.Header(&header))
	if err != nil {
		return err
	}

	var traceID string
	if tids, ok := header["uber-trace-id"]; ok && len(tids) > 0 {
		traceID = tids[0]
	}

	return queryWait(ctx, traceID, replies.Reply)
}

var queryResultTplStr = `
{{define "DataSet" -}}
{{- if .Categorization -}}
Categories:         {{range .Categorization.Value}}{{.}} {{end}}
{{end}}

{{- if .Malicious -}}
Malicious:          {{malicious .Malicious}}
{{end}}

{{- if .Echo}}Echo:               {{.Echo.Url}}
{{end}}
{{- end}}

{{define "Status" -}}
Error Code:         {{errorcode .Code}}
{{if .Message}}Error Message:      {{.Message}}
{{end}}
{{- end}}

{{define "QueryStatus" -}}
Complete:           {{if .}}{{.Complete}}
{{else}}false
{{end}}
{{- if . -}}
{{if .FetchCode}}Fetch Status:       {{httpStatus .FetchCode}}
{{end}}
{{- if .Location}}Redirect Location:  {{.Location}}
{{end}}
{{- if .Error}}{{template "Status" .Error}}{{end}}
{{- end}}
{{- end}}

{{define "QueryResult" -}}
{{- if url .RequestId}}URL/Content:        {{url .RequestId}}
{{end}}
{{- if .RequestId}}Request ID:         {{.RequestId}}
{{end}}
{{- if .ResponseDataset}}{{template "DataSet" .ResponseDataset}}{{end}}
{{- template "QueryStatus" .QueryStatus}}
{{- end}}`

var queryResultTpl = template.Must(template.New("QueryResult").Funcs(template.FuncMap{
	"url": func(reqID string) string {
		u, ok := reqIDs[reqID]
		if !ok {
			return "<UNKNOWN>"
		}
		return u
	},
	"malicious": func(m *msg.DataSet_Malicious) string {
		if m.Verdict == msg.VERDICT_MALICIOUS {
			return m.Category.String()
		}

		return m.Verdict.String()
	},
	"httpStatus": func(i int32) string {
		return fmt.Sprintf("%s(%d)", http.StatusText(int(i)), i)
	},
	"errorcode": func(i int32) string {
		return fmt.Sprintf("%s(%d)", codes.Code(i), i)
	},
}).Parse(queryResultTplStr))

func queryWait(ctx context.Context, traceID string, replies []*msg.QueryReply) error {
	color.Set(color.FgCyan)

	w := tabwriter.NewWriter(os.Stderr, 0, 0, 1, ' ', 0)

	if traceID != "" {
		fmt.Fprintf(w, "Trace ID:\t%s\n", traceID[:strings.Index(traceID, ":")])
	}

	for i, reply := range replies {
		var u string
		if i < len(queryReq.Url) {
			u = queryReq.Url[i]
		} else if j := i - len(queryReq.Url); j >= 0 && j < len(queryReq.Content) {
			u = queryReq.Content[j].Url

			p, err := url.Parse(u)
			if err != nil {
				return err
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

		reqIDs[reply.RequestId] = u
		fmt.Fprintf(w, "%s:\t%s\n", u, reply.RequestId)
	}

	if err := w.Flush(); err != nil {
		color.Unset()
		return err
	}

	color.Unset()

	if queryReq.Callback != "" {
		return queryWaitCallback(ctx)
	}

	if queryPoll {
		return pollReqIDs(ctx)
	}

	return nil
}

func queryWaitCallback(ctx context.Context) error {
	var numComplete int

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result := <-queryCh:
			fmt.Fprintf(os.Stderr, "\nreceived callback\n")

			fmt.Println()
			color.Set(color.FgCyan)
			if err := queryResultTpl.ExecuteTemplate(os.Stdout, "QueryResult", result); err != nil {
				color.Unset()
				return err
			}
			color.Unset()

			if result.QueryStatus != nil && result.QueryStatus.Complete {
				numComplete++

				if numComplete == len(queryReq.Url) {
					return nil
				}
			}
		}
	}
}
