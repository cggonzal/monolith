package session

import (
	"monolith/config"
	"net/http"

	"github.com/gorilla/sessions"
)

const SESSION_NAME_KEY = "session"
const LOGGED_IN_KEY = "logged_in"
const EMAIL_KEY = "email"

var store *sessions.CookieStore

// InitStore initializes the session store with the secret key
func InitSession() {
	store = sessions.NewCookieStore([]byte(config.SECRET_KEY))
}

// GetSession retrieves the session from the request
func GetSession(r *http.Request) (*sessions.Session, error) {
	return store.Get(r, SESSION_NAME_KEY)
}

// SetLoggedIn marks the session as logged in and stores the email
func SetLoggedIn(w http.ResponseWriter, r *http.Request, email string) {
	session, _ := GetSession(r)
	session.Options = &sessions.Options{MaxAge: 7 * 24 * 60 * 60, SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil}
	session.Values[LOGGED_IN_KEY] = true
	session.Values[EMAIL_KEY] = email
	session.Save(r, w)
}

// Logout clears login related session values
func Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := GetSession(r)
	delete(session.Values, LOGGED_IN_KEY)
	delete(session.Values, EMAIL_KEY)
	session.Save(r, w)
}

// IsLoggedIn checks if the request is associated with a logged in session
func IsLoggedIn(r *http.Request) bool {
	session, _ := GetSession(r)
	loggedIn, ok := session.Values[LOGGED_IN_KEY].(bool)
	return ok && loggedIn
}
