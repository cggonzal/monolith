package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"monolith/app/config"
)

func TestRateLimitMiddleware(t *testing.T) {
	// Set small limit for test
	old := config.RATE_LIMIT_REQUESTS_PER_MINUTE
	config.RATE_LIMIT_REQUESTS_PER_MINUTE = 2
	defer func() { config.RATE_LIMIT_REQUESTS_PER_MINUTE = old }()

	h := RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Result().StatusCode)
	}

	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Result().StatusCode)
	}

	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Result().StatusCode)
	}

	// After a minute, should allow again
	time.Sleep(time.Minute)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after reset, got %d", w.Result().StatusCode)
	}
}
