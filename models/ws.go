package models

import (
	"errors"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Message is the GORM model used to store incoming messages used by web sockets.
type Message struct {
	ID        uint `gorm:"primaryKey"`
	Channel   string
	Content   string
	CreatedAt time.Time
}

// Validate ensures required message fields are populated.
func (m *Message) Validate() error {
	if strings.TrimSpace(m.Channel) == "" {
		log.Print("channel required")
		return errors.New("channel required")
	}
	if strings.TrimSpace(m.Content) == "" {
		log.Print("content required")
		return errors.New("content required")
	}
	return nil
}

// BeforeSave calls Validate before persisting the Message.
func (m *Message) BeforeSave(tx *gorm.DB) error {
	return beforeSave(m, tx)
}
