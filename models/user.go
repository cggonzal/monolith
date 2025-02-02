package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user in the database
type User struct {
	ID        uint   `gorm:"primaryKey"`
	Email     string `gorm:"unique;not null"`
	Name      string
	AvatarURL string
	CreatedAt time.Time
	UpdatedAt time.Time
	IsActive  bool `gorm:"default:true"`
	IsAdmin   bool `gorm:"default:false"`
}

// GetUser fetches a user by email from the database
func GetUser(db *gorm.DB, email string) (*User, error) {
	var user User
	result := db.Where(&User{Email: email}).Take(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// CreateUser inserts a new user into the database
func CreateUser(db *gorm.DB, email, name, avatarURL string) (*User, error) {
	user := User{
		Email:     email,
		Name:      name,
		AvatarURL: avatarURL,
	}
	if err := db.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
