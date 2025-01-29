package middleware

import (
	"crudapp/session"
	"net/http"
)

// RequireLogin ensures the user is logged in before accessing a route
func RequireLogin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !session.IsLoggedIn(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}
