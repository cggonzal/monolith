package jobs

import "testing"

func TestPrintJob(t *testing.T) {
	if err := PrintJob(`{"message":"hi"}`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := PrintJob("invalid"); err == nil {
		t.Fatalf("expected error on invalid json")
	}
}

func TestSumJob(t *testing.T) {
	if err := SumJob(`{"a":2,"b":3}`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := SumJob("bad"); err == nil {
		t.Fatalf("expected error on invalid json")
	}
}
