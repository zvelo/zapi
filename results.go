package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/fatih/color"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"zvelo.io/msg"
)

type queryResult struct {
	*msg.QueryResult
	pollTraceID string
	pollStart   time.Time
}

var (
	resultWg sync.WaitGroup
	resultCh = make(chan queryResult)
)

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
{{- if poll .}}Poll Duration:      {{poll .}}
{{end}}
{{- if complete .}}Complete:           {{complete .}}
{{end}}
{{- if .ResponseDataset}}{{template "DataSet" .ResponseDataset}}{{end}}
{{- template "QueryStatus" .QueryStatus}}
{{- end}}`

var queryResultTpl = template.Must(template.New("QueryResult").Funcs(template.FuncMap{
	"url": func(reqID string) string {
		return getReqIDData(reqID, "<UNKNOWN>").url
	},
	"complete": func(result queryResult) string {
		if !isComplete(result.QueryResult) {
			return "false"
		}

		if data := getReqIDData(result.RequestId, ""); data.start != (time.Time{}) {
			return time.Since(data.start).String()
		}

		return "false"
	},
	"poll": func(result queryResult) string {
		if result.pollStart != (time.Time{}) {
			return time.Since(result.pollStart).String()
		}
		return ""
	},
	"malicious": func(m *msg.DataSet_Malicious) string {
		if m.Verdict == msg.VERDICT_MALICIOUS {
			return m.Category.String()
		}

		return m.Verdict.String()
	},
	"httpStatus": func(i int32) string {
		return fmt.Sprintf("%s (%d)", http.StatusText(int(i)), i)
	},
	"errorcode": func(i int32) string {
		return fmt.Sprintf("%s (%d)", codes.Code(i), i)
	},
}).Parse(queryResultTplStr))

func isComplete(result *msg.QueryResult) bool {
	if result == nil || result.QueryStatus == nil {
		return false
	}

	if status.ErrorProto(result.QueryStatus.Error) != nil {
		// there was an error, complete is implied
		return true
	}

	return result.QueryStatus.Complete
}

func resultHandler() {
	for result := range resultCh {
		fmt.Fprintf(os.Stderr, "\nreceived result\n")

		if traceID := result.pollTraceID; traceID != "" {
			printf := printfFunc(color.FgCyan, os.Stderr)
			printf("Trace ID:           %s\n", traceID[:strings.Index(traceID, ":")])
		}

		var buf bytes.Buffer
		if err := queryResultTpl.ExecuteTemplate(&buf, "QueryResult", result); err != nil {
			errorf("%s\n", err)
		}

		printf := printfFunc(color.FgCyan, os.Stdout)
		printf(buf.String())

		if isComplete(result.QueryResult) {
			resultWg.Done()
		}
	}
}
