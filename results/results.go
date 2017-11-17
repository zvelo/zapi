package results

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/fatih/color"

	"google.golang.org/grpc/codes"

	"zvelo.io/msg"
	"zvelo.io/zapi/internal/zvelo"
)

type Result struct {
	*msg.QueryResult
	URL         string
	PollTraceID string
	PollStart   time.Time
	Start       time.Time
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
{{- if . -}}
{{if .FetchCode}}Fetch Status:       {{httpStatus .FetchCode}}
{{end}}
{{- if .Location}}Redirect Location:  {{.Location}}
{{end}}
{{- if .Error}}{{template "Status" .Error}}{{end}}
{{- end}}
{{- end}}

{{define "QueryResult" -}}
{{- if url .}}URL/Content:        {{url .}}
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
	"url": func(result Result) string {
		if result.URL != "" {
			return result.URL
		}
		return "<UNKNOWN>"
	},
	"complete": func(result Result) string {
		if !zvelo.IsComplete(result.QueryResult) {
			return "false"
		}

		if result.Start != (time.Time{}) {
			return time.Since(result.Start).String()
		}

		return "false"
	},
	"poll": func(result Result) string {
		if result.PollStart != (time.Time{}) {
			return time.Since(result.PollStart).String()
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

func Print(result Result) {
	fmt.Fprintf(os.Stderr, "\nreceived result\n")

	if traceID := result.PollTraceID; traceID != "" {
		printf := zvelo.PrintfFunc(color.FgCyan, os.Stderr)
		printf("Trace ID:           %s\n", zvelo.TraceIDString(traceID))
	}

	var buf bytes.Buffer
	if err := queryResultTpl.ExecuteTemplate(&buf, "QueryResult", result); err != nil {
		zvelo.Errorf("%s\n", err)
	}

	printf := zvelo.PrintfFunc(color.FgCyan, os.Stdout)
	printf(buf.String())
}
