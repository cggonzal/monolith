package session

import (
	"monolith/app/config"
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
