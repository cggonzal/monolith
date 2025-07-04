package email

import (
	"encoding/json"
	"reflect"
	"testing"
	"unsafe"

	"monolith/app/jobs"
	"monolith/app/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setQueueDB(db *gorm.DB) {
	jq := jobs.GetJobQueue()
	v := reflect.ValueOf(jq).Elem().FieldByName("db")
	reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Set(reflect.ValueOf(db))
}

func setup(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.Job{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	jobs.InitJobQueue()
	setQueueDB(db)
	return db
}

func TestSendEmail(t *testing.T) {
	db := setup(t)
	if err := SendEmail("s", "b", "from@example.com", []string{"to@example.com"}); err != nil {
		t.Fatalf("SendEmail: %v", err)
	}
	var job models.Job
	if err := db.First(&job).Error; err != nil {
		t.Fatalf("db: %v", err)
	}
	if job.Type != models.JobTypeEmail {
		t.Fatalf("type %v", job.Type)
	}
	var p map[string]any
	if err := json.Unmarshal(job.Payload, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p["subject"] != "s" {
		t.Fatalf("payload %v", p)
	}
}
