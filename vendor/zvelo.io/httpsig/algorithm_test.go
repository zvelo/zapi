package httpsig

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

var (
	ecdsaKey *ecdsa.PrivateKey
	rsaKey   *rsa.PrivateKey
	hmacKey  = []byte("9449e221334aca9fd237ba3caeb1a00e")
)

func init() {
	var err error

	if ecdsaKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader); err != nil {
		panic(err)
	}

	if rsaKey, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
		panic(err)
	}
}

func testKeyGetter() KeyGetter {
	return KeyGetterFunc(func(keyID string) (interface{}, error) {
		switch keyID {
		case "rsa-key-1":
			return rsaKey, nil
		case "ecdsa-key-1":
			return ecdsaKey, nil
		case "hmac-key-1":
			return hmacKey, nil
		}
		return nil, errors.New("unknown key id")
	})
}

func TestSign(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(Middleware(SignatureHeader, testKeyGetter(), nil))

	for _, s := range []struct {
		algo  Algorithm
		key   interface{}
		keyID string
	}{{
		algo:  RSASHA1,
		key:   rsaKey,
		keyID: "rsa-key-1",
	}, {
		algo:  RSASHA256,
		key:   rsaKey,
		keyID: "rsa-key-1",
	}, {
		algo:  HMACSHA256,
		key:   hmacKey,
		keyID: "hmac-key-1",
	}, {
		algo:  ECDSASHA256,
		key:   ecdsaKey,
		keyID: "ecdsa-key-1",
	}} {
		for _, u := range []struct {
			method  string
			body    string
			getBody bool
			headers map[string]string
		}{{
			method:  "POST",
			body:    `{"hello": "world"}`,
			getBody: true,
			headers: map[string]string{
				"Content-Length": "this shouldn't be set",
			},
		}, {
			method: "POST",
			body:   "some other body",
			headers: map[string]string{
				"Host": "this shouldn't be set",
			},
		}, {
			method: "GET",
			headers: map[string]string{
				"User-Agent": "test ua",
			},
		}, {
			method: "DELETE",
			headers: map[string]string{
				"Transfer-Encoding": "this shouldn't be set",
			},
		}, {
			method: "HEAD",
			headers: map[string]string{
				"Trailer": "this shouldn't be set",
			},
		}} {
			name := fmt.Sprintf("%s/%s", s.algo, u.method)
			var body io.Reader
			if u.body != "" {
				name += "/" + u.body
				body = strings.NewReader(u.body)
			}

			t.Run(name, func(t *testing.T) {
				req, err := http.NewRequest(u.method, ts.URL, body)
				if err != nil {
					t.Fatal(err)
				}

				for k, v := range u.headers {
					req.Header.Set(k, v)
				}

				if !u.getBody {
					req.GetBody = nil
				}

				client := http.Client{
					Transport: s.algo.Transport(s.keyID, s.key),
				}

				if _, err = client.Do(req); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
