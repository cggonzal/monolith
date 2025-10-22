package middleware

import (
	"log/slog"
	"net/http"
)

var crossOriginProtector = newCrossOriginProtector()

func newCrossOriginProtector() *http.CrossOriginProtection {
	protector := http.NewCrossOriginProtection()
	protector.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Warn("request blocked by cross-origin protection",
			"method", r.Method,
			"origin", r.Header.Get("Origin"),
			"sec_fetch_site", r.Header.Get("Sec-Fetch-Site"),
		)
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	}))
	return protector
}

// CrossOriginProtector exposes the shared CrossOriginProtection instance so applications can
// register trusted origins or customize the deny handler during startup.
func CrossOriginProtector() *http.CrossOriginProtection {
	return crossOriginProtector
}

// CSRFMiddleware applies Go 1.25's built-in cross-origin request protection.
func CSRFMiddleware(next http.Handler) http.Handler {
	return crossOriginProtector.Handler(next)
}
