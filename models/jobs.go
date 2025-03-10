package models

import (
	"time"
)

// JobType defines an enum for job types.
type JobType int

const (
	JobTypePrint JobType = iota
	JobTypeSum
)

// JobStatus defines an enum for job status
type JobStatus int

const (
	JobStatusPending JobStatus = iota
	JobStatusProcessing
	JobStatusCompleted
	JobStatusFailed
)

// Job represents a unit of work.
type Job struct {
	ID        uint      `gorm:"primaryKey"`
	Type      JobType   // Using our enum for job types.
	Payload   string    // JSON encoded arguments.
	Status    JobStatus // Using our enum for status types.
	CreatedAt time.Time
	UpdatedAt time.Time
}
