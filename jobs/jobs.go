package jobs

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// JobType defines an enum for job types.
type JobType int

const (
	JobTypePrint JobType = iota
	JobTypeSum
)

// jobStatus defines an enum for job status
type jobStatus int

const (
	jobStatusPending jobStatus = iota
	jobStatusProcessing
	jobStatusCompleted
	jobStatusFailed
)

// Job represents a unit of work.
type Job struct {
	ID        uint      `gorm:"primaryKey"`
	Type      JobType   // Using our enum for job types.
	Payload   string    // JSON encoded arguments.
	Status    jobStatus // "pending", "processing", "completed", "failed".
	CreatedAt time.Time
	UpdatedAt time.Time
}

// JobFunc defines the signature for functions that process jobs.
type JobFunc func(payload string) error

// JobQueue handles enqueuing and processing jobs.
// It uses a registry to map job types to their processing functions.
type JobQueue struct {
	db         *gorm.DB
	numWorkers int
	registry   map[JobType]JobFunc
	quit       chan struct{}
}

// NewJobQueue creates a new JobQueue with a database connection and number of workers.
func NewJobQueue(db *gorm.DB, numWorkers int) *JobQueue {
	return &JobQueue{
		db:         db,
		numWorkers: numWorkers,
		registry:   make(map[JobType]JobFunc),
		quit:       make(chan struct{}),
	}
}

// Register associates a job type with its processing function.
func (jq *JobQueue) Register(jobType JobType, jobFunc JobFunc) {
	jq.registry[jobType] = jobFunc
}

// Start launches worker goroutines to process jobs.
func (jq *JobQueue) Start() {
	for i := 0; i < jq.numWorkers; i++ {
		go jq.worker(i)
	}
}

// Stop signals the job queue to stop processing.
func (jq *JobQueue) Stop() {
	close(jq.quit)
}

// worker continuously fetches and processes jobs.
func (jq *JobQueue) worker(workerID int) {
	log.Printf("Worker %d started", workerID)
	for {
		select {
		case <-jq.quit:
			log.Printf("Worker %d stopping", workerID)
			return
		default:
			job, err := jq.fetchJob()
			if err != nil {
				log.Printf("Worker %d encountered error fetching job: %v", workerID, err)
				time.Sleep(2 * time.Second)
				continue
			}
			if job == nil {
				// No pending job found; wait before polling again.
				time.Sleep(2 * time.Second)
				continue
			}

			log.Printf("Worker %d processing job %d of type %d", workerID, job.ID, job.Type)
			jobFunc, exists := jq.registry[job.Type]
			if !exists {
				log.Printf("Worker %d: no registered function for job type %d", workerID, job.Type)
				job.Status = jobStatusFailed
			} else {
				err = jobFunc(job.Payload)
				if err != nil {
					log.Printf("Worker %d: job %d failed: %v", workerID, job.ID, err)
					job.Status = jobStatusFailed
				} else {
					job.Status = jobStatusCompleted
				}
			}
			// Update the job status in the database.
			if err := jq.db.Save(job).Error; err != nil {
				log.Printf("Worker %d: failed to update job %d: %v", workerID, job.ID, err)
			}
		}
	}
}

// fetchJob retrieves one pending job and marks it as processing in a transaction.
func (jq *JobQueue) fetchJob() (*Job, error) {
	var job Job
	err := jq.db.Transaction(func(tx *gorm.DB) error {
		// Lock the row so no other worker picks it up.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ?", jobStatusPending).
			Order("created_at").
			Limit(1).
			First(&job).Error; err != nil {
			return err
		}
		// Mark the job as processing.
		job.Status = jobStatusProcessing
		return tx.Save(&job).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

// AddJob enqueues a new job with status "pending".
// The payload should be a JSON-encoded string representing the arguments.
func (jq *JobQueue) AddJob(jobType JobType, payload string) error {
	job := Job{
		Type:    jobType,
		Payload: payload,
		Status:  jobStatusPending,
	}
	return jq.db.Create(&job).Error
}

// printJob is an example job function that expects a JSON payload with a "message" field.
func printJob(payload string) error {
	var data struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return err
	}
	log.Printf("printJob: %s", data.Message)
	return nil
}

// sumJob is an example job function that expects a JSON payload with "a" and "b" fields.
func sumJob(payload string) error {
	var data struct {
		A int `json:"a"`
		B int `json:"b"`
	}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return err
	}
	sum := data.A + data.B
	log.Printf("sumJob: %d + %d = %d", data.A, data.B, sum)
	return nil
}

/*
// example usage
func main() {
	// Choose your database driver:
	// Using Sqlite:
	db, err := gorm.Open(sqlite.Open("jobs.db"), &gorm.Config{})
	// To use PostgreSQL instead, uncomment below and comment out the Sqlite connection.

	//	dsn := "host=localhost user=postgres password=postgres dbname=jobqueue port=5432 sslmode=disable"
	//	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Auto-migrate the Job model.
	if err := db.AutoMigrate(&Job{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// Initialize the job queue with, for example, 3 workers.
	jobQueue := NewJobQueue(db, 3)

	// Register job functions using our enum types.
	jobQueue.Register(JobTypePrint, printJob)
	jobQueue.Register(JobTypeSum, sumJob)

	jobQueue.Start()

	// Add demo jobs.
	// Enqueue a "print" job.
	payloadPrint, _ := json.Marshal(map[string]string{"message": "Hello, World!"})
	if err := jobQueue.AddJob(JobTypePrint, string(payloadPrint)); err != nil {
		log.Printf("failed to add job: %v", err)
	}

	// Enqueue a "sum" job.
	payloadSum, _ := json.Marshal(map[string]int{"a": 10, "b": 20})
	if err := jobQueue.AddJob(JobTypeSum, string(payloadSum)); err != nil {
		log.Printf("failed to add job: %v", err)
	}

	// Let the queue process jobs for a while.
	time.Sleep(10 * time.Second)
	jobQueue.Stop()
	log.Println("Job queue stopped")
}
*/
