package models

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User represents a user in the database
type User struct {
	gorm.Model          // Adds ID, CreatedAt, UpdatedAt, DeletedAt fields
	Email        string `gorm:"unique;not null"`
	PasswordHash []byte
	IsActive     bool `gorm:"default:true"`
	IsAdmin      bool `gorm:"default:false"`
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
func CreateUser(db *gorm.DB, email, password string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := User{
		Email:        email,
		PasswordHash: hash,
	}
	if err := db.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// AuthenticateUser verifies the provided credentials and returns the user if valid.
func AuthenticateUser(db *gorm.DB, email, password string) (*User, error) {
	user, err := GetUser(db, email)
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)) != nil {
		return nil, errors.New("invalid credentials")
	}
	return user, nil
}
