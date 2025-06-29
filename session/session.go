package session

import (
	"net/http"

	"github.com/gorilla/sessions"
	"monolith/config"
)

const SESSION_NAME_KEY = "session"
const LOGGED_IN_KEY = "logged_in"
const EMAIL_KEY = "email"

var store *sessions.CookieStore

// InitStore initializes the session store with the secret key
func InitStore() {
	store = sessions.NewCookieStore([]byte(config.SECRET_KEY))
}

// GetSession retrieves the session from the request
func GetSession(r *http.Request) (*sessions.Session, error) {
	return store.Get(r, SESSION_NAME_KEY)
}

func SetLoggedIn(w http.ResponseWriter, r *http.Request, email string) {
	session, _ := GetSession(r)
	session.Options = &sessions.Options{MaxAge: 7 * 24 * 60 * 60, SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil}
	session.Values[LOGGED_IN_KEY] = true
	session.Values[EMAIL_KEY] = email
	session.Save(r, w)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := GetSession(r)
	delete(session.Values, LOGGED_IN_KEY)
	delete(session.Values, EMAIL_KEY)
	session.Save(r, w)
}

func IsLoggedIn(r *http.Request) bool {
	session, _ := GetSession(r)
	loggedIn, ok := session.Values[LOGGED_IN_KEY].(bool)
	return ok && loggedIn
}
