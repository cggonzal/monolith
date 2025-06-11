package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestCreateAndGetUser(t *testing.T) {
	db := setupTestDB(t)
	email := "test@example.com"
	_, err := CreateUser(db, email, "name", "")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	u, err := GetUser(db, email)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if u.Email != email {
		t.Fatalf("expected email %s got %s", email, u.Email)
	}
}
