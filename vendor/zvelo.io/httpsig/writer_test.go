package httpsig

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWriter(t *testing.T) {
	var w writer

	w.write("none")
	w.write("bad", " ")
	w.write("(request-target)", "get /foo")
	w.write("Host", "example.org", " ")
	w.write("Date", "Tue, 07 Jun 2014 20:51:35 GMT")
	w.write("Cache-Control", "max-age=60", "must-revalidate")
	w.write("X-Example", "Example header\nwith some whitespace. ")

	sig := w.Bytes()

	const expectSig = `(request-target): get /foo
host: example.org
date: Tue, 07 Jun 2014 20:51:35 GMT
cache-control: max-age=60, must-revalidate
x-example: Example header with some whitespace.`

	if !cmp.Equal(string(sig), expectSig) {
		t.Log(cmp.Diff(string(sig), expectSig))
		t.Error("unexpected signature")
	}

	const expectHeaders = "(request-target) host date cache-control x-example"

	if strings.Join(*w.headers, " ") != expectHeaders {
		t.Log(cmp.Diff(*w.headers, expectHeaders))
		t.Error("unexpected headers")
	}
}
