/*
Package models defines the database models used throughout the application.
*/
package models

import (
	"time"

	"gorm.io/gorm"
)

// JobType defines an enum for job types.
type JobType int

const (
	JobTypePrint JobType = iota
	JobTypeEmail
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
	gorm.Model           // Adds ID, CreatedAt, UpdatedAt, DeletedAt fields
	Type       JobType   // Using our enum for job types.
	Payload    string    // JSON encoded arguments.
	Status     JobStatus // Using our enum for status types.
}

// RecurringJob defines a job that should be enqueued on a schedule described
// by a cron expression. NextRunAt stores the next time this job should be
// enqueued.
type RecurringJob struct {
	gorm.Model
	Type      JobType
	Payload   string
	CronExpr  string
	NextRunAt time.Time
}
