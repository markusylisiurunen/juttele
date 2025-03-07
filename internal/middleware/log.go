package middleware

import (
	"net/http"

	"github.com/felixge/httpsnoop"
	"github.com/markusylisiurunen/juttele/internal/logger"
)

func Log() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m := httpsnoop.CaptureMetrics(next, w, r)
			keysAndValues := []any{
				"status", m.Code,
				"method", r.Method,
				"path", r.URL.Path,
				"duration", m.Duration.Milliseconds(),
			}
			if m.Code >= 400 {
				logger.Get().Error("an outgoing response", keysAndValues...)
			} else {
				logger.Get().Debug("an outgoing response", keysAndValues...)
			}
		})
	}
}
