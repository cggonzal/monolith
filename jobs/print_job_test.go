package jobs

import "testing"

func TestPrintJob(t *testing.T) {
	if err := PrintJob([]byte(`{"message":"hi"}`)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := PrintJob([]byte("invalid")); err == nil {
		t.Fatalf("expected error on invalid json")
	}
}
