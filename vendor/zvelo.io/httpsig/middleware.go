package httpsig

import "net/http"

// Middleware is an HTTP middleware that will call next only if the request's
// HTTP signature is valid
func Middleware(t HeaderType, getter KeyGetter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := t.Verify(getter, r); err != nil {
			code := http.StatusBadRequest
			if t == AuthorizationHeader {
				code = http.StatusUnauthorized
			}
			http.Error(w, err.Error(), code)
			return
		}

		if next != nil {
			next.ServeHTTP(w, r)
		}
	})
}
