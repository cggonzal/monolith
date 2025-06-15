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

// GetCSRFTokenForForm ensures a CSRF token cookie exists and returns a hidden
// input element containing the token. Pass the returned string directly into
// template data as {{csrf_token}}.
func GetCSRFTokenForForm(w http.ResponseWriter, r *http.Request) string {
	token := ""
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		token = generateToken()
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	} else {
		token = c.Value
	}
	return "<input type=\"hidden\" name=\"csrf_token\" value=\"" + token + "\">"
}
