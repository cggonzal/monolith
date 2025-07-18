/*
An implementation of a job queue.

Job functions live in separate `*_job.go` files within the `jobs` package and
are registered in `jobs/job_queue.go`.

Use the generator to scaffold a new job:

	make generator job DoSomething

The generator creates `jobs/do_something_job.go` with a stub `DoSomethingJob` function,
adds a matching `JobTypeDoSomething` enum and registers it with the job queue for you.
*/
package jobs

import (
	"errors"
	"log/slog"
	"monolith/app/models"
	"strconv"
	"strings"
	"time"

	"monolith/app/config"
	"monolith/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// matcher represents a single cron field matcher.
type matcher func(int) bool

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
	jobQueue.register(models.JobTypeExample, ExampleJob)
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
				err = jobFunc(job.Payload)
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
// The payload should be JSON-encoded bytes representing the arguments.
func (jq *JobQueue) AddJob(jobType models.JobType, payload []byte) error {
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
func (jq *JobQueue) AddRecurringJob(jobType models.JobType, payload []byte, cron string) error {
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

// parseField parses a cron field value and returns a matcher for that field.
// Supported forms:
//
//	"*"     - any value
//	"*/N"   - every N units
//	"N"     - exact value N
func parseField(field string, min, max int, dow bool) (matcher, error) {
	if field == "*" {
		return func(int) bool { return true }, nil
	}
	if strings.HasPrefix(field, "*/") {
		n, err := strconv.Atoi(field[2:])
		if err != nil || n <= 0 {
			return nil, errors.New("invalid step")
		}
		return func(v int) bool { return (v-min)%n == 0 }, nil
	}
	val, err := strconv.Atoi(field)
	if err != nil {
		return nil, errors.New("invalid field")
	}
	if dow && val == 7 {
		val = 0
	}
	if val < min || val > max {
		return nil, errors.New("value out of range")
	}
	return func(v int) bool { return v == val }, nil
}

// nextCronTime returns the next time after 'from' that matches the cron
// expression. It supports standard 5-field cron syntax (minute, hour,
// day-of-month, month, day-of-week) with simple forms for each field.
func nextCronTime(expr string, from time.Time) (time.Time, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return time.Time{}, errors.New("invalid cron expression")
	}

	minuteM, err := parseField(fields[0], 0, 59, false)
	if err != nil {
		return time.Time{}, err
	}
	hourM, err := parseField(fields[1], 0, 23, false)
	if err != nil {
		return time.Time{}, err
	}
	domM, err := parseField(fields[2], 1, 31, false)
	if err != nil {
		return time.Time{}, err
	}
	monthM, err := parseField(fields[3], 1, 12, false)
	if err != nil {
		return time.Time{}, err
	}
	dowM, err := parseField(fields[4], 0, 7, true)
	if err != nil {
		return time.Time{}, err
	}

	t := from.Add(time.Minute).Truncate(time.Minute)
	// search up to two years ahead
	for i := 0; i < 2*525600; i++ {
		if minuteM(t.Minute()) && hourM(t.Hour()) &&
			monthM(int(t.Month())) &&
			(domM(t.Day()) || dowM(int(t.Weekday()))) {
			return t, nil
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, errors.New("unable to compute next run time")
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
       if err := jobQueue.AddJob(JobTypePrint, payloadPrint); err != nil {
               log.Printf("failed to add job: %v", err)
       }

	// Enqueue a "sum" job.
       payloadSum, _ := json.Marshal(map[string]int{"a": 10, "b": 20})
       if err := jobQueue.AddJob(JobTypeSum, payloadSum); err != nil {
               log.Printf("failed to add job: %v", err)
       }

	// Let the queue process jobs for a while.
	time.Sleep(10 * time.Second)
	jobQueue.Stop()
	log.Println("Job queue stopped")
}
*/
