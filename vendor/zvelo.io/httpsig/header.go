package httpsig

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/pkg/errors"

	"zvelo.io/httpsig/digest"
)

// HeaderType dictates which HTTP header to use and the format of the value for
// generating and verifying HTTP signatures
type HeaderType string

// These are the available HeaderTypes to use
const (
	SignatureHeader     HeaderType = "Signature"
	AuthorizationHeader HeaderType = "Authorization"
)

func (t HeaderType) String() string {
	return string(t)
}

// A KeyGetter gets a key for HTTP signature verification based on the keyID
type KeyGetter interface {
	GetKey(keyID string) (key interface{}, err error)
}

// The KeyGetterFunc type is an adapter to allow the use of ordinary functions
// as KeyGetters. If f is a function with the appropriate signature,
// KeyGetterFunc(f) is a KeyGetter that calls f.
type KeyGetterFunc func(keyID string) (key interface{}, err error)

// GetKey returns f(keyID)
func (f KeyGetterFunc) GetKey(keyID string) (key interface{}, err error) {
	return f(keyID)
}

var _ KeyGetter = (*KeyGetterFunc)(nil)

// A Header represents the parts of the HTTP signature header
type Header struct {
	KeyID     string
	Algorithm Algorithm
	Headers   []string
	Signature []byte
}

func (h Header) String() string {
	return fmt.Sprintf(
		`keyId="%s",algorithm="%s",headers="%s",signature="%s"`,
		h.KeyID,
		h.Algorithm,
		strings.Join(h.Headers, " "),
		base64.RawURLEncoding.EncodeToString(h.Signature),
	)
}

// Verify ensures that the Header is valid
func (h Header) Verify(getter KeyGetter, req *http.Request) error {
	// if there is a digest header in the signature, make sure it's valid
	for _, n := range h.Headers {
		if strings.EqualFold(n, digest.Header) {
			if err := digest.VerifyHeader(req); err != nil {
				return err
			}
			break
		}
	}

	key, err := getter.GetKey(h.KeyID)
	if err != nil {
		return err
	}

	return h.Algorithm.Verify(
		key,
		h.SignatureString(req, h.Headers...),
		h.Signature,
	)
}

func extractParam(val, name string) (string, bool) {
	if strings.HasPrefix(val, name+`="`) {
		// looking for param at the beginning of val, e.g. `keyId="`
		if len(val) == len(name)+2 {
			return "", false
		}

		val = val[len(name)+2:]
	} else {
		// looking for param in middle of val, e.g. `",algorithm="`
		i := strings.LastIndex(val, `",`+name+`="`)
		if i == -1 || i+1 == len(name)+4 {
			return "", false
		}

		val = val[i+len(name)+4:]
	}

	if val == "" {
		return "", false
	}

	// looking for first occurrence of `",` to designate end of val
	if i := strings.Index(val, `",`); i != -1 {
		return val[:i], true
	}

	// looking to see if string ends with `"`
	if i := len(val) - 1; val[i] == '"' {
		return val[:i], true
	}

	return "", false
}

// Parse returns a Header populated with the fields extracted from the HTTP
// signature
func (t HeaderType) Parse(req *http.Request) (*Header, error) {
	val := req.Header.Get(t.String())
	if val == "" {
		return nil, errors.Errorf("%s header not found", t)
	}

	if t == AuthorizationHeader {
		if !strings.HasPrefix(val, "Signature ") {
			return nil, errors.New("invalid Signature header")
		}

		val = val[len("Signature "):]
	}

	h := Header{
		// the headers field is optional
		// if it doesn't exist implementations MUST operate as if the field were
		// specified with a single value, the `Date` header
		Headers: []string{"date"},
	}
	var ok bool

	if h.KeyID, ok = extractParam(val, "keyId"); !ok {
		return nil, errors.Errorf("%s header missing keyId field", t)
	}

	algo, ok := extractParam(val, "algorithm")
	if !ok {
		return nil, errors.Errorf("%s header missing algorithm field", t)
	}

	h.Algorithm = ParseAlgorithm(algo)

	headers, ok := extractParam(val, "headers")
	if ok { // remember, this field is optional
		h.Headers = strings.Fields(headers)
	}

	sig, ok := extractParam(val, "signature")
	if !ok {
		return nil, errors.Errorf("%s header missing signature field", t)
	}

	var err error
	if h.Signature, err = base64.RawURLEncoding.DecodeString(sig); err != nil {
		return nil, err
	}

	return &h, nil
}

// ValidKeyID returns true if the keyID doesn't contain any quote (`"`) or comma
// (`,`) characters
func ValidKeyID(keyID string) bool {
	return !strings.ContainsAny(keyID, `",`)
}

// Set sets the appropriate HTTP header to make an HTTP signature on the request
func (t HeaderType) Set(a Algorithm, keyID string, key interface{}, req *http.Request, digestAlgo digest.Algorithm) error {
	if !ValidKeyID(keyID) {
		return errors.New("invalid key id")
	}

	h := Header{
		KeyID:     keyID,
		Algorithm: a,
	}

	var err error
	if err = setRequiredHeaders(req, digestAlgo); err != nil {
		return err
	}

	if h.Signature, err = a.Sign(key, h.SignatureString(req)); err != nil {
		return err
	}

	if t == AuthorizationHeader {
		req.Header.Set(t.String(), "Signature "+h.String())
	} else {
		req.Header.Set(t.String(), h.String())
	}

	return nil
}

// Verify ensures that the HTTP signature on the request is valid
func (t HeaderType) Verify(getter KeyGetter, req *http.Request) error {
	h, err := t.Parse(req)
	if err != nil {
		return err
	}

	return h.Verify(getter, req)
}

func setRequiredHeaders(req *http.Request, digestAlgo digest.Algorithm) error {
	if val := req.Header.Get("Date"); val == "" {
		req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	}

	// don't clobber an existing Digest header, but return an error if it is
	// invalid
	if val := req.Header.Get(digest.Header); val != "" {
		return digest.VerifyHeader(req)
	}

	// there was no Digest header, so let's set one
	return digestAlgo.SetHeader(req)
}

// SignatureString returns the string that should be signed for the request. It
// also populates Header.Headers. If headers has values, only those will be used
// to produce the string. Otherwise, all headers except Host, User-Agent,
// Content-Length, Transfer-Encoding and Trailer will be used.
func (h *Header) SignatureString(req *http.Request, headers ...string) []byte {
	w := writer{headers: &h.Headers}

	w.write("(request-target)", strings.ToLower(req.Method)+" "+req.URL.RequestURI())
	w.write("host", requestHost(req))

	if len(headers) > 0 {
		for _, n := range headers {
			w.write(n, req.Header[textproto.CanonicalMIMEHeaderKey(n)]...)
		}
	} else {
		for _, kv := range sortedKeyValues(req.Header, reqWriteExcludeHeader) {
			w.write(kv.key, kv.values...)
		}
	}

	return w.Bytes()
}
