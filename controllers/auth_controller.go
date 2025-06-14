package controllers

import (
	"net/http"

	"monolith/db"
	"monolith/models"
	"monolith/session"
	"monolith/templates"
)

type AuthController struct{}

var AuthCtrl = &AuthController{}

// ShowLoginForm renders the login page
func (ac *AuthController) ShowLoginForm(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "login.html.tmpl", nil)
}

// ShowSignupForm renders the signup page
func (ac *AuthController) ShowSignupForm(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "signup.html.tmpl", nil)
}

// Logout clears the session and redirects to home
func (ac *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	session.Logout(w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Signup handles user registration
func (ac *AuthController) Signup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	email := r.FormValue("email")
	password := r.FormValue("password")
	if email == "" || password == "" {
		http.Error(w, "missing credentials", http.StatusBadRequest)
		return
	}
	if _, err := models.CreateUser(db.GetDB(), email, password); err != nil {
		http.Error(w, "could not create user", http.StatusInternalServerError)
		return
	}
	session.SetLoggedIn(w, r, email)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// Login authenticates an existing user
func (ac *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	email := r.FormValue("email")
	password := r.FormValue("password")
	if _, err := models.AuthenticateUser(db.GetDB(), email, password); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	session.SetLoggedIn(w, r, email)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
