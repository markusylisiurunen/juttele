package middleware

import (
	"net/http"
	"strings"
)

func Auth(token string) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				apiKey := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
				if apiKey == token {
					next.ServeHTTP(w, r)
					return
				}
			}
			apiKey := r.URL.Query().Get("api_key")
			if apiKey != "" && apiKey == token {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		})
	}
}
