package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestToCamelCase(t *testing.T) {
	cases := map[string]string{
		"foo_bar": "FooBar",
		"User":    "User",
		"":        "",
	}
	for in, exp := range cases {
		if got := toCamelCase(in); got != exp {
			t.Errorf("toCamelCase(%q)=%q want %q", in, got, exp)
		}
	}
}

func TestToSnakeCase(t *testing.T) {
	cases := map[string]string{
		"FooBar": "foo_bar",
		"widget": "widget",
		"":       "",
	}
	for in, exp := range cases {
		if got := toSnakeCase(in); got != exp {
			t.Errorf("toSnakeCase(%q)=%q want %q", in, got, exp)
		}
	}
}

func TestLeadingWhitespace(t *testing.T) {
	if ws := leadingWhitespace(" \t foo"); ws != " \t " {
		t.Fatalf("unexpected %q", ws)
	}
}

func TestRunModel(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	writeFile(t, "db/db.go", `package db
import "monolith/models"
func Connect() {
    dbHandle.AutoMigrate(
        &models.User{},
    )
}`)
	os.MkdirAll("models", 0755)

	if err := runModel([]string{"Widget", "name:string"}); err != nil {
		t.Fatalf("runModel: %v", err)
	}
	if _, err := os.Stat("models/widget.go"); err != nil {
		t.Fatalf("model file: %v", err)
	}
	data, _ := os.ReadFile("db/db.go")
	if !strings.Contains(string(data), "&models.Widget{}") {
		t.Fatalf("db not updated: %s", string(data))
	}
	if _, err := os.Stat("models/widget_test.go"); err != nil {
		t.Fatalf("model test not created: %v", err)
	}
}

func TestRunController(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	writeFile(t, "routes/routes.go", `package routes
import (
    "embed"
    "net/http"
    "net/http/pprof"
    "monolith/controllers"
)
func registerRoutes(mux *http.ServeMux, staticFiles embed.FS) {
    staticFileServer := http.FileServer(http.FS(staticFiles))
    _ = staticFileServer
    // pprof routes
    mux.HandleFunc("GET /debug/pprof/", pprof.Index)
}`)
	os.MkdirAll("controllers", 0755)
	os.MkdirAll("views", 0755)

	if err := runController([]string{"widgets", "index", "show"}); err != nil {
		t.Fatalf("runController: %v", err)
	}
	if _, err := os.Stat("controllers/widgets_controller.go"); err != nil {
		t.Fatalf("controller file: %v", err)
	}
	if _, err := os.Stat("views/widgets_index.html.tmpl"); err != nil {
		t.Fatalf("index template: %v", err)
	}
	data, _ := os.ReadFile("routes/routes.go")
	if !strings.Contains(string(data), "GET /widgets") || !strings.Contains(string(data), "controllers.WidgetsCtrl.Show") {
		t.Fatalf("routes not updated: %s", string(data))
	}
	if _, err := os.Stat("controllers/widgets_controller_test.go"); err != nil {
		t.Fatalf("controller test: %v", err)
	}
}

func TestRunControllerAddsImport(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	writeFile(t, "routes/routes.go", `package routes
import (
    "embed"
    "net/http"
    "net/http/pprof"
)
func registerRoutes(mux *http.ServeMux, staticFiles embed.FS) {
    staticFileServer := http.FileServer(http.FS(staticFiles))
    _ = staticFileServer
    // pprof routes
    mux.HandleFunc("GET /debug/pprof/", pprof.Index)
}`)
	os.MkdirAll("controllers", 0755)
	os.MkdirAll("views", 0755)

	if err := runController([]string{"widgets", "index"}); err != nil {
		t.Fatalf("runController: %v", err)
	}
	data, _ := os.ReadFile("routes/routes.go")
	if !strings.Contains(string(data), "\"monolith/controllers\"") {
		t.Fatalf("import not added: %s", string(data))
	}
	if !strings.Contains(string(data), "GET /widgets") {
		t.Fatalf("route not added: %s", string(data))
	}
}

func setupBaseFiles(t *testing.T, dir string) {
	writeFile(t, filepath.Join(dir, "db/db.go"), `package db
import "monolith/models"
func Connect() {
    dbHandle.AutoMigrate(
        &models.User{},
    )
}`)
	writeFile(t, filepath.Join(dir, "routes/routes.go"), `package routes
import (
    "embed"
    "net/http"
    "net/http/pprof"
    "monolith/controllers"
)
func registerRoutes(mux *http.ServeMux, staticFiles embed.FS) {
    staticFileServer := http.FileServer(http.FS(staticFiles))
    _ = staticFileServer
    // pprof routes
    mux.HandleFunc("GET /debug/pprof/", pprof.Index)
}`)
	os.MkdirAll(filepath.Join(dir, "jobs"), 0755)
	os.MkdirAll(filepath.Join(dir, "models"), 0755)
	os.MkdirAll(filepath.Join(dir, "controllers"), 0755)
	os.MkdirAll(filepath.Join(dir, "views"), 0755)
	os.MkdirAll(filepath.Join(dir, "session"), 0755)
	os.MkdirAll(filepath.Join(dir, "middleware"), 0755)
}

func TestRunResource(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)
	setupBaseFiles(t, dir)

	if err := runResource([]string{"gadget"}); err != nil {
		t.Fatalf("runResource: %v", err)
	}
	if _, err := os.Stat("models/gadget.go"); err != nil {
		t.Fatalf("model: %v", err)
	}
	if _, err := os.Stat("controllers/gadgets_controller.go"); err != nil {
		t.Fatalf("controller: %v", err)
	}
}

func TestRunJob(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	writeFile(t, "models/jobs.go", `package models
type JobType int
const (
    JobTypePrint JobType = iota
)`)
	writeFile(t, "jobs/job_queue.go", `package jobs
import "monolith/models"
func init() {
    jobQueue.register(models.JobTypePrint, PrintJob)
}`)
	if err := runJob([]string{"Email"}); err != nil {
		t.Fatalf("runJob: %v", err)
	}
	if _, err := os.Stat("jobs/email_job.go"); err != nil {
		t.Fatalf("job file: %v", err)
	}
	if _, err := os.Stat("jobs/email_job_test.go"); err != nil {
		t.Fatalf("job test: %v", err)
	}
	data, _ := os.ReadFile("models/jobs.go")
	if !strings.Contains(string(data), "JobTypeEmail") {
		t.Fatalf("enum not updated: %s", string(data))
	}
	data, _ = os.ReadFile("jobs/job_queue.go")
	if !strings.Contains(string(data), "JobTypeEmail") {
		t.Fatalf("queue not updated: %s", string(data))
	}
}

func TestRunAuthentication(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)
	setupBaseFiles(t, dir)

	if err := runAuthentication([]string{}); err != nil {
		t.Fatalf("runAuthentication: %v", err)
	}
	files := []string{
		"models/user.go",
		"session/session.go",
		"session/session_test.go",
		"middleware/auth.go",
		"middleware/auth_test.go",
		"controllers/auth_controller.go",
		"views/login.html.tmpl",
		"views/signup.html.tmpl",
	}
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("missing %s", f)
		}
	}
	data, _ := os.ReadFile("routes/routes.go")
	if !strings.Contains(string(data), "/login") || !strings.Contains(string(data), "/signup") {
		t.Fatalf("routes not updated: %s", string(data))
	}
	db, _ := os.ReadFile("db/db.go")
	if !strings.Contains(string(db), "&models.User{}") {
		t.Fatalf("db not updated: %s", string(db))
	}
}
