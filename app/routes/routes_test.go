package routes

import (
	"embed"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRandomRouteNotFound(t *testing.T) {
	// zero-value embed.FS provides an empty filesystem for static files
	handler := InitServerHandler(embed.FS{})

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Result().StatusCode)
	}
}

func TestHelloEndpoint(t *testing.T) {
	handler := InitServerHandler(embed.FS{})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Result().StatusCode)
	}
	if w.Body.String() != "Hello World" {
		t.Fatalf("expected body 'Hello World', got %q", w.Body.String())
	}
}
