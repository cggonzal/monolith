package config

import "testing"

func TestJobQueueNumWorkers(t *testing.T) {
	if JOB_QUEUE_NUM_WORKERS != 4 {
		t.Fatalf("expected 4 workers, got %d", JOB_QUEUE_NUM_WORKERS)
	}
}
