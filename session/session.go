package session

import (
	"net/http"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Session store
var store = sessions.NewCookieStore([]byte("super-secret-key"))

// Google OAuth2 Config
var googleOAuthConfig = &oauth2.Config{
	ClientID:     "YOUR_GOOGLE_CLIENT_ID",     // Replace
	ClientSecret: "YOUR_GOOGLE_CLIENT_SECRET", // Replace
	RedirectURL:  "http://localhost:8080/auth/google/callback",
	Scopes:       []string{"profile", "email"},
	Endpoint:     google.Endpoint,
}

func GetGoogleOAuthConfig() *oauth2.Config {
	return googleOAuthConfig
}

func SetLoggedIn(w http.ResponseWriter, r *http.Request, email string) {
	session, _ := store.Get(r, "session")
	session.Values["logged_in"] = true
	session.Values["email"] = email
	session.Save(r, w)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	delete(session.Values, "logged_in")
	delete(session.Values, "email")
	session.Save(r, w)
}

func IsLoggedIn(r *http.Request) bool {
	session, _ := store.Get(r, "session")
	loggedIn, ok := session.Values["logged_in"].(bool)
	return ok && loggedIn
}
