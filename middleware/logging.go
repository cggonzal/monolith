package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// LoggingMiddleware logs each incoming HTTP request using slog
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)

		// Use the globally configured slog logger
		slog.Info("HTTP Request",
			"method", r.Method,
			"path", r.URL.Path,
			"remoteAddr", r.RemoteAddr,
			"userAgent", r.UserAgent(),
			"duration", time.Since(start),
		)
	})
}
