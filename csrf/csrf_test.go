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
