package jobs

import (
	"crudapp/models"
	"errors"
	"log"
	"time"

	"crudapp/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// JobFunc defines the signature for functions that process jobs.
type JobFunc func(payload string) error

// JobQueue handles enqueuing and processing jobs.
// It uses a registry to map job types to their processing functions.
type JobQueue struct {
	db         *gorm.DB
	numWorkers int
	registry   map[models.JobType]JobFunc
	quit       chan struct{}
}

var JOB_QUEUE *JobQueue

// number of workers in the job queue. Modify as needed.
const NUM_WORKERS = 4

func init() {
	JOB_QUEUE = newJobQueue(db.DB, NUM_WORKERS)
}

// Use this function to access the job queue. returns a pointer to the job queue.
func GetJobQueue() *JobQueue {
	return JOB_QUEUE
}

// NewJobQueue creates a new JobQueue with a database connection and number of workers.
func newJobQueue(db *gorm.DB, numWorkers int) *JobQueue {
	return &JobQueue{
		db:         db,
		numWorkers: numWorkers,
		registry:   make(map[models.JobType]JobFunc),
		quit:       make(chan struct{}),
	}
}

// Register associates a job type with its processing function.
func (jq *JobQueue) register(jobType models.JobType, jobFunc JobFunc) {
	jq.registry[jobType] = jobFunc
}

// Start launches worker goroutines to process jobs.
func (jq *JobQueue) start() {
	for i := 0; i < jq.numWorkers; i++ {
		go jq.worker(i)
	}
}

// Stop signals the job queue to stop processing.
func (jq *JobQueue) stop() {
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
				job.Status = models.JobStatusFailed
			} else {
				err = jobFunc(job.Payload)
				if err != nil {
					log.Printf("Worker %d: job %d failed: %v", workerID, job.ID, err)
					job.Status = models.JobStatusFailed
				} else {
					job.Status = models.JobStatusCompleted
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
func (jq *JobQueue) fetchJob() (*models.Job, error) {
	var job models.Job
	err := jq.db.Transaction(func(tx *gorm.DB) error {
		// Lock the row so no other worker picks it up.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ?", models.JobStatusPending).
			Order("created_at").
			Limit(1).
			First(&job).Error; err != nil {
			return err
		}
		// Mark the job as processing.
		job.Status = models.JobStatusProcessing
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
func (jq *JobQueue) AddJob(jobType models.JobType, payload string) error {
	job := models.Job{
		Type:    jobType,
		Payload: payload,
		Status:  models.JobStatusPending,
	}
	return jq.db.Create(&job).Error
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
