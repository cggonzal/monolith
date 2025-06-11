package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"monolith/session"
)

func TestRequireLogin(t *testing.T) {
	handlerCalled := false
	handler := RequireLogin(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})
	// request without login
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Result().StatusCode != http.StatusSeeOther {
		t.Errorf("expected redirect status, got %d", w.Result().StatusCode)
	}
	if handlerCalled {
		t.Errorf("handler should not be called when not logged in")
	}

	// logged in request
	req2 := httptest.NewRequest("GET", "/", nil)
	w2 := httptest.NewRecorder()
	session.SetLoggedIn(w2, req2, "test@example.com")
	cookie := w2.Result().Cookies()[0]
	req2.AddCookie(cookie)
	w3 := httptest.NewRecorder()
	handler(w3, req2)
	if w3.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200 status when logged in, got %d", w3.Result().StatusCode)
	}
	if !handlerCalled {
		t.Errorf("handler should be called when logged in")
	}
}
