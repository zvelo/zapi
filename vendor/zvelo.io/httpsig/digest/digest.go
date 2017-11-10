package digest

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/adler32"
	"hash/crc32"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// https://tools.ietf.org/html/rfc3230
// https://www.iana.org/assignments/http-dig-alg/http-dig-alg.xhtml

func copyBody(w io.Writer, req *http.Request) error {
	// if there is no GetBody method, then we'll create one

	if req.GetBody == nil {
		var body bytes.Buffer

		// ensure that the copy, later, also writes to our body buffer
		w = io.MultiWriter(w, &body)

		req.GetBody = func() (io.ReadCloser, error) {
			return ioutil.NopCloser(&body), nil
		}
	}

	var err error

	if _, err = io.Copy(w, req.Body); err != nil {
		return err
	}

	// close the old body

	if err = req.Body.Close(); err != nil {
		return err
	}

	// set the new body

	if req.Body, err = req.GetBody(); err != nil {
		return err
	}

	return nil
}

// Algorithm represents the type of HTTP Digest to use
type Algorithm int

// These are the available Algorithms. Note that UNIXsum and UNIXcksum are not
// fully supported.
const (
	Unknown Algorithm = iota
	ADLER32
	CRC32c
	MD5
	SHA
	SHA256
	SHA512
	UNIXsum
	UNIXcksum
)

func (a Algorithm) String() string {
	switch a {
	case ADLER32:
		return "ADLER32"
	case CRC32c:
		return "CRC32c"
	case MD5:
		return "MD5"
	case SHA:
		return "SHA"
	case SHA256:
		return "SHA-256"
	case SHA512:
		return "SHA-512"
	case UNIXsum:
		return "UNIXsum"
	case UNIXcksum:
		return "UNIXcksum"
	}
	return "Algorithm(" + strconv.Itoa(int(a)) + ")"
}

type codec interface {
	Encode(dst, src []byte)
	EncodedLen(n int) int
	Decode(dst, src []byte) (int, error)
	DecodedLen(n int) int
}

type hexCodec struct{}

func (e hexCodec) Encode(dst, src []byte) {
	hex.Encode(dst, src)
}

func (e hexCodec) EncodedLen(n int) int {
	return hex.EncodedLen(n)
}

func (e hexCodec) Decode(dst, src []byte) (int, error) {
	return hex.Decode(dst, src)
}

func (e hexCodec) DecodedLen(n int) int {
	return hex.DecodedLen(n)
}

var hexEncoding codec = hexCodec{}

func (a Algorithm) codec() (codec, error) {
	switch a {
	case ADLER32, CRC32c:
		return hexEncoding, nil
	case MD5, SHA, SHA256, SHA512:
		return base64.RawURLEncoding, nil
	case UNIXsum, UNIXcksum:
		// unsupported
	}

	return nil, errors.Errorf("%s is unsupported", a)
}

// Hash returns a hash for the body of the request
func (a Algorithm) Hash(req *http.Request) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}

	var h hash.Hash

	switch a {
	case ADLER32:
		h = adler32.New()
	case CRC32c:
		h = crc32.New(crc32.MakeTable(crc32.Castagnoli))
	case MD5:
		h = md5.New()
	case SHA:
		h = sha1.New()
	case SHA256:
		h = sha256.New()
	case SHA512:
		h = sha512.New()
	case UNIXsum, UNIXcksum:
		return nil, errors.Errorf("%s is unsupported", a)
	}

	if err := copyBody(h, req); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

// Verify the hash of the body of the request
func (a Algorithm) Verify(req *http.Request, hash []byte) error {
	h, err := a.Hash(req)
	if err != nil {
		return err
	}

	if !bytes.Equal(h, hash) {
		return errors.New("invalid hash")
	}

	return nil
}

// VerifyHeader parses the HTTP Digest header and verifies it against the body
// of the request
func VerifyHeader(req *http.Request) error {
	algo, hash, err := ParseHeader(req)
	if err != nil {
		return err
	}

	return algo.Verify(req, hash)
}

// Header is the HTTP header used
const Header = "Digest"

// SetHeader sets the HTTP Digest header to the hash of the body of the request
func (a Algorithm) SetHeader(req *http.Request) error {
	hash, err := a.Hash(req)
	if err != nil {
		return err
	}

	if hash == nil {
		req.Header.Del(Header)
		return nil
	}

	enc, err := a.codec()
	if err != nil {
		return err
	}

	if enc != nil {
		buf := make([]byte, enc.EncodedLen(len(hash)))
		enc.Encode(buf, hash)
		hash = buf
	}

	req.Header.Set(Header, fmt.Sprintf("%s=%s", a, string(hash)))

	return nil
}

// ParseHeader parses the HTTP Digest header and extracts the Algorithm and hash
// of the request body
func ParseHeader(req *http.Request) (Algorithm, []byte, error) {
	h := req.Header.Get(Header)
	i := strings.Index(h, "=")
	if i == -1 || i >= len(h)-1 {
		return Unknown, nil, errors.Errorf("invalid %s header", Header)
	}

	name := h[:i]
	value := []byte(h[i+1:])

	algo := ParseAlgorithm(name)

	dec, err := algo.codec()
	if err != nil {
		return Unknown, nil, err
	}

	if dec == nil {
		return algo, value, nil
	}

	buf := make([]byte, dec.DecodedLen(len(value)))
	if _, err = dec.Decode(buf, value); err != nil {
		return Unknown, nil, err
	}

	return algo, buf, nil
}

// ParseAlgorithm parses a string into an Algorithm
func ParseAlgorithm(val string) Algorithm {
	switch val {
	case ADLER32.String():
		return ADLER32
	case CRC32c.String():
		return CRC32c
	case MD5.String():
		return MD5
	case SHA.String():
		return SHA
	case SHA256.String():
		return SHA256
	case SHA512.String():
		return SHA512
	case UNIXsum.String():
		return UNIXsum
	case UNIXcksum.String():
		return UNIXcksum
	}
	return Unknown
}
