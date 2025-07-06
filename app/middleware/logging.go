package middleware

import (
	"fmt"
	"log/slog"
	"net"
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
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		slog.Info("HTTP Request",
			"method", r.Method,
			"path", r.URL.Path,
			"ip", ip,
			"userAgent", r.UserAgent(),
			"status", fmt.Sprintf("%d %s", rw.status, http.StatusText(rw.status)),
			"duration", time.Since(start).Milliseconds(),
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
