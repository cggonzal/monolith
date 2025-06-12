package ws

import (
	"testing"
	"time"

	"monolith/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.Message{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestBroadcastPersists(t *testing.T) {
	db := setupDB(t)
	h := newHub(db)
	go h.Run()
	h.Broadcast("ch", []byte("hello"))
	time.Sleep(50 * time.Millisecond)
	var msg models.Message
	if err := db.First(&msg).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if msg.Channel != "ch" || msg.Content != "hello" {
		t.Fatalf("unexpected message %#v", msg)
	}
}
