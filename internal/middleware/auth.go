package middleware

import (
	"net/http"
	"strings"
)

func Auth(token string) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			_token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if _token != token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
