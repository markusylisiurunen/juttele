package middleware

import (
	"net/http"
	"strings"

	"github.com/markusylisiurunen/juttele/internal/repo"
)

func Auth(repo *repo.Repository, token string) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				apiKey := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
				if apiKey == token {
					next.ServeHTTP(w, r)
					return
				}
				knownKeys, err := repo.ListAPIKeys(r.Context())
				if err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				for _, key := range knownKeys.Items {
					if apiKey == key.UUID {
						next.ServeHTTP(w, r)
						return
					}
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
