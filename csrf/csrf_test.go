package csrf

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetCSRFTokenForFormSetsCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	field := GetCSRFTokenForForm(w, req)

	if !strings.Contains(field, "name=\"csrf_token\"") {
		t.Fatalf("token field missing name attribute")
	}
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected cookie to be set")
	}
	token := cookies[0].Value
	if token == "" {
		t.Fatalf("empty token in cookie")
	}
	if !strings.Contains(field, token) {
		t.Fatalf("token not in hidden field")
	}
}

func TestGetCSRFTokenReturnsToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	token := GetCSRFToken(w, req)
	if token == "" {
		t.Fatalf("expected token")
	}
	cookies := w.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Value != token {
		t.Fatalf("cookie not set to token")
	}
}

func TestGetCSRFMetaTagIncludesToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	meta := GetCSRFMetaTag(w, req)
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected cookie")
	}
	token := cookies[0].Value
	if !strings.Contains(meta, token) {
		t.Fatalf("meta tag missing token")
	}
}
