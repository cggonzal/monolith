package csrf

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

const tokenLength = 32
const cookieName = "csrf_token"

// generateToken creates a random token string.
func generateToken() string {
	b := make([]byte, tokenLength)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.RawStdEncoding.EncodeToString(b)
}

// ensureToken returns the CSRF token stored in the cookie, creating a new one
// if necessary.
func ensureToken(w http.ResponseWriter, r *http.Request) string {
	c, err := r.Cookie(cookieName)
	if err == nil && c.Value != "" {
		return c.Value
	}

	token := generateToken()
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
	return token
}

// GetCSRFToken returns the raw CSRF token, ensuring a cookie is set.
func GetCSRFToken(w http.ResponseWriter, r *http.Request) string {
	return ensureToken(w, r)
}

// GetCSRFMetaTag returns a meta tag containing the CSRF token. This can be
// included in templates so JavaScript can read the token for AJAX requests.
func GetCSRFMetaTag(w http.ResponseWriter, r *http.Request) string {
	token := ensureToken(w, r)
	return "<meta name=\"csrf-token\" content=\"" + token + "\">"
}

// GetCSRFTokenForForm ensures a CSRF token cookie exists and returns a hidden
// input element containing the token. Pass the returned string directly into
// template data as {{csrf_token}}.
func GetCSRFTokenForForm(w http.ResponseWriter, r *http.Request) string {
	token := ensureToken(w, r)
	return "<input type=\"hidden\" name=\"csrf_token\" value=\"" + token + "\">"
}
