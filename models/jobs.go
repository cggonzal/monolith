/*
Package models defines the database models used throughout the application.
*/
package models

import (
	"errors"
	"log"
	"strings"

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

// Validate ensures the Job has required fields.
func (j *Job) Validate() error {
	switch j.Type {
	case JobTypePrint, JobTypeEmail:
	default:
		log.Print("invalid job type")
		return errors.New("invalid job type")
	}

	switch j.Status {
	case JobStatusPending, JobStatusProcessing, JobStatusCompleted, JobStatusFailed:
	default:
		log.Print("invalid job status")
		return errors.New("invalid job status")
	}

	if strings.TrimSpace(j.Payload) == "" {
		log.Print("payload required")
		return errors.New("payload required")
	}
	return nil
}

// BeforeSave calls Validate before persisting the Job.
func (j *Job) BeforeSave(tx *gorm.DB) error {
	return beforeSave(j, tx)
}
