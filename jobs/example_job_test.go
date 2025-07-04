package jobs

import "testing"

func TestExampleJob(t *testing.T) {
	if err := ExampleJob([]byte(`{"message":"hi"}`)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ExampleJob([]byte("invalid")); err == nil {
		t.Fatalf("expected error on invalid json")
	}
}
