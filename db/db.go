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
	// Apply recommended connection pragmas for better concurrency and durability
	// See https://www.sqlite.org/pragma.html for details about each pragma.
	dsn := "app.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=wal_autocheckpoint(0)"
	dbHandle, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{}) // Change to postgres.Open(...) for PostgreSQL
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}

	// Auto-migrate all registered models
	dbHandle.AutoMigrate(
		&models.Job{},
		&models.RecurringJob{},
		&models.Message{},
	)
}
