package jobs

import (
	"encoding/json"
	"log/slog"
)

type ExamplePayload struct {
	Message string `json:"message"`
}

// ExampleJob is an example job function that expects a JSON payload with a "message" field.
func ExampleJob(payload []byte) error {
	var p ExamplePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return err
	}
	slog.Info("ExampleJob", "message", p.Message)
	return nil
}
