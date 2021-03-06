package results

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"text/template"

	"github.com/fatih/color"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/segmentio/ksuid"

	"google.golang.org/grpc/codes"

	msg "zvelo.io/msg/msgpb"
	"zvelo.io/zapi/internal/zvelo"
)

var jsonMarshaler = jsonpb.Marshaler{OrigName: true}

var queryResultTplStr = `
{{define "DataSet" -}}
{{- if .Categorization -}}
Categories:         {{range .Categorization.Value}}{{.}} {{end}}
{{if .Categorization.Error}}{{template "Error" .Categorization.Error}}{{end}}
{{- end}}

{{- if .Malicious -}}
Malicious:          {{if .Malicious.Category}}{{range .Malicious.Category}}{{.}} {{end}}{{else}}CLEAN{{end}}
{{if .Malicious.Error}}{{template "Error" .Malicious.Error}}{{end}}
{{- end}}

{{- if .Echo}}Echo:               {{.Echo.Url}}
{{if .Echo.Error}}{{template "Error" .Echo.Error}}{{end}}
{{end}}
{{- end}}

{{define "Error" -}}
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
{{- if .Error}}{{template "Error" .Error}}{{end}}
{{- end}}
{{- end}}

{{define "QueryResult" -}}
{{- if .Url}}URL/Content:        {{.Url}}
{{end}}
{{- if .RequestId}}Request ID:         {{.RequestId}}
{{end}}
{{- if complete .}}Complete:           {{complete .}}
{{end}}
{{- if .ResponseDataset}}{{template "DataSet" .ResponseDataset}}{{end}}
{{- template "QueryStatus" .QueryStatus}}
{{- end}}`

var queryResultTpl = template.Must(template.New("QueryResult").Funcs(template.FuncMap{
	"complete": func(result *msg.QueryResult) string {
		if !zvelo.IsComplete(result) {
			return "false"
		}

		return ""
	},
	"httpStatus": func(i int32) string {
		return fmt.Sprintf("%s (%d)", http.StatusText(int(i)), i)
	},
	"errorcode": func(i int32) string {
		return fmt.Sprintf("%s (%d)", codes.Code(i), i)
	},
}).Parse(queryResultTplStr))

func Print(result *msg.QueryResult, json bool) {
	fmt.Fprintf(os.Stderr, "\nreceived result\n")

	if json {
		if err := jsonMarshaler.Marshal(os.Stdout, result); err != nil {
			zvelo.Errorf("marshal error: %s\n", err)
		}
		fmt.Fprintln(os.Stdout)
		return
	}

	var buf bytes.Buffer
	if err := queryResultTpl.ExecuteTemplate(&buf, "QueryResult", result); err != nil {
		zvelo.Errorf("%s\n", err)
	}

	printf := zvelo.PrintfFunc(color.FgCyan, os.Stdout)
	printf(buf.String())
}

func TracingTag() ksuid.KSUID {
	id := ksuid.New()
	printf := zvelo.PrintfFunc(color.FgCyan, os.Stdout)
	printf("Tracing Tag: guid:x-client-trace-id=%s\n", id)
	return id
}
