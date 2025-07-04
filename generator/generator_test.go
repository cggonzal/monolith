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
import "monolith/app/models"
func InitDB() {
    dbHandle.AutoMigrate(
        &models.User{},
    )
}`)
	os.MkdirAll("app/models", 0755)

	if err := runModel([]string{"Widget", "name:string"}); err != nil {
		t.Fatalf("runModel: %v", err)
	}
	if _, err := os.Stat("app/models/widget.go"); err != nil {
		t.Fatalf("model file: %v", err)
	}
	data, _ := os.ReadFile("db/db.go")
	if !strings.Contains(string(data), "&models.Widget{}") {
		t.Fatalf("db not updated: %s", string(data))
	}
	if _, err := os.Stat("app/models/widget_test.go"); err != nil {
		t.Fatalf("model test not created: %v", err)
	}
}

func TestRunController(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	writeFile(t, "app/routes/routes.go", `package routes
import (
    "embed"
    "net/http"
    "monolith/app/controllers"
)
func registerRoutes(mux *http.ServeMux, staticFiles embed.FS) {
    staticFileServer := http.FileServer(http.FS(staticFiles))
    _ = staticFileServer
}`)
	os.MkdirAll("app/controllers", 0755)
	os.MkdirAll("app/views", 0755)

	if err := runController([]string{"widgets", "index", "show"}); err != nil {
		t.Fatalf("runController: %v", err)
	}
	if _, err := os.Stat("app/controllers/widgets_controller.go"); err != nil {
		t.Fatalf("controller file: %v", err)
	}
	if _, err := os.Stat(filepath.Join("app/views", "widgets", "widgets_index.html.tmpl")); err != nil {
		t.Fatalf("index template: %v", err)
	}
	data, _ := os.ReadFile("app/routes/routes.go")
	if !strings.Contains(string(data), "GET /widgets") || !strings.Contains(string(data), "controllers.WidgetsCtrl.Show") {
		t.Fatalf("routes not updated: %s", string(data))
	}
	if _, err := os.Stat("app/controllers/widgets_controller_test.go"); err != nil {
		t.Fatalf("controller test: %v", err)
	}
}

func TestRunControllerAddsImport(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	writeFile(t, "app/routes/routes.go", `package routes
import (
    "embed"
    "net/http"
)
func registerRoutes(mux *http.ServeMux, staticFiles embed.FS) {
    staticFileServer := http.FileServer(http.FS(staticFiles))
    _ = staticFileServer
}`)
	os.MkdirAll("app/controllers", 0755)
	os.MkdirAll("app/views", 0755)

	if err := runController([]string{"widgets", "index"}); err != nil {
		t.Fatalf("runController: %v", err)
	}
	data, _ := os.ReadFile("app/routes/routes.go")
	if !strings.Contains(string(data), "\"monolith/app/controllers\"") {
		t.Fatalf("import not added: %s", string(data))
	}
	if !strings.Contains(string(data), "GET /widgets") {
		t.Fatalf("route not added: %s", string(data))
	}
}

func setupBaseFiles(t *testing.T, dir string) {
	writeFile(t, filepath.Join(dir, "db/db.go"), `package db
import "monolith/app/models"
func InitDB() {
    dbHandle.AutoMigrate(
        &models.User{},
    )
}`)
	writeFile(t, filepath.Join(dir, "app/routes/routes.go"), `package routes
import (
    "embed"
    "net/http"
    "monolith/app/controllers"
)
func registerRoutes(mux *http.ServeMux, staticFiles embed.FS) {
    staticFileServer := http.FileServer(http.FS(staticFiles))
    _ = staticFileServer
}`)
	os.MkdirAll(filepath.Join(dir, "app/jobs"), 0755)
	os.MkdirAll(filepath.Join(dir, "app/models"), 0755)
	os.MkdirAll(filepath.Join(dir, "app/controllers"), 0755)
	os.MkdirAll(filepath.Join(dir, "app/views"), 0755)
	os.MkdirAll(filepath.Join(dir, "app/session"), 0755)
	os.MkdirAll(filepath.Join(dir, "app/middleware"), 0755)
	writeFile(t, filepath.Join(dir, "go.mod"), "module monolith\n\ngo 1.23\n")
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
	if _, err := os.Stat("app/models/gadget.go"); err != nil {
		t.Fatalf("model: %v", err)
	}
	if _, err := os.Stat("app/controllers/gadgets_controller.go"); err != nil {
		t.Fatalf("controller: %v", err)
	}
}

func TestRunJob(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	writeFile(t, "app/models/job.go", `package models
type JobType int
const (
    JobTypeExample JobType = iota
)`)
	writeFile(t, "app/jobs/job_queue.go", `package jobs
import "monolith/app/models"
func init() {
    jobQueue.register(models.JobTypeExample, ExampleJob)
}`)
	if err := runJob([]string{"Email"}); err != nil {
		t.Fatalf("runJob: %v", err)
	}
	if _, err := os.Stat("app/jobs/email_job.go"); err != nil {
		t.Fatalf("job file: %v", err)
	}
	if _, err := os.Stat("app/jobs/email_job_test.go"); err != nil {
		t.Fatalf("job test: %v", err)
	}
	data, _ := os.ReadFile("app/models/job.go")
	if !strings.Contains(string(data), "JobTypeEmail") {
		t.Fatalf("enum not updated: %s", string(data))
	}
	data, _ = os.ReadFile("app/jobs/job_queue.go")
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
		"app/models/user.go",
		"app/session/session.go",
		"app/models/user_test.go",
		"app/middleware/auth.go",
		"app/middleware/auth_test.go",
		"app/controllers/auth_controller.go",
		filepath.Join("app/views", "auth", "auth_login.html.tmpl"),
		filepath.Join("app/views", "auth", "auth_signup.html.tmpl"),
	}
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("missing %s", f)
		}
	}
	data, _ := os.ReadFile("app/routes/routes.go")
	if !strings.Contains(string(data), "/login") || !strings.Contains(string(data), "/signup") {
		t.Fatalf("routes not updated: %s", string(data))
	}
	db, _ := os.ReadFile("db/db.go")
	if !strings.Contains(string(db), "&models.User{}") {
		t.Fatalf("db not updated: %s", string(db))
	}
	userModel, _ := os.ReadFile("app/models/user.go")
	if !strings.Contains(string(userModel), "BeforeSave") || !strings.Contains(string(userModel), "AfterSave") {
		t.Fatalf("hooks not added: %s", string(userModel))
	}
	if !strings.Contains(string(userModel), "SetLoggedIn") || !strings.Contains(string(userModel), "IsLoggedIn") {
		t.Fatalf("session helpers not added: %s", string(userModel))
	}
	modData, _ := os.ReadFile("go.mod")
	if !strings.Contains(string(modData), "github.com/gorilla/sessions") || !strings.Contains(string(modData), "golang.org/x/crypto") {
		t.Fatalf("go.mod not updated: %s", string(modData))
	}
}

func TestRunAdmin(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)
	setupBaseFiles(t, dir)

	if err := runAdmin([]string{}); err != nil {
		t.Fatalf("runAdmin: %v", err)
	}
	files := []string{
		"app/models/user.go",
		"app/controllers/admin_controller.go",
		filepath.Join("app/views", "admin", "admin_dashboard.html.tmpl"),
		"app/middleware/admin.go",
		"app/middleware/admin_test.go",
		"app/session/email.go",
	}
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("missing %s", f)
		}
	}
	data, _ := os.ReadFile("app/routes/routes.go")
	if !strings.Contains(string(data), "GET /admin") || !strings.Contains(string(data), "POST /admin") {
		t.Fatalf("route not added: %s", string(data))
	}
	if !strings.Contains(string(data), "middleware.RequireAdmin(pprof.Index)") {
		t.Fatalf("pprof routes not added: %s", string(data))
	}
	modData, _ := os.ReadFile("go.mod")
	if !strings.Contains(string(modData), "github.com/gorilla/sessions") || !strings.Contains(string(modData), "golang.org/x/crypto") {
		t.Fatalf("go.mod not updated: %s", string(modData))
	}
}
