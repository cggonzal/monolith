package server_management

import (
	"os"
	"testing"
)

func TestSdListenersNoEnv(t *testing.T) {
	os.Unsetenv("LISTEN_PID")
	os.Unsetenv("LISTEN_FDS")
	if _, err := sdListeners(); err == nil {
		t.Fatal("expected error when env vars not set")
	}
}

func TestSdNotifyReadyNoSocket(t *testing.T) {
	os.Unsetenv("NOTIFY_SOCKET")
	sdNotifyReady()
}
