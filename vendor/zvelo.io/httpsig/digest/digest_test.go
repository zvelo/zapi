package digest

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func TestDigest(t *testing.T) {
	t.Parallel()

	for _, s := range []struct {
		algo   Algorithm
		body   string
		expect string
		error  string
	}{
		{ADLER32, "Wiki", "03da0195", ""},
		{CRC32c, "dog", "0a72a4df", ""},
		{MD5, "abc", "kAFQmDzST7DWlj99KOF_cg", ""},
		{SHA, "abc", "qZk-NkcGgWq6PiVxeFDCbJzQ2J0", ""},
		{SHA256, "abc", "ungWv48Bz-pBQUDeXa4iI7ADYaOWF3qctBD_YfIAFa0", ""},
		{SHA512, "abc", "3a81oZNherrMQXNJriBBMRLm-k6JqX6iCp7u5ktV05ohkpkqJ0_BqDa6PCOj_uu9RU1EI2Q86A4qmslPpUyknw", ""},
		{UNIXsum, "abc", "", "UNIXsum is unsupported"},
		{UNIXcksum, "abc", "", "UNIXcksum is unsupported"},
	} {
		t.Run(s.algo.String(), func(t *testing.T) {
			req, err := http.NewRequest("POST", "http://example.com", strings.NewReader(s.body))
			if err != nil {
				t.Fatal(err)
			}

			hash, err := s.algo.Hash(req)
			if err != nil {
				if s.error != "" {
					if s.error != err.Error() {
						t.Error("unexpected error", err.Error())
					}
					return
				}

				t.Fatal(err)
			}

			enc, err := s.algo.codec()
			if err != nil {
				t.Fatal(err)
			}

			buf := make([]byte, enc.EncodedLen(len(hash)))
			enc.Encode(buf, hash)

			if string(buf) != s.expect {
				t.Error("unexpected result", string(buf))
			}

			if err = s.algo.SetHeader(req); err != nil {
				t.Fatal(err)
			}

			if h := req.Header.Get(Header); h == "" {
				t.Error("header wasn't set")
			}

			algo, hash2, err := ParseHeader(req)
			if err != nil {
				t.Fatal(err)
			}

			if algo != s.algo {
				t.Error("wrong algorithm")
			}

			if !bytes.Equal(hash, hash2) {
				t.Error("wrong hash")
			}

			if err = algo.Verify(req, hash2); err != nil {
				t.Error(err)
			}

			if err = VerifyHeader(req); err != nil {
				t.Error(err)
			}
		})
	}
}
