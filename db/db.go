package db

import (
	"log"

	"crudapp/models"

	// "gorm.io/driver/postgres" // Change to postgres if desired
	"github.com/glebarez/sqlite" // Pure go SQLite driver, checkout https://github.com/glebarez/sqlite for details
	"gorm.io/gorm"
)

var DB *gorm.DB

// Connect initializes the database connection
func Connect() {
	var err error
	DB, err = gorm.Open(sqlite.Open("app.db"), &gorm.Config{}) // Change to postgres.Open(...) for PostgreSQL
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}

	// Auto-migrate the User model
	DB.AutoMigrate(
		&models.User{},
		&models.Job{},
	)
}
