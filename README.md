# Monolith Documentation

[![Monolith](https://github.com/cggonzal/monolith/actions/workflows/ci.yml/badge.svg)](https://github.com/cggonzal/monolith/actions/workflows/ci.yml)

Welcome to the **Monolith** reference guide.

This document explains every major subsystem of the project and shows how the pieces fit together.

If you are new, start with **Quick‑start** then come back to read the architecture chapters.


---

## Table of Contents

1. [Introduction](#introduction)
2. [Quickstart](#quickstart)
3. [Request Flow](#request-flow)
4. [Practical Walk‑throughs](#practical-walkthroughs)
   1. [Authentication flow](#authentication-flow-example)
   2. [WebSocket chat](#websocket-chat-example)
   3. [Background job](#background-job-example)
   4. [Interactive debug session](#interactive-debug-session)
   5. [Generating a job](#generating-a-job)
   6. [Generating a resource](#generating-a-resource)
   7. [Generating authentication](#generating-authentication)
   8. [Generating an admin dashboard](#generating-an-admin-dashboard)
   9. [Generating a model](#generating-a-model)
   10. [Generating a controller and view](#generating-a-controller-and-view)
5. [Core Concepts](#core-concepts)
   1. [Configuration](#configuration)
   2. [Database Layer](#database-layer)
   3. [Domain Models](#domain-models)
   4. [Sessions & Authentication](#sessions--authentication)
   5. [CSRF Protection](#csrf-protection)
   6. [Middleware](#middleware)
   7. [Routing & HTTP controllers](#routing--http-controllers)
   8. [Templates & Static Assets](#templates--static-assets)
   9. [WebSockets](#websockets)
   10. [Job Queue](#job-queue)
   11. [Server Management & Zero‑downtime Deploys](#server-management--zero-downtime-deploys)
   12. [Debugging & Profiling](#debugging--profiling)
6. [Project Layout](#project-layout)
7. [Extending the Monolith](#extending-the-monolith)
8. [Generators](#generators)
9. [Testing](#testing)
10. [Development](#development)
11. [Server Setup](#server-setup)
12. [Deployment](#deployment)
13. [Appendix](#appendix)

---

## Introduction

**Monolith** is a full‑stack Go application that demonstrates:

* Cookie‑based sessions with built‑in login
* GORM‑powered persistence (SQLite by default) 
* Zero downtime deploys
* A tiny background job queue  
* Real‑time WebSocket messaging  
* Structured logging, graceful shutdown, & systemd socket activation  
* Embedded templates and static assets  
* Built‑in performance profiling with the standard library  

Everything uses the **Go standard library** with a handful of small, focused dependencies:

| Purpose | Package |
| --- | --- |
| Database driver | `github.com/glebarez/sqlite` |
| Sessions | `github.com/gorilla/sessions` |
| ORM | `gorm.io/gorm` |
| WebSocket library | `github.com/gorilla/websocket` |
| Singular/plural helpers | `github.com/jinzhu/inflection` |

---

## Quickstart

```bash
# 1. clone & enter
git clone <repo> && cd monolith

# 2. start the server (uses air if available)
make       # hot reload during development

# or without air installed
make run

# open http://localhost:9000
```

Set the `SECRET_KEY` environment variable to a random string before running the server.

The first launch creates **app.db** and auto‑migrates the schema.

---

## Request Flow

```mermaid
flowchart TD
    A[Client Request] -->|1| B[Caddy proxy]
    B -->|2| C[App :9000]
    C -->|3| D[Router]
    D -->|4| E[Controller]
    E -- "5 if needed" --> F[(DB)]
    F -->|6| E
    E -->|7| G[Render View]
    G -->|8| B
    B -->|9| H[Encode / gzip]
    H -->|10| I[Client Response]
```

---

## Practical Walk‑throughs

### Authentication flow Example

1. Visit `/signup` and create an account
2. On success you’re logged in and redirected to `/`
3. Existing users go to `/login` with their credentials
4. A cookie named `session` tracks login state

Use `/logout` to clear the session

### WebSocket chat Example

```html
<script>
const sock = new WebSocket("ws://localhost:9000/ws");
sock.onopen = () => {
  sock.send(JSON.stringify({command: "subscribe", identifier: "ChatChannel"}));
  sock.send(JSON.stringify({command: "message", identifier: "ChatChannel", data: "Hello from JS!"}));
};
sock.onmessage = ev => console.log("got:", ev.data);
</script>
```

All messages are persisted and broadcast to every subscriber of `chat`.

### Background job Example

```go
payload := []byte(`{"message":"Hello"}`)
jobs.GetJobQueue().AddJob(models.JobTypePrint, payload)
```

To schedule a recurring job using a cron expression:

```go
payload := []byte(`{"message":"Hello"}`)
jobs.GetJobQueue().AddRecurringJob(models.JobTypePrint, payload, "0 0 * * *")
```

`app/jobs/job_queue.go` registers job handlers and the queue starts automatically.

### Interactive debug session

```bash
# in one terminal
go run .            # start app

# in another (requires an admin account and the admin generator)
curl http://localhost:9000/debug/pprof/heap > heap.out
go tool pprof heap.out
```

### Generating a job

```bash
make generator job Email
```

The command above creates:

* `app/jobs/email_job.go`
* `app/jobs/email_job_test.go`
* `app/models/job.go` (adds `JobTypeEmail`)
* `app/jobs/job_queue.go` (registers the job)

Inside `app/jobs/email_job.go` you will find a stubbed function to implement:

```go
func EmailJob(payload []byte) error {
    var p EmailPayload
    if err := json.Unmarshal(payload, &p); err != nil {
        return err
    }

    // TODO: implement job

    return nil
}
```

### Generating a resource

```bash
make generator resource widget name:string price:int
```

This creates the model and the full set of REST pieces:

* `app/models/widget.go` and `app/models/widget_test.go`
* `db/db.go` updated with the new model
* `app/controllers/widgets_controller.go` and test file
* templates under `app/views/widgets/` for `index`, `show`, `new` and `edit`
* routes injected into `app/routes/routes.go`:
  ```go
  mux.HandleFunc("GET /widgets", controllers.WidgetsCtrl.Index)
  mux.HandleFunc("GET /widgets/new", controllers.WidgetsCtrl.New)
  mux.HandleFunc("POST /widgets", controllers.WidgetsCtrl.Create)
  mux.HandleFunc("GET /widgets/{id}", controllers.WidgetsCtrl.Show)
  mux.HandleFunc("GET /widgets/{id}/edit", controllers.WidgetsCtrl.Edit)
  mux.HandleFunc("PUT /widgets/{id}", controllers.WidgetsCtrl.Update)
  mux.HandleFunc("PATCH /widgets/{id}", controllers.WidgetsCtrl.Update)
  mux.HandleFunc("DELETE /widgets/{id}", controllers.WidgetsCtrl.Destroy)
  ```

The generated controller functions contain placeholders, for example the index
action:

```go
func (c *WidgetsController) Index(w http.ResponseWriter, r *http.Request) {
    records, _ := models.GetAllWidgets(db.GetDB())
    views.Render(w, "widgets_index.html.tmpl", records)
}
```

Each template is a basic skeleton ready to be filled in:

```html
{{define "title"}}<title></title>{{end}}

{{define "body"}}
{{end}}
```

### Generating authentication

```bash
make generator authentication
```

Scaffolds a `User` model with session helpers, login & signup templates, and authentication middleware.
The generator also injects the following routes:
```go
mux.HandleFunc("GET /login", controllers.AuthCtrl.ShowLoginForm)
mux.HandleFunc("POST /login", controllers.AuthCtrl.Login)
mux.HandleFunc("GET /signup", controllers.AuthCtrl.ShowSignupForm)
mux.HandleFunc("POST /signup", controllers.AuthCtrl.Signup)
mux.HandleFunc("GET /logout", controllers.AuthCtrl.Logout)
```

### Generating an admin dashboard

```bash
make generator admin
```

Creates an `/admin` dashboard with profiling helpers. If a User model does not
exist it will be generated automatically.
The generator also wires up routes for the dashboard and pprof:
```go
mux.HandleFunc("GET /admin", middleware.RequireAdmin(controllers.AdminCtrl.Dashboard))
mux.HandleFunc("POST /admin", middleware.RequireAdmin(controllers.AdminCtrl.Dashboard))
// pprof routes
mux.HandleFunc("GET /debug/pprof/", middleware.RequireAdmin(pprof.Index))
mux.HandleFunc("GET /debug/pprof/cmdline", middleware.RequireAdmin(pprof.Cmdline))
mux.HandleFunc("GET /debug/pprof/profile", middleware.RequireAdmin(pprof.Profile))
mux.HandleFunc("GET /debug/pprof/symbol", middleware.RequireAdmin(pprof.Symbol))
mux.HandleFunc("GET /debug/pprof/trace", middleware.RequireAdmin(pprof.Trace))
```

### Generating a model

```bash
make generator model Widget name:string price:int
```

Files created:

* `app/models/widget.go`
* `app/models/widget_test.go`
* `db/db.go` updated to migrate the model

The generated model file defines blank GORM hooks to customise later:

```go
// BeforeSave is called by GORM before persisting a Widget.
func (m *Widget) BeforeSave(tx *gorm.DB) error {
    return nil
}
```

### Generating a controller and view

```bash
make generator controller widgets index show
```

This will generate:

* `app/controllers/widgets_controller.go`
* `app/controllers/widgets_controller_test.go`
* templates `app/views/widgets/widgets_index.html.tmpl` and
  `app/views/widgets/widgets_show.html.tmpl`
* route entries in `app/routes/routes.go`:
  ```go
  mux.HandleFunc("GET /widgets", controllers.WidgetsCtrl.Index)
  mux.HandleFunc("GET /widgets/{id}", controllers.WidgetsCtrl.Show)
  ```

The controller skeleton looks like:

```go
func (c *WidgetsController) Index(w http.ResponseWriter, r *http.Request) {
    views.Render(w, "widgets_index.html.tmpl", nil)
}
```

And the templates start with an empty body block ready for content:

```html
{{define "body"}}
{{end}}
```

---



## Core Concepts
### Configuration

`app/config/config.go` contains **constants** that rarely change at runtime, e.g.

```go
const JOB_QUEUE_NUM_WORKERS = 4
```

Everything dynamic (port and database DSN) is read from **environment variables** inside `main.go` or the relevant package:

| Variable | Default | Used in |
| -------- | ------- | ------- |
| `PORT` | `9000` | HTTP listener |
| `LISTEN_FDS`, `LISTEN_PID` | – | systemd socket activation |

### Database Layer

`db/db.go` initialises a GORM connection:

```go
dbHandle, err = gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
```

Switching to Postgres is one line:

```go
// import "gorm.io/driver/postgres"
gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})
```

`db.InitDB()` runs **auto‑migration** for every registered model so your schema stays in sync.

### Domain Models

| Model | File | Purpose |
| ----- | ---- | ------- |
| `User` | `app/models/user.go` | Registered users (email, avatar, flags) |
| `Job`  | `app/models/job.go` | Background work unit with `Type` & `Status` enums |
| `Message` | `app/models/ws.go` | Persisted WebSocket chat message |

All models embed GORM timestamps, so you automatically get `CreatedAt` / `UpdatedAt`.

Generated models also include blank `BeforeSave` and `AfterSave` hooks. GORM
automatically invokes these methods before and after a record is persisted, so
you can implement validation or post‑processing logic as needed.

Example: Creating a user

```go
user, _ := models.CreateUser(db.GetDB(), "foo@example.com", "secret")
```

### Sessions & Authentication

Session helpers live in `app/session/session.go`:

* **SecureCookie** store (`gorilla/sessions`)
* `SetLoggedIn`, `Logout`, `IsLoggedIn`

Authentication flow: browser posts credentials to `/login` which validates the
password and redirects to `/` on success.

If `session.IsLoggedIn(r)` is **false**, the `middleware.RequireLogin` decorator redirects the request to `/login`.

### CSRF Protection

`app/csrf/` provides helpers to embed a CSRF token into HTML forms or expose it to JavaScript. `app/middleware/CSRFMiddleware` verifies the token on every mutating request and returns **403 Forbidden** if it is missing or invalid.

Use `csrf.GetCSRFTokenForForm` inside your controllers when rendering templates:

```go
data := map[string]any{
    "csrf_token": csrf.GetCSRFTokenForForm(w, r),
}
templates.ExecuteTemplate(w, "form.html.tmpl", data)
```

In the template simply output `{{.csrf_token}}` inside the `<form>`:

```html
<form method="POST" action="/items">
    {{.csrf_token}}
    <!-- rest of fields -->
</form>
```

For AJAX requests include the meta tag returned by `csrf.GetCSRFMetaTag` and send the token in the `X-CSRF-Token` header:

```go
data := map[string]any{
    "csrf_meta": csrf.GetCSRFMetaTag(w, r),
}
templates.ExecuteTemplate(w, "index.html.tmpl", data)
```

```html
<head>
    {{.csrf_meta}}
</head>
<script>
const token = document.querySelector('meta[name="csrf-token"]').content;
fetch('/items', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-CSRF-Token': token,
  },
  body: JSON.stringify({name: 'foo'}),
});
</script>
```

### Middleware

Three middlewares are shipped:

| File | Function | Description |
| ---- | -------- | ----------- |
| `app/middleware/logging.go` | `LoggingMiddleware` | Structured request log using `log/slog` |
| `app/middleware/auth.go` | `RequireLogin` | Gate routes behind authentication |
| `app/middleware/csrf.go` | `CSRFMiddleware` | Validate CSRF token for unsafe requests |

Compose them like:

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /dashboard", middleware.RequireLogin(controllers.Dashboard))
handler := middleware.CSRFMiddleware(middleware.LoggingMiddleware(mux))
http.ListenAndServe(":9000", handler)
```

### Routing & HTTP controllers

All controllers are in `app/controllers/` and are wired inside `main.go` using the **new routing syntax** (Go 1.23+):

```go
mux.HandleFunc("GET /", controllers.Home)
mux.HandleFunc("POST /items/new", controllers.CreateItemHandler)
```

Templates are parsed once during startup through `views.InitTemplates(embed.FS)` giving you the full power of Go’s `html/template`.

### Templates & Static Assets

Assets live beside code but are **embedded** thanks to the `embed` package:

```go
//go:embed static/*
var staticFiles embed.FS

//go:embed app/views/**
var templateFiles embed.FS
```

* `static/` is served under `/static/…`
* `app/views/*.html.tmpl` are executed server‑side

This makes the final binary self‑contained & easy to deploy.

### WebSockets

`ws/` provides a lightweight **publish/subscribe** layer:

* `Hub` – single central switchboard created at startup
* `Client` – represents one browser connection
* Messages are JSON encoded and stored in the DB for history.  Broadcasting is
  done concurrently so thousands of clients can be serviced with minimal delay.

Upgrading a request to WebSocket:

```go
func HandleWS(hub *ws.Hub) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ws.ServeWs(hub, w, r) // handles upgrade & registration
    }
}
```

Broadcast from anywhere:

```go
hub.Broadcast("chat", []byte("Hello, world!"))
```
`Broadcast` is safe to call from any goroutine and fans the message out to
subscribers concurrently.

### Job Queue

`jobs/` is a minimal in‑process queue with workers:

```go
jobs.RegisterHandler(models.JobTypePrint, func(j *models.Job) error {
    fmt.Println(string(j.Payload))
    return nil
})
jobs.GetJobQueue().AddJob(models.JobTypePrint,
    []byte(`{"message":"Hello background!"}`))
```

Features:

* FIFO ordering backed by the `jobs` DB table
* Automatic retries & exponential back‑off (see `JobQueue.process()`)
* Configurable workers via `config.JOB_QUEUE_NUM_WORKERS`
* Recurring jobs with `AddRecurringJob`

### Email Package

The `email` package provides a single `SendEmail` helper that enqueues an
email‑sending job. Emails are delivered asynchronously through Mailgun using
the REST API. Example:

```go
err := email.SendEmail(
    "Hello",
    "Welcome to the app!",
    "no-reply@example.com",
    []string{"user@example.com"},
)
if err != nil {
    log.Println("unable to queue email:", err)
}
```

Set the `MAILGUN_DOMAIN` and `MAILGUN_API_KEY` environment variables so the job
workers can talk to Mailgun.

### Server Management & Zero‑downtime Deploys

`server_management/` abstracts **systemd socket activation**:

* `SdListeners()` fetches inherited file descriptors.  
* `SdNotifyReady()` (see `main.go`) tells systemd we reached *READY*.

The `deploy.sh` and `server_setup.sh` scripts show how to:

1. Build the binary with `make build`
2. Upload & atomically switch `/opt/monolith/current -> new`
3. `systemctl restart monolith.service` (systemd sends **SIGTERM** by default)

Because the listener is handed over, the old process finishes in‑flight requests while the new one starts accepting immediately → **zero downtime**.

### Debugging & Profiling

After running `make generator admin` and creating an admin user, the `/debug/pprof/*` routes become available:

```
GET /debug/pprof/
GET /debug/pprof/profile   # CPU profile
GET /debug/pprof/heap      # Heap snapshot
```

Example CPU profile for 30 s:

```bash
go tool pprof http://localhost:9000/debug/pprof/profile?seconds=30
```

Debugging the application with **Visual Studio Code** is also supported. Open
the project in VS Code and use the `Launch Package` configuration provided in
`.vscode/launch.json` to run the server under the debugger.

---
## Project Layout

```
.
├── main.go                  # Program entry‑point
├── app/
│   ├── config/             # Compile‑time configuration knobs
│   ├── controllers/        # HTTP controllers (HTML + auth callbacks)
│   ├── middleware/         # Reusable HTTP middleware
│   ├── session/            # Session helpers
│   ├── csrf/               # CSRF helpers
│   ├── routes/             # Route definitions
│   ├── services/           # Business logic helpers
│   ├── jobs/               # Simple in‑process job queue
│   ├── models/             # GORM models (User, Job, Message)
│   └── views/              # `embed`ded HTML templates
├── db/                      # DB connection bootstrap
│   └── db.go
├── ws/                      # WebSocket hub, client & message types
├── static/                  # `embed`ded public files
├── server_management/       # systemd helpers + deployment scripts
└── tests, Makefile, etc.
```


---

## Extending the Monolith

### Adding a new Service Layer module

Create `services/email.go`:

```go
package services

func SendWelcome(to string) error {
    // …
}
```

Import and call it from controllers or jobs – services keep business logic away from HTTP glue.

### New Job Type

Use the generator to scaffold a job:

```bash
make generator job Email
```

This creates `app/jobs/email_job.go` with a stub `EmailJob` function, registers it
in `app/jobs/job_queue.go` and adds `JobTypeEmail` to `app/models/job.go`.

### Custom WebSocket Channel

```go
hub.Subscribe(client, "notifications")
hub.Broadcast("notifications", []byte(`{"title":"Build finished"}`))
```

---

## Generators

Generators scaffold common pieces of the application. They can be run through
the main program or via `make`:

```bash
go run main.go generator <type> [...options]
# or
make generator <type> [...options]
```

Supported types are `model`, `controller`, `resource`, `authentication`, `job` and `admin`.

### Model

```bash
make generator model Widget name:string price:int
```

Creates `app/models/widget.go` with a `Widget` struct and updates `db/db.go` so the
model is automatically migrated. The file also defines empty `BeforeSave` and
`AfterSave` hooks which you can use to validate your model before and after it
is saved.

### Controller

Controllers are typically named using the plural form:

```bash
make generator controller widgets index show
```

This generates `app/controllers/widgets_controller.go`, inserts matching routes into
`app/routes/routes.go` and creates templates like `app/views/widgets/widgets_index.html.tmpl`.
Example routes when generating `index` and `show` actions:
```go
mux.HandleFunc("GET /widgets", controllers.WidgetsCtrl.Index)
mux.HandleFunc("GET /widgets/{id}", controllers.WidgetsCtrl.Show)
```

### Resource

The resource generator produces a model and a full REST controller in one step.
Pass the **singular** name; the controller and routes will be pluralised.

```bash
make generator resource widget name:string price:int
```

This creates the model, a `widgets` controller with all CRUD actions, placeholder
tests and templates, and RESTful routes under `/widgets`.
The following routes are injected:
```go
mux.HandleFunc("GET /widgets", controllers.WidgetsCtrl.Index)
mux.HandleFunc("GET /widgets/new", controllers.WidgetsCtrl.New)
mux.HandleFunc("POST /widgets", controllers.WidgetsCtrl.Create)
mux.HandleFunc("GET /widgets/{id}", controllers.WidgetsCtrl.Show)
mux.HandleFunc("GET /widgets/{id}/edit", controllers.WidgetsCtrl.Edit)
mux.HandleFunc("PUT /widgets/{id}", controllers.WidgetsCtrl.Update)
mux.HandleFunc("PATCH /widgets/{id}", controllers.WidgetsCtrl.Update)
mux.HandleFunc("DELETE /widgets/{id}", controllers.WidgetsCtrl.Destroy)
```

### Authentication

```bash
make generator authentication
```

Generates a basic user model, session management and routes for user signup,
login and logout.
Routes added:
```go
mux.HandleFunc("GET /login", controllers.AuthCtrl.ShowLoginForm)
mux.HandleFunc("POST /login", controllers.AuthCtrl.Login)
mux.HandleFunc("GET /signup", controllers.AuthCtrl.ShowSignupForm)
mux.HandleFunc("POST /signup", controllers.AuthCtrl.Signup)
mux.HandleFunc("GET /logout", controllers.AuthCtrl.Logout)
```

### Job

```bash
make generator job MyJob
```

Creates `app/jobs/my_job_job.go` with a stub `MyJobJob` function, registers it in
`app/jobs/job_queue.go` and adds `JobTypeMyJob` to `app/models/job.go`.

### Admin

```bash
make generator admin
```

Scaffolds an `/admin` dashboard for profiling and wraps it in admin-only
middleware. If no `User` model exists it will be generated along with the
authentication pieces. It registers the following routes:
```go
mux.HandleFunc("GET /admin", middleware.RequireAdmin(controllers.AdminCtrl.Dashboard))
mux.HandleFunc("POST /admin", middleware.RequireAdmin(controllers.AdminCtrl.Dashboard))
mux.HandleFunc("GET /debug/pprof/", middleware.RequireAdmin(pprof.Index))
mux.HandleFunc("GET /debug/pprof/cmdline", middleware.RequireAdmin(pprof.Cmdline))
mux.HandleFunc("GET /debug/pprof/profile", middleware.RequireAdmin(pprof.Profile))
mux.HandleFunc("GET /debug/pprof/symbol", middleware.RequireAdmin(pprof.Symbol))
mux.HandleFunc("GET /debug/pprof/trace", middleware.RequireAdmin(pprof.Trace))
```

---

## Testing

Run the unit tests by running following in the root of the repo:

```bash
make test
```

`app/controllers/controllers_test.go` shows how to spin up an in‑memory HTTP server and assert redirects.

---
## Development
If you have `air` installed, then you can start a development server with hot reloading by running the following in the root of the repo:
```
make
```

Otherwise, just run the app with:
```
make run
```

You can also create a standalone binary with:
```bash
make build
```
---
## Server Setup
Assuming you have a newly created ubuntu server that you have ssh access into, just run:
```bash
make server-setup root@{{ip address of server}}
```

Edit `server_management/Caddyfile` with your domain and any desired tweaks before running the setup.

For example,
```bash
make server-setup root@203.0.113.5
```

---

## Deployment


Run the following from the root of the repo:

`make deploy {{ip address of server}}`

where `ip address of server` is the hostname and IP address of your server.

For example,
```bash
make deploy root@203.0.113.5
```

This will do a zero downtime deploy by calling,
```bash
./server_management/deploy.sh
```
By default the script prunes old releases after deployment. Set `PRUNE=false` to skip pruning.

---

## Appendix

### Environment Variables

| Name | Description | Default |
| ---- | ----------- | ------- |
| `PORT` | Fallback TCP port when not using socket activation | `9000` |
| `DATABASE_URL` | Postgres DSN (if you switch drivers) | – |
| `MAILGUN_DOMAIN` | Mailgun domain used for sending mail | – |
| `MAILGUN_API_KEY` | Private API key for Mailgun | – |
| `SECRET_KEY` | Key used to sign session cookies; must be set to a random string | – |

### Make Targets

| Command | Effect |
| ------- | ------ |
| `make`       | Run a hot reloaded development server using `air`
| `make build` | Build a statically linked binary |
| `make run`   | `go run ./...` |
| `make test`  | `go test ./...` |
| `make clean` | Clear test cache |
| `make deploy`| Zero downtime deploy via server_management/deploy.sh

---