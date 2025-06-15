package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRFMiddlewareSuccess(t *testing.T) {
	token := "token123"
	handlerCalled := false
	h := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	body := strings.NewReader("csrf_token=" + token)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Result().StatusCode)
	}
	if !handlerCalled {
		t.Fatalf("handler not called")
	}
}

func TestCSRFMiddlewareMissingToken(t *testing.T) {
	h := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "abc"})
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Result().StatusCode)
	}
}

func TestCSRFMiddlewareInvalidToken(t *testing.T) {
	h := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	body := strings.NewReader("csrf_token=wrong")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "right"})
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Result().StatusCode)
	}
}
