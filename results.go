package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/template"

	"github.com/fatih/color"
	"google.golang.org/grpc/codes"
	"zvelo.io/msg"
)

type queryResult struct {
	result  *msg.QueryResult
	traceID string
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
		return urlFromReqID(reqID, "<UNKNOWN>")
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
		return fmt.Sprintf("%s(%d)", codes.Code(i), i)
	},
}).Parse(queryResultTplStr))

func isComplete(result *msg.QueryResult) bool {
	return result != nil && result.QueryStatus != nil && result.QueryStatus.Complete
}

func resultHandler() {
	for result := range resultCh {
		fmt.Fprintf(os.Stderr, "\nreceived result\n")

		color.Set(color.FgCyan)

		if traceID := result.traceID; traceID != "" {
			fmt.Fprintf(os.Stderr, "Trace ID:           %s\n", traceID[:strings.Index(traceID, ":")])
		}

		if err := queryResultTpl.ExecuteTemplate(os.Stdout, "QueryResult", result.result); err != nil {
			color.Unset()
			panic(err)
		}

		color.Unset()

		if isComplete(result.result) {
			resultWg.Done()
		}
	}
}
