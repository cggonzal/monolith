package jobs

import (
	"errors"
	"log/slog"
	"monolith/models"
	"time"

	"monolith/config"
	"monolith/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// JobFunc defines the signature for functions that process jobs.
type JobFunc func(payload []byte) error

// JobQueue handles enqueuing and processing jobs.
// It uses a registry to map job types to their processing functions.
type JobQueue struct {
	db         *gorm.DB
	numWorkers int
	registry   map[models.JobType]JobFunc
	notifyCh   chan struct{}
}

// to access the job queue, use GetJobQueue(). DO NOT use this variable directly except for inside the init()
var jobQueue *JobQueue

func InitJobQueue() {
	jobQueue = newJobQueue(db.GetDB(), config.JOB_QUEUE_NUM_WORKERS)

	// register all jobs
	jobQueue.register(models.JobTypePrint, PrintJob)
	jobQueue.register(models.JobTypeEmail, EmailJob)

	// start the job queue only if a database connection is available
	if jobQueue.db != nil {
		jobQueue.start()
	} else {
		slog.Error("Job queue not started: no database connection available")
	}
}

// Use this function to access the job queue.
func GetJobQueue() *JobQueue {
	return jobQueue
}

// NewJobQueue creates a new JobQueue with a database connection and number of workers.
func newJobQueue(db *gorm.DB, numWorkers int) *JobQueue {
	return &JobQueue{
		db:         db,
		numWorkers: numWorkers,
		registry:   make(map[models.JobType]JobFunc),
		notifyCh:   make(chan struct{}, numWorkers),
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
	go jq.recurringScheduler()
}

// notify wakes workers that may be waiting for new jobs.
//
// The notify() function in the JobQueue struct is responsible for waking up worker goroutines
// that may be waiting for new jobs to process. Here’s how it works, step-by-step:
//
// 1. The function tries to send an empty struct (struct{}) into the notifyCh channel.
// 2. The channel notifyCh is buffered (with a size equal to the number of workers), so it can hold a limited number of notifications.
// 3. If the channel is not full, the notification is sent, and a waiting worker will be unblocked and can check for new jobs.
// 4. If the channel is already full (all workers are already notified or awake), the default case is executed, and nothing happens—this prevents blocking or overfilling the channel.
//
// This mechanism ensures that workers are efficiently notified of new jobs without unnecessary wake-ups or blocking.
func (jq *JobQueue) notify() {
	select {
	case jq.notifyCh <- struct{}{}:
	default:
	}
}

// worker continuously fetches and processes jobs.
func (jq *JobQueue) worker(workerID int) {
	slog.Info("worker started", "workerID", workerID)
	for {
		job, err := jq.fetchJob()
		if err == nil && job != nil {
			slog.Info("processing job", "workerID", workerID, "jobID", job.ID, "type", job.Type)
			jobFunc, exists := jq.registry[job.Type]
			if !exists {
				slog.Error("no registered job function", "workerID", workerID, "type", job.Type)
				job.Status = models.JobStatusFailed
			} else {
				err = jobFunc([]byte(job.Payload))
				if err != nil {
					slog.Error("job failed", "workerID", workerID, "jobID", job.ID, "error", err)
					job.Status = models.JobStatusFailed
				} else {
					job.Status = models.JobStatusCompleted
				}
			}
			if err := jq.db.Save(job).Error; err != nil {
				slog.Error("failed to update job", "workerID", workerID, "jobID", job.ID, "error", err)
			}
			continue
		}

		if err != nil {
			slog.Error("worker fetch error", "workerID", workerID, "error", err)
		}
		select {
		case <-jq.notifyCh:
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// fetchJob retrieves one pending job and marks it as processing in a transaction.
func (jq *JobQueue) fetchJob() (*models.Job, error) {
	var job models.Job
	err := jq.db.Transaction(func(tx *gorm.DB) error {
		// Lock the row so no other worker picks it up.
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ?", models.JobStatusPending).
			Order("created_at").
			Limit(1).
			Find(&job)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
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
	if err := jq.db.Create(&job).Error; err != nil {
		return err
	}
	jq.notify()
	return nil
}

// AddRecurringJob registers a job that should be enqueued on a recurring
// schedule described by a cron expression. The provided payload is passed to the job handler each time.
func (jq *JobQueue) AddRecurringJob(jobType models.JobType, payload string, cron string) error {
	if cron == "" {
		return errors.New("cron expression required")
	}
	next, err := nextCronTime(cron, time.Now())
	if err != nil {
		return err
	}
	rj := models.RecurringJob{
		Type:      jobType,
		Payload:   payload,
		CronExpr:  cron,
		NextRunAt: next,
	}
	return jq.db.Create(&rj).Error
}

// recurringScheduler periodically checks for recurring jobs that are due and
// enqueues them. It runs indefinitely in its own goroutine.
func (jq *JobQueue) recurringScheduler() {
	for {
		jq.processRecurringJobs(time.Now())
		time.Sleep(time.Minute)
	}
}

// processRecurringJobs enqueues all recurring jobs that should run at or before
// the provided time. It is exposed for tests.
func (jq *JobQueue) processRecurringJobs(now time.Time) {
	var rjobs []models.RecurringJob
	if err := jq.db.Where("next_run_at <= ?", now).Find(&rjobs).Error; err != nil {
		slog.Error("recurring scheduler query failed", "error", err)
		return
	}
	for _, rj := range rjobs {
		job := models.Job{Type: rj.Type, Payload: rj.Payload, Status: models.JobStatusPending}
		if err := jq.db.Create(&job).Error; err != nil {
			slog.Error("create job for recurring", "error", err)
			continue
		}
		jq.notify()
		next, err := nextCronTime(rj.CronExpr, now)
		if err != nil {
			slog.Error("compute next run", "error", err)
			continue
		}
		rj.NextRunAt = next
		if err := jq.db.Save(&rj).Error; err != nil {
			slog.Error("update recurring job", "error", err)
		}
	}
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
