package jobs

import (
	"encoding/json"
	"log/slog"
)

// printJob is an example job function that expects a JSON payload with a "message" field.
func PrintJob(payload []byte) error {
	var data struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return err
	}
	slog.Info("printJob", "message", data.Message)
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
