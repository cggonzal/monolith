package db

import (
	"os"
	"testing"
)

func TestConnectAndGetDB(t *testing.T) {
	os.Remove("app.db")
	InitDB()
	if GetDB() == nil {
		t.Fatal("expected database handle, got nil")
	}
	// cleanup
	os.Remove("app.db")
}
