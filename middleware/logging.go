package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// LoggingMiddleware logs each incoming HTTP request using slog
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the ResponseWriter to capture the status code
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rw, r)

		// Use the globally configured slog logger
		slog.Info("HTTP Request",
			"method", r.Method,
			"path", r.URL.Path,
			"remoteAddr", r.RemoteAddr,
			"userAgent", r.UserAgent(),
			"status", fmt.Sprintf("%d %s", rw.status, http.StatusText(rw.status)),
			"duration", time.Since(start),
		)
	})
}

// statusRecorder wraps http.ResponseWriter to record the status code written.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
