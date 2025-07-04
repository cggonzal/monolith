package models

import "time"

// Message is the GORM model used to store incoming messages used by web sockets.
type Message struct {
	ID        uint `gorm:"primaryKey"`
	IsActive  bool `gorm:"default:true"`
	Channel   string
	Content   string
	CreatedAt time.Time
}
