package httpsig

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestHeader(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(AuthorizationHeader.String(), `Signature keyId="hmac-key-1",algorithm="hmac-sha256",headers="(request-target) host date digest content-length",signature="YWJj"`)
	h, err := AuthorizationHeader.Parse(req)
	if err != nil {
		t.Fatal(err)
	}

	expect := Header{
		KeyID:     "hmac-key-1",
		Algorithm: HMACSHA256,
		Headers:   []string{"(request-target)", "host", "date", "digest", "content-length"},
		Signature: []byte("abc"),
	}

	if !cmp.Equal(h, &expect) {
		t.Log(cmp.Diff(h, &expect))
		t.Error("Parse returned unexpected result")
	}
}
