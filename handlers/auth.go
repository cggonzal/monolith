package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"crudapp/db"
	"crudapp/models"
	"crudapp/session"

	"gorm.io/gorm"
)

// HandleGoogleLogin redirects the user to Google's OAuth login page
func HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	conf := session.GetGoogleOAuthConfig()
	url := conf.AuthCodeURL("random-state")
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// HandleGoogleCallback processes the OAuth2 callback and saves user to DB
func HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	conf := session.GetGoogleOAuthConfig()
	code := r.URL.Query().Get("code")

	token, err := conf.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	client := conf.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Extract user info from Google API
	var userInfo struct {
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"picture"`
	}
	json.NewDecoder(resp.Body).Decode(&userInfo)

	// Try to get user from DB
	user, err := models.GetUser(db.DB, userInfo.Email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// If user does not exist, create a new one
		user, err = models.CreateUser(db.DB, userInfo.Email, userInfo.Name, userInfo.AvatarURL)
		if err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Store user session
	session.SetLoggedIn(w, r, user.Email)

	// Redirect to dashboard
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
