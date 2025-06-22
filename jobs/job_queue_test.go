package jobs

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"monolith/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupQueue creates an in-memory database and a JobQueue with the given number of workers.
func setupQueue(t *testing.T, workers int) (*JobQueue, *gorm.DB) {
	t.Helper()
	path := fmt.Sprintf("%s/queue_%d.db", t.TempDir(), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.Job{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	jq := newJobQueue(db, workers)
	if dbConn, err := db.DB(); err == nil {
		dbConn.SetMaxOpenConns(1)
		t.Cleanup(func() { dbConn.Close() })
	}
	return jq, db
}

func TestAddAndFetchJob(t *testing.T) {
	jq, db := setupQueue(t, 0)
	jq.register(models.JobTypePrint, func(string) error { return nil })
	if err := jq.AddJob(models.JobTypePrint, `{"message":"hi"}`); err != nil {
		t.Fatalf("AddJob: %v", err)
	}
	var job models.Job
	if err := db.First(&job).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if job.Status != models.JobStatusPending {
		t.Fatalf("expected pending, got %v", job.Status)
	}
	fetched, err := jq.fetchJob()
	if err != nil {
		t.Fatalf("fetchJob: %v", err)
	}
	if fetched == nil || fetched.ID != job.ID {
		t.Fatalf("wrong job fetched")
	}
	if fetched.Status != models.JobStatusProcessing {
		t.Fatalf("expected processing, got %v", fetched.Status)
	}
	var updated models.Job
	if err := db.First(&updated, job.ID).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if updated.Status != models.JobStatusProcessing {
		t.Fatalf("db not updated, status %v", updated.Status)
	}
	none, err := jq.fetchJob()
	if err != nil {
		t.Fatalf("fetchJob: %v", err)
	}
	if none != nil {
		t.Fatalf("expected nil job when none pending")
	}
}

func TestWorkerSuccess(t *testing.T) {
	jq, db := setupQueue(t, 1)
	done := make(chan struct{}, 1)
	jq.register(models.JobTypePrint, func(string) error {
		done <- struct{}{}
		return nil
	})
	jq.start()
	if err := jq.AddJob(models.JobTypePrint, "{}"); err != nil {
		t.Fatalf("AddJob: %v", err)
	}
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("job not processed")
	}
	var job models.Job
	if err := db.First(&job).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	for i := 0; i < 5 && job.Status != models.JobStatusCompleted; i++ {
		time.Sleep(20 * time.Millisecond)
		if err := db.First(&job, job.ID).Error; err != nil {
			t.Fatalf("query: %v", err)
		}
	}
	if job.Status != models.JobStatusCompleted {
		t.Fatalf("status %v", job.Status)
	}
}

func TestWorkerFailure(t *testing.T) {
	jq, db := setupQueue(t, 1)
	done := make(chan struct{}, 1)
	jq.register(models.JobTypePrint, func(string) error {
		done <- struct{}{}
		return errors.New("boom")
	})
	jq.start()
	if err := jq.AddJob(models.JobTypePrint, "{}"); err != nil {
		t.Fatalf("AddJob: %v", err)
	}
	<-done
	var job models.Job
	if err := db.First(&job).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	for i := 0; i < 5 && job.Status != models.JobStatusFailed; i++ {
		time.Sleep(20 * time.Millisecond)
		if err := db.First(&job, job.ID).Error; err != nil {
			t.Fatalf("query: %v", err)
		}
	}
	if job.Status != models.JobStatusFailed {
		t.Fatalf("status %v", job.Status)
	}
}

func TestUnregisteredJob(t *testing.T) {
	jq, db := setupQueue(t, 1)
	jq.register(models.JobTypePrint, func(string) error { return nil })
	jq.start()
	if err := jq.AddJob(models.JobTypeEmail, "{}"); err != nil {
		t.Fatalf("AddJob: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	var job models.Job
	if err := db.First(&job).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if job.Status != models.JobStatusFailed {
		t.Fatalf("status %v", job.Status)
	}
}

func TestMultipleWorkers(t *testing.T) {
	jq, db := setupQueue(t, 3)
	var mu sync.Mutex
	count := 0
	jq.register(models.JobTypePrint, func(string) error {
		mu.Lock()
		count++
		mu.Unlock()
		return nil
	})
	jq.start()
	for i := 0; i < 5; i++ {
		if err := jq.AddJob(models.JobTypePrint, fmt.Sprintf("{\"id\":%d}", i)); err != nil {
			t.Fatalf("AddJob: %v", err)
		}
	}
	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		if count == 5 {
			mu.Unlock()
			break
		}
		mu.Unlock()
	}
	mu.Lock()
	processed := count
	mu.Unlock()
	if processed != 5 {
		t.Fatalf("processed %d jobs", processed)
	}
	var jobs []models.Job
	if err := db.Find(&jobs).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	for _, j := range jobs {
		if j.Status != models.JobStatusCompleted {
			t.Fatalf("job %d status %v", j.ID, j.Status)
		}
	}
}
