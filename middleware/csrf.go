package middleware

import (
	"log/slog"
	"net/http"
)

const csrfCookieName = "csrf_token"

// CSRFMiddleware validates the CSRF token on mutating requests.
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		if err := r.ParseForm(); err != nil {
			slog.Error("csrf parse form", "error", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		formToken := r.FormValue("csrf_token")
		cookie, err := r.Cookie(csrfCookieName)
		if err != nil {
			slog.Warn("CSRF token missing in cookie")
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		if formToken == "" {
			slog.Warn("CSRF token missing in form")
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		if cookie.Value != formToken {
			slog.Warn("CSRF token mismatch")
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
