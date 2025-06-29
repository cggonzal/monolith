package session

import (
	"net/http/httptest"
	"testing"

	"monolith/config"
)

func TestSessionLoginLogout(t *testing.T) {
	config.InitConfig()
	InitSession()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	SetLoggedIn(w, req, "test@example.com")
	cookie := w.Result().Cookies()[0]

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.AddCookie(cookie)
	if !IsLoggedIn(req2) {
		t.Fatal("expected logged in")
	}

	w2 := httptest.NewRecorder()
	Logout(w2, req2)
	cookie2 := w2.Result().Cookies()[0]
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.AddCookie(cookie2)
	if IsLoggedIn(req3) {
		t.Fatal("expected logged out")
	}
}
