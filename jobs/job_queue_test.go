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
	if err := db.AutoMigrate(&models.Job{}, &models.RecurringJob{}); err != nil {
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
	jq.register(models.JobTypePrint, func([]byte) error { return nil })
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
	jq.register(models.JobTypePrint, func([]byte) error {
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
	jq.register(models.JobTypePrint, func([]byte) error {
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
	jq.register(models.JobTypePrint, func([]byte) error { return nil })
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
	jq.register(models.JobTypePrint, func([]byte) error {
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

func TestRecurringJob(t *testing.T) {
	jq, db := setupQueue(t, 1)
	jq.register(models.JobTypePrint, func([]byte) error { return nil })
	jq.start()
	if err := jq.AddRecurringJob(models.JobTypePrint, "{}", "* * * * *"); err != nil {
		t.Fatalf("AddRecurringJob: %v", err)
	}
	var rj models.RecurringJob
	if err := db.First(&rj).Error; err != nil {
		t.Fatalf("query recurring: %v", err)
	}
	if err := db.Model(&rj).Update("next_run_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("update: %v", err)
	}
	jq.processRecurringJobs(time.Now())
	var job models.Job
	if err := db.First(&job).Error; err != nil {
		t.Fatalf("queued job missing: %v", err)
	}
}

func TestCronRunsAtFutureTime(t *testing.T) {
	jq, db := setupQueue(t, 0)
	jq.register(models.JobTypePrint, func([]byte) error { return nil })
	if err := jq.AddRecurringJob(models.JobTypePrint, "{}", "* * * * *"); err != nil {
		t.Fatalf("AddRecurringJob: %v", err)
	}
	var rj models.RecurringJob
	if err := db.First(&rj).Error; err != nil {
		t.Fatalf("query recurring: %v", err)
	}
	now := time.Now()
	if err := db.Model(&rj).Update("next_run_at", now.Add(1*time.Second)).Error; err != nil {
		t.Fatalf("update: %v", err)
	}
	jq.processRecurringJobs(now)
	var count int64
	if err := db.Model(&models.Job{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no job queued yet")
	}
	jq.processRecurringJobs(now.Add(2 * time.Second))
	if err := db.Model(&models.Job{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected job to be queued after time elapsed")
	}
}

func TestNextCronTime(t *testing.T) {
	base := time.Date(2023, time.June, 30, 12, 34, 56, 0, time.UTC)
	tests := []struct {
		expr string
		from time.Time
		want time.Time
	}{
		{"* * * * *", base, time.Date(2023, time.June, 30, 12, 35, 0, 0, time.UTC)},
		{"*/5 * * * *", time.Date(2023, time.June, 30, 12, 34, 0, 0, time.UTC), time.Date(2023, time.June, 30, 12, 35, 0, 0, time.UTC)},
		{"30 14 * * *", time.Date(2023, time.June, 30, 14, 29, 0, 0, time.UTC), time.Date(2023, time.June, 30, 14, 30, 0, 0, time.UTC)},
		{"30 14 * * *", time.Date(2023, time.June, 30, 14, 31, 0, 0, time.UTC), time.Date(2023, time.July, 1, 14, 30, 0, 0, time.UTC)},
		{"0 6 * * *", time.Date(2023, time.June, 30, 5, 59, 0, 0, time.UTC), time.Date(2023, time.June, 30, 6, 0, 0, 0, time.UTC)},
		{"0 0 5 * 1", time.Date(2023, time.June, 3, 23, 59, 0, 0, time.UTC), time.Date(2023, time.June, 5, 0, 0, 0, 0, time.UTC)},
		{"0 0 1 7 *", time.Date(2023, time.June, 15, 0, 0, 0, 0, time.UTC), time.Date(2023, time.July, 1, 0, 0, 0, 0, time.UTC)},
		{"0 0 * 12 *", time.Date(2023, time.November, 30, 0, 0, 0, 0, time.UTC), time.Date(2023, time.December, 1, 0, 0, 0, 0, time.UTC)},
		{"0 0 1 * 1", time.Date(2023, time.June, 10, 0, 0, 0, 0, time.UTC), time.Date(2023, time.June, 12, 0, 0, 0, 0, time.UTC)},
		{"0 0 * * 7", time.Date(2023, time.June, 10, 0, 0, 0, 0, time.UTC), time.Date(2023, time.June, 11, 0, 0, 0, 0, time.UTC)},
		{"0 */3 * * *", time.Date(2023, time.June, 10, 2, 10, 0, 0, time.UTC), time.Date(2023, time.June, 10, 3, 0, 0, 0, time.UTC)},
		{"0 0 */2 * *", time.Date(2023, time.June, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, time.June, 2, 0, 0, 0, 0, time.UTC)},
		{"0 0 10 * 1", time.Date(2023, time.June, 9, 12, 0, 0, 0, time.UTC), time.Date(2023, time.June, 10, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range tests {
		got, err := nextCronTime(tc.expr, tc.from)
		if err != nil {
			t.Fatalf("nextCronTime(%s): %v", tc.expr, err)
		}
		if !got.Equal(tc.want) {
			t.Fatalf("expr %s from %v expected %v got %v", tc.expr, tc.from, tc.want, got)
		}
	}
}
