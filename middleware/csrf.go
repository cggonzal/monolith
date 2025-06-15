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

		reqToken := r.Header.Get("X-CSRF-Token")
		if reqToken == "" {
			if err := r.ParseForm(); err != nil {
				slog.Error("csrf parse form", "error", err)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			reqToken = r.FormValue("csrf_token")
		}
		cookie, err := r.Cookie(csrfCookieName)
		if err != nil {
			slog.Warn("CSRF token missing in cookie")
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		if reqToken == "" {
			slog.Warn("CSRF token missing in request")
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		if cookie.Value != reqToken {
			slog.Warn("CSRF token mismatch")
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
