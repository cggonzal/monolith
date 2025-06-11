package jobs

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"monolith/config"
)

func TestEmailJob(t *testing.T) {
	var received url.Values
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST got %s", r.Method)
		}
		if r.URL.Path != "/test/messages" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		u, p, ok := r.BasicAuth()
		if !ok || u != "api" || p != "key" {
			t.Errorf("unexpected auth %v %v", u, p)
		}
		body, _ := io.ReadAll(r.Body)
		r.Body.Close()
		received, _ = url.ParseQuery(string(body))
		w.WriteHeader(200)
	}))
	defer ts.Close()

	oldBase := config.MAILGUN_API_BASE
	oldDomain := config.MAILGUN_DOMAIN
	oldKey := config.MAILGUN_API_KEY
	config.MAILGUN_API_BASE = ts.URL
	config.MAILGUN_DOMAIN = "test"
	config.MAILGUN_API_KEY = "key"
	defer func() {
		config.MAILGUN_API_BASE = oldBase
		config.MAILGUN_DOMAIN = oldDomain
		config.MAILGUN_API_KEY = oldKey
	}()

	p := emailPayload{Subject: "subj", Body: "body", Sender: "from@example.com", To: []string{"a@example.com"}}
	b, _ := json.Marshal(p)
	if err := EmailJob(string(b)); err != nil {
		t.Fatalf("EmailJob error: %v", err)
	}
	if received.Get("subject") != "subj" || received.Get("text") != "body" {
		t.Fatalf("unexpected payload: %v", received)
	}
}
