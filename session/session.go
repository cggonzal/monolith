/*
Package session handles cookie-based sessions for user authentication.
*/
package session

import (
	"net/http"

	"github.com/gorilla/sessions"
)

// session keys
const SESSION_NAME_KEY = "session"
const LOGGED_IN_KEY = "logged_in"
const EMAIL_KEY = "email"

// Session store
var store = sessions.NewCookieStore([]byte("super-secret-key"))

func SetLoggedIn(w http.ResponseWriter, r *http.Request, email string) {
	session, _ := store.Get(r, SESSION_NAME_KEY)
	session.Values[LOGGED_IN_KEY] = true
	session.Values[EMAIL_KEY] = email
	session.Save(r, w)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, SESSION_NAME_KEY)
	delete(session.Values, LOGGED_IN_KEY)
	delete(session.Values, EMAIL_KEY)
	session.Save(r, w)
}

func IsLoggedIn(r *http.Request) bool {
	session, _ := store.Get(r, SESSION_NAME_KEY)
	loggedIn, ok := session.Values[LOGGED_IN_KEY].(bool)
	return ok && loggedIn
}
