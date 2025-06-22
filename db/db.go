/*
Package db bootstraps the GORM database connection and exposes helpers for
other packages to retrieve the initialized handle.
*/
package db

import (
	"log"

	"monolith/models"

	// "gorm.io/driver/postgres" // Change to postgres if desired
	"github.com/glebarez/sqlite" // Pure go SQLite driver, checkout https://github.com/glebarez/sqlite for details
	"gorm.io/gorm"
)

var dbHandle *gorm.DB

func GetDB() *gorm.DB {
	return dbHandle
}

// InitDB initializes the database connection
func InitDB() {
	var err error
	dbHandle, err = gorm.Open(sqlite.Open("app.db"), &gorm.Config{}) // Change to postgres.Open(...) for PostgreSQL
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}

	// Auto-migrate all registered models
	dbHandle.AutoMigrate(
		&models.Job{},
		&models.Message{},
	)
}
