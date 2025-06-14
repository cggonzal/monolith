package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/jinzhu/inflection"
)

const helpMessage = `Usage: go run main.go generator <command> [arguments]

Available commands:
  model NAME [field:type...]    Create a model struct and update db migrations
  controller NAME [actions]     Create a controller, templates and routes
  resource NAME [field:type...] Create model and full REST controller
  authentication                Scaffold basic user authentication
  job NAME                      Create a skeleton background job

Examples:
  go run main.go generator model Widget name:string price:int
  go run main.go generator controller widgets index show
  go run main.go generator resource widget name:string price:int
  go run main.go generator authentication
  go run main.go generator job MyJob
`

// Run dispatches to the specific generator based on args.
// args should not include the leading "generator" argument.
func Run(args []string) error {
	if len(args) == 0 {
		fmt.Print(helpMessage)
		return errors.New("missing generator type")
	}

	switch args[0] {
	case "help":
		fmt.Print(helpMessage)
		return nil
	case "model":
		return runModel(args[1:])
	case "controller":
		return runController(args[1:])
	case "resource":
		return runResource(args[1:])
	case "authentication":
		return runAuthentication(args[1:])
	case "job":
		return runJob(args[1:])
	default:
		fmt.Print(helpMessage)
		return fmt.Errorf("unknown generator: %s", args[0])
	}
}

// runModel creates a new model struct and updates db migrations.
func runModel(args []string) error {
	if len(args) == 0 {
		return errors.New("model name required")
	}
	modelName := args[0]
	fields := args[1:]

	if err := createModelFile(modelName, fields); err != nil {
		return err
	}
	if err := updateDBFile(modelName); err != nil {
		return err
	}
	if err := createModelTestFile(modelName); err != nil {
		return err
	}
	return nil
}

// runController creates a new controller with optional REST actions and updates routes and templates.
func runController(args []string) error {
	if len(args) == 0 {
		return errors.New("controller name required")
	}
	name := args[0]
	actions := args[1:]
	if len(actions) == 1 && actions[0] == "all" {
		actions = []string{"index", "show", "new", "create", "edit", "update", "destroy"}
	}

	if err := createControllerFile(name, actions); err != nil {
		return err
	}
	if len(actions) > 0 {
		if err := updateRoutesFile(name, actions); err != nil {
			return err
		}
	}
	if err := createTemplateFiles(name, actions); err != nil {
		return err
	}
	if err := createControllerTestFile(name); err != nil {
		return err
	}
	return nil
}

// runResource generates a model, a controller with REST actions,
// associated templates, routes and placeholder tests.
func runResource(args []string) error {
	if len(args) == 0 {
		return errors.New("resource name required")
	}
	name := args[0]
	fields := args[1:]

	// create the model (singular name)
	if err := runModel(append([]string{name}, fields...)); err != nil {
		return err
	}

	// controller uses pluralized name
	ctrlName := inflection.Plural(name)
	if err := runController([]string{ctrlName, "all"}); err != nil {
		return err
	}

	return nil
}

// runJob scaffolds a new background job function and registers it.
func runJob(args []string) error {
	if len(args) == 0 {
		return errors.New("job name required")
	}
	name := toCamelCase(args[0])
	if err := createJobFunction(name); err != nil {
		return err
	}
	if err := updateJobTypeEnum(name); err != nil {
		return err
	}
	if err := registerJobInQueue(name); err != nil {
		return err
	}
	return nil
}

// runAuthentication scaffolds user authentication helpers, controller,
// templates and routes.
func runAuthentication(args []string) error {
	if len(args) != 0 {
		return errors.New("authentication generator takes no arguments")
	}
	if err := createUserModelAuth(); err != nil {
		return err
	}
	if err := createSessionFile(); err != nil {
		return err
	}
	if err := createSessionTestFile(); err != nil {
		return err
	}
	if err := createAuthMiddlewareFile(); err != nil {
		return err
	}
	if err := createAuthMiddlewareTestFile(); err != nil {
		return err
	}
	if err := createAuthControllerFile(); err != nil {
		return err
	}
	if err := createLoginTemplate(); err != nil {
		return err
	}
	if err := createSignupTemplate(); err != nil {
		return err
	}
	if err := updateRoutesForAuth(); err != nil {
		return err
	}
	return nil
}

// createModelFile generates the model struct file in models/ directory.
func createModelFile(modelName string, fields []string) error {
	fileName := toSnakeCase(modelName) + ".go"
	path := filepath.Join("models", fileName)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}

	var buf bytes.Buffer
	buf.WriteString("package models\n\n")
	buf.WriteString("import \"gorm.io/gorm\"\n\n")
	buf.WriteString(fmt.Sprintf("type %s struct {\n", modelName))
	buf.WriteString("\tgorm.Model\n")

	for _, f := range fields {
		parts := strings.SplitN(f, ":", 2)
		name := parts[0]
		typ := "string"
		if len(parts) > 1 {
			typ = parts[1]
		}
		buf.WriteString(fmt.Sprintf("\t%s %s\n", toCamelCase(name), typ))
	}
	buf.WriteString("}\n")

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("create", path)
	return nil
}

// updateDBFile injects the new model into db/db.go AutoMigrate list.
func updateDBFile(modelName string) error {
	path := filepath.Join("db", "db.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.Contains(line, "AutoMigrate(") {
			// find closing line
			j := i + 1
			for ; j < len(lines); j++ {
				if strings.Contains(lines[j], ")") {
					break
				}
			}
			indent := leadingWhitespace(lines[j])
			insert := fmt.Sprintf("%s&models.%s{},", indent+"\t", modelName)
			// check if already inserted
			for k := i + 1; k < j; k++ {
				if strings.Contains(lines[k], "&models."+modelName+"{}") {
					fmt.Println("exists", path)
					return nil
				}
			}
			lines = append(lines[:j], append([]string{insert}, lines[j:]...)...)
			break
		}
	}

	out := strings.Join(lines, "\n")
	formatted, err := format.Source([]byte(out))
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("update", path)
	return nil
}

// createControllerFile generates a controller with the specified actions.
func createControllerFile(name string, actions []string) error {
	file := toSnakeCase(name) + "_controller.go"
	path := filepath.Join("controllers", file)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}

	ctrlName := toCamelCase(name) + "Controller"
	varName := toCamelCase(name) + "Ctrl"
	modelName := toCamelCase(inflection.Singular(toSnakeCase(name)))
	needDB := false
	needTemplates := false
	for _, a := range actions {
		switch a {
		case "index", "show", "create", "edit", "update", "destroy":
			needDB = true
		}
		switch a {
		case "index", "show", "new", "edit":
			needTemplates = true
		}
	}

	var buf bytes.Buffer
	buf.WriteString("package controllers\n\n")

	imports := []string{"\"net/http\""}
	if needTemplates {
		imports = append(imports, "\"monolith/templates\"")
	}
	if needDB {
		imports = append(imports, "\"monolith/db\"", "\"monolith/models\"")
	}
	if len(imports) > 1 {
		buf.WriteString("import (\n")
		for _, imp := range imports {
			buf.WriteString("\t" + imp + "\n")
		}
		buf.WriteString(")\n\n")
	} else {
		buf.WriteString("import " + imports[0] + "\n\n")
	}

	buf.WriteString(fmt.Sprintf("type %s struct{}\n\n", ctrlName))
	buf.WriteString(fmt.Sprintf("var %s = &%s{}\n\n", varName, ctrlName))

	for _, act := range actions {
		switch act {
		case "index":
			buf.WriteString(fmt.Sprintf("func (c *%s) Index(w http.ResponseWriter, r *http.Request) {\n", ctrlName))
			buf.WriteString(fmt.Sprintf("\tvar records []models.%s\n", modelName))
			buf.WriteString("\tdb.GetDB().Find(&records)\n")
			buf.WriteString(fmt.Sprintf("\ttemplates.ExecuteTemplate(w, \"%s_index.html.tmpl\", records)\n", toSnakeCase(name)))
			buf.WriteString("}\n\n")
		case "show":
			buf.WriteString(fmt.Sprintf("func (c *%s) Show(w http.ResponseWriter, r *http.Request) {\n", ctrlName))
			buf.WriteString("\tid := r.PathValue(\"id\")\n")
			buf.WriteString(fmt.Sprintf("\tvar record models.%s\n", modelName))
			buf.WriteString("\tdb.GetDB().First(&record, id)\n")
			buf.WriteString(fmt.Sprintf("\ttemplates.ExecuteTemplate(w, \"%s_show.html.tmpl\", record)\n", toSnakeCase(name)))
			buf.WriteString("}\n\n")
		case "new":
			buf.WriteString(fmt.Sprintf("func (c *%s) New(w http.ResponseWriter, r *http.Request) {\n", ctrlName))
			buf.WriteString(fmt.Sprintf("\ttemplates.ExecuteTemplate(w, \"%s_new.html.tmpl\", nil)\n", toSnakeCase(name)))
			buf.WriteString("}\n\n")
		case "create":
			buf.WriteString(fmt.Sprintf("func (c *%s) Create(w http.ResponseWriter, r *http.Request) {\n", ctrlName))
			buf.WriteString(fmt.Sprintf("\tvar record models.%s\n", modelName))
			buf.WriteString("\t// TODO: parse form values into &record\n")
			buf.WriteString("\tdb.GetDB().Create(&record)\n")
			buf.WriteString(fmt.Sprintf("\thttp.Redirect(w, r, \"/%s\", http.StatusSeeOther)\n", toSnakeCase(name)))
			buf.WriteString("}\n\n")
		case "edit":
			buf.WriteString(fmt.Sprintf("func (c *%s) Edit(w http.ResponseWriter, r *http.Request) {\n", ctrlName))
			buf.WriteString("\tid := r.PathValue(\"id\")\n")
			buf.WriteString(fmt.Sprintf("\tvar record models.%s\n", modelName))
			buf.WriteString("\tdb.GetDB().First(&record, id)\n")
			buf.WriteString(fmt.Sprintf("\ttemplates.ExecuteTemplate(w, \"%s_edit.html.tmpl\", record)\n", toSnakeCase(name)))
			buf.WriteString("}\n\n")
		case "update":
			buf.WriteString(fmt.Sprintf("func (c *%s) Update(w http.ResponseWriter, r *http.Request) {\n", ctrlName))
			buf.WriteString("\tid := r.PathValue(\"id\")\n")
			buf.WriteString(fmt.Sprintf("\tvar record models.%s\n", modelName))
			buf.WriteString("\tdb.GetDB().First(&record, id)\n")
			buf.WriteString("\t// TODO: update record fields\n")
			buf.WriteString("\tdb.GetDB().Save(&record)\n")
			buf.WriteString(fmt.Sprintf("\thttp.Redirect(w, r, \"/%s/\"+id, http.StatusSeeOther)\n", toSnakeCase(name)))
			buf.WriteString("}\n\n")
		case "destroy":
			buf.WriteString(fmt.Sprintf("func (c *%s) Destroy(w http.ResponseWriter, r *http.Request) {\n", ctrlName))
			buf.WriteString("\tid := r.PathValue(\"id\")\n")
			buf.WriteString(fmt.Sprintf("\tdb.GetDB().Delete(&models.%s{}, id)\n", modelName))
			buf.WriteString(fmt.Sprintf("\thttp.Redirect(w, r, \"/%s\", http.StatusSeeOther)\n", toSnakeCase(name)))
			buf.WriteString("}\n\n")
		}
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("create", path)
	return nil
}

// updateRoutesFile injects new routes for the controller actions.
func updateRoutesFile(name string, actions []string) error {
	path := filepath.Join("routes", "routes.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	insertIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "pprof routes") {
			insertIdx = i
			break
		}
	}
	if insertIdx == -1 {
		return fmt.Errorf("could not find insertion point in routes.go")
	}

	indent := leadingWhitespace(lines[insertIdx])
	ctrlVar := toCamelCase(name) + "Ctrl"
	ctrlName := toCamelCase(name) + "Controller"
	base := "/" + toSnakeCase(name)

	var newLines []string
	newLines = append(newLines, "")
	newLines = append(newLines, fmt.Sprintf("%s// routes for %s", indent, ctrlName))
	for _, act := range actions {
		switch act {
		case "index":
			newLines = append(newLines, fmt.Sprintf("%smux.HandleFunc(\"GET %s\", controllers.%s.Index)", indent, base, ctrlVar))
		case "show":
			newLines = append(newLines, fmt.Sprintf("%smux.HandleFunc(\"GET %s/{id}\", controllers.%s.Show)", indent, base, ctrlVar))
		case "new":
			newLines = append(newLines, fmt.Sprintf("%smux.HandleFunc(\"GET %s/new\", controllers.%s.New)", indent, base, ctrlVar))
		case "create":
			newLines = append(newLines, fmt.Sprintf("%smux.HandleFunc(\"POST %s\", controllers.%s.Create)", indent, base, ctrlVar))
		case "edit":
			newLines = append(newLines, fmt.Sprintf("%smux.HandleFunc(\"GET %s/{id}/edit\", controllers.%s.Edit)", indent, base, ctrlVar))
		case "update":
			newLines = append(newLines, fmt.Sprintf("%smux.HandleFunc(\"PUT %s/{id}\", controllers.%s.Update)", indent, base, ctrlVar))
			newLines = append(newLines, fmt.Sprintf("%smux.HandleFunc(\"PATCH %s/{id}\", controllers.%s.Update)", indent, base, ctrlVar))
		case "destroy":
			newLines = append(newLines, fmt.Sprintf("%smux.HandleFunc(\"DELETE %s/{id}\", controllers.%s.Destroy)", indent, base, ctrlVar))
		}
	}
	newLines = append(newLines, "") // add a blank line for readability

	// check if any of the routes already exist
	for _, nl := range newLines {
		exist := false
		for _, l := range lines {
			if strings.TrimSpace(l) == strings.TrimSpace(nl) {
				exist = true
				break
			}
		}
		if !exist {
			lines = append(lines[:insertIdx], append([]string{nl}, lines[insertIdx:]...)...)
			insertIdx++
		}
	}

	out := strings.Join(lines, "\n")
	formatted, err := format.Source([]byte(out))
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("update", path)
	return nil
}

// createTemplateFiles generates HTML templates for GET actions.
func createTemplateFiles(name string, actions []string) error {
	snake := toSnakeCase(name)
	titleName := toCamelCase(name)
	for _, act := range actions {
		switch act {
		case "index", "show", "new", "edit":
			file := filepath.Join("templates", fmt.Sprintf("%s_%s.html.tmpl", snake, act))
			if _, err := os.Stat(file); err == nil {
				fmt.Println("exists", file)
				continue
			}
			var buf bytes.Buffer
			buf.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n    <meta charset=\"UTF-8\">\n")
			buf.WriteString(fmt.Sprintf("    <title>%s %s</title>\n</head>\n<body>\n", titleName, strings.Title(act)))
			buf.WriteString(fmt.Sprintf("    <h1>%s %s</h1>\n</body>\n</html>\n", titleName, strings.Title(act)))
			if err := os.WriteFile(file, buf.Bytes(), 0644); err != nil {
				return err
			}
			fmt.Println("create", file)
		}
	}
	return nil
}

// createModelTestFile creates a placeholder _test.go file for the model.
func createModelTestFile(modelName string) error {
	file := filepath.Join("models", toSnakeCase(modelName)+"_test.go")
	if _, err := os.Stat(file); err == nil {
		fmt.Println("exists", file)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("package models\n\n")
	buf.WriteString("import \"testing\"\n\n")
	buf.WriteString(fmt.Sprintf("func Test%sPlaceholder(t *testing.T) {}\n", toCamelCase(modelName)))
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	if err := os.WriteFile(file, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("create", file)
	return nil
}

// createControllerTestFile creates a placeholder _test.go for the controller.
func createControllerTestFile(name string) error {
	file := filepath.Join("controllers", toSnakeCase(name)+"_controller_test.go")
	if _, err := os.Stat(file); err == nil {
		fmt.Println("exists", file)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("package controllers\n\n")
	buf.WriteString("import \"testing\"\n\n")
	buf.WriteString(fmt.Sprintf("func Test%sControllerPlaceholder(t *testing.T) {}\n", toCamelCase(name)))
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	if err := os.WriteFile(file, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("create", file)
	return nil
}

// createUserModelAuth writes a basic User model used for authentication if it
// doesn't already exist and ensures it is migrated in db/db.go.
func createUserModelAuth() error {
	path := filepath.Join("models", "user.go")
	exists := false
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		exists = true
	} else if !os.IsNotExist(err) {
		return err
	}
	if !exists {
		var buf bytes.Buffer
		buf.WriteString("package models\n\n")
		buf.WriteString("import \"gorm.io/gorm\"\n\n")
		buf.WriteString("// User represents an application user\n")
		buf.WriteString("type User struct {\n")
		buf.WriteString("\tgorm.Model\n")
		buf.WriteString("\tEmail        string `gorm:\"unique;not null\"`\n")
		buf.WriteString("\tPasswordHash []byte\n")
		buf.WriteString("\tIsActive bool	`gorm:\"default:true\"`\n")
		buf.WriteString("\tIsAdmin bool `gorm:\"default:false\"`\n")
		buf.WriteString("}\n")
		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, formatted, 0644); err != nil {
			return err
		}
		fmt.Println("create", path)
	}
	if err := updateDBFile("User"); err != nil {
		return err
	}
	return nil
}

// createSessionFile sets up cookie session helpers for login state.
func createSessionFile() error {
	path := filepath.Join("session", "session.go")
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("package session\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"net/http\"\n\n")
	buf.WriteString("\t\"github.com/gorilla/sessions\"\n")
	buf.WriteString(")\n\n")
	buf.WriteString("const SESSION_NAME_KEY = \"session\"\n")
	buf.WriteString("const LOGGED_IN_KEY = \"logged_in\"\n")
	buf.WriteString("const EMAIL_KEY = \"email\"\n\n")
	buf.WriteString("var store = sessions.NewCookieStore([]byte(\"super-secret-key\"))\n\n")
	buf.WriteString("func SetLoggedIn(w http.ResponseWriter, r *http.Request, email string) {\n")
	buf.WriteString("\tsession, _ := store.Get(r, SESSION_NAME_KEY)\n")
	buf.WriteString("\tsession.Values[LOGGED_IN_KEY] = true\n")
	buf.WriteString("\tsession.Values[EMAIL_KEY] = email\n")
	buf.WriteString("\tsession.Save(r, w)\n")
	buf.WriteString("}\n\n")
	buf.WriteString("func Logout(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\tsession, _ := store.Get(r, SESSION_NAME_KEY)\n")
	buf.WriteString("\tdelete(session.Values, LOGGED_IN_KEY)\n")
	buf.WriteString("\tdelete(session.Values, EMAIL_KEY)\n")
	buf.WriteString("\tsession.Save(r, w)\n")
	buf.WriteString("}\n\n")
	buf.WriteString("func IsLoggedIn(r *http.Request) bool {\n")
	buf.WriteString("\tsession, _ := store.Get(r, SESSION_NAME_KEY)\n")
	buf.WriteString("\tloggedIn, ok := session.Values[LOGGED_IN_KEY].(bool)\n")
	buf.WriteString("\treturn ok && loggedIn\n")
	buf.WriteString("}\n")
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("create", path)
	return nil
}

// createSessionTestFile sets up tests for session helpers.
func createSessionTestFile() error {
	path := filepath.Join("session", "session_test.go")
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("package session\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"net/http/httptest\"\n")
	buf.WriteString("\t\"testing\"\n")
	buf.WriteString(")\n\n")
	buf.WriteString("func TestSessionLoginLogout(t *testing.T) {\n")
	buf.WriteString("\treq := httptest.NewRequest(\"GET\", \"/\", nil)\n")
	buf.WriteString("\tw := httptest.NewRecorder()\n")
	buf.WriteString("\tSetLoggedIn(w, req, \"test@example.com\")\n")
	buf.WriteString("\tcookie := w.Result().Cookies()[0]\n\n")
	buf.WriteString("\treq2 := httptest.NewRequest(\"GET\", \"/\", nil)\n")
	buf.WriteString("\treq2.AddCookie(cookie)\n")
	buf.WriteString("\tif !IsLoggedIn(req2) {\n")
	buf.WriteString("\t\tt.Fatal(\"expected logged in\")\n")
	buf.WriteString("\t}\n\n")
	buf.WriteString("\tw2 := httptest.NewRecorder()\n")
	buf.WriteString("\tLogout(w2, req2)\n")
	buf.WriteString("\tcookie2 := w2.Result().Cookies()[0]\n")
	buf.WriteString("\treq3 := httptest.NewRequest(\"GET\", \"/\", nil)\n")
	buf.WriteString("\treq3.AddCookie(cookie2)\n")
	buf.WriteString("\tif IsLoggedIn(req3) {\n")
	buf.WriteString("\t\tt.Fatal(\"expected logged out\")\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n")
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("create", path)
	return nil
}

// createAuthMiddlewareFile creates RequireLogin middleware.
func createAuthMiddlewareFile() error {
	path := filepath.Join("middleware", "auth.go")
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("package middleware\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"monolith/session\"\n")
	buf.WriteString("\t\"net/http\"\n")
	buf.WriteString(")\n\n")
	buf.WriteString("// RequireLogin ensures the user is logged in before accessing a route\n")
	buf.WriteString("func RequireLogin(next http.HandlerFunc) http.HandlerFunc {\n")
	buf.WriteString("\treturn func(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\t\tif !session.IsLoggedIn(r) {\n")
	buf.WriteString("\t\t\thttp.Redirect(w, r, \"/login\", http.StatusSeeOther)\n")
	buf.WriteString("\t\t\treturn\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t\tnext(w, r)\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n")
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("create", path)
	return nil
}

// createAuthMiddlewareTestFile generates tests for the RequireLogin middleware.
func createAuthMiddlewareTestFile() error {
	path := filepath.Join("middleware", "auth_test.go")
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("package middleware\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"net/http\"\n")
	buf.WriteString("\t\"net/http/httptest\"\n")
	buf.WriteString("\t\"testing\"\n\n")
	buf.WriteString("\t\"monolith/session\"\n")
	buf.WriteString(")\n\n")
	buf.WriteString("func TestRequireLogin(t *testing.T) {\n")
	buf.WriteString("\thandlerCalled := false\n")
	buf.WriteString("\thandler := RequireLogin(func(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\t\thandlerCalled = true\n")
	buf.WriteString("\t})\n")
	buf.WriteString("\treq := httptest.NewRequest(\"GET\", \"/\", nil)\n")
	buf.WriteString("\tw := httptest.NewRecorder()\n")
	buf.WriteString("\thandler(w, req)\n")
	buf.WriteString("\tif w.Result().StatusCode != http.StatusSeeOther {\n")
	buf.WriteString("\t\tt.Errorf(\"expected redirect status, got %d\", w.Result().StatusCode)\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\tif handlerCalled {\n")
	buf.WriteString("\t\tt.Errorf(\"handler should not be called when not logged in\")\n")
	buf.WriteString("\t}\n\n")
	buf.WriteString("\treq2 := httptest.NewRequest(\"GET\", \"/\", nil)\n")
	buf.WriteString("\tw2 := httptest.NewRecorder()\n")
	buf.WriteString("\tsession.SetLoggedIn(w2, req2, \"test@example.com\")\n")
	buf.WriteString("\tcookie := w2.Result().Cookies()[0]\n")
	buf.WriteString("\treq2.AddCookie(cookie)\n")
	buf.WriteString("\tw3 := httptest.NewRecorder()\n")
	buf.WriteString("\thandler(w3, req2)\n")
	buf.WriteString("\tif w3.Result().StatusCode != http.StatusOK {\n")
	buf.WriteString("\t\tt.Errorf(\"expected 200 status when logged in, got %d\", w3.Result().StatusCode)\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\tif !handlerCalled {\n")
	buf.WriteString("\t\tt.Errorf(\"handler should be called when logged in\")\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n")
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("create", path)
	return nil
}

// createAuthControllerFile creates controller handling signup and login.
func createAuthControllerFile() error {
	path := filepath.Join("controllers", "auth_controller.go")
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("package controllers\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"net/http\"\n\n")
	buf.WriteString("\t\"monolith/db\"\n")
	buf.WriteString("\t\"monolith/models\"\n")
	buf.WriteString("\t\"monolith/session\"\n")
	buf.WriteString("\t\"monolith/templates\"\n")
	buf.WriteString(")\n\n")
	buf.WriteString("type AuthController struct{}\n\n")
	buf.WriteString("var AuthCtrl = &AuthController{}\n\n")
	buf.WriteString("func (ac *AuthController) ShowLoginForm(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\ttemplates.ExecuteTemplate(w, \"login.html.tmpl\", nil)\n")
	buf.WriteString("}\n\n")
	buf.WriteString("func (ac *AuthController) ShowSignupForm(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\ttemplates.ExecuteTemplate(w, \"signup.html.tmpl\", nil)\n")
	buf.WriteString("}\n\n")
	buf.WriteString("func (ac *AuthController) Signup(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\tif err := r.ParseForm(); err != nil {\n")
	buf.WriteString("\t\thttp.Error(w, \"invalid form\", http.StatusBadRequest)\n")
	buf.WriteString("\t\treturn\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\temail := r.FormValue(\"email\")\n")
	buf.WriteString("\tpassword := r.FormValue(\"password\")\n")
	buf.WriteString("\tif email == \"\" || password == \"\" {\n")
	buf.WriteString("\t\thttp.Error(w, \"missing credentials\", http.StatusBadRequest)\n")
	buf.WriteString("\t\treturn\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\tif _, err := models.CreateUser(db.GetDB(), email, password); err != nil {\n")
	buf.WriteString("\t\thttp.Error(w, \"could not create user\", http.StatusInternalServerError)\n")
	buf.WriteString("\t\treturn\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\tsession.SetLoggedIn(w, r, email)\n")
	buf.WriteString("\thttp.Redirect(w, r, \"/dashboard\", http.StatusSeeOther)\n")
	buf.WriteString("}\n\n")
	buf.WriteString("func (ac *AuthController) Login(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\tif err := r.ParseForm(); err != nil {\n")
	buf.WriteString("\t\thttp.Error(w, \"invalid form\", http.StatusBadRequest)\n")
	buf.WriteString("\t\treturn\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\temail := r.FormValue(\"email\")\n")
	buf.WriteString("\tpassword := r.FormValue(\"password\")\n")
	buf.WriteString("\tif _, err := models.AuthenticateUser(db.GetDB(), email, password); err != nil {\n")
	buf.WriteString("\t\thttp.Error(w, \"invalid credentials\", http.StatusUnauthorized)\n")
	buf.WriteString("\t\treturn\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\tsession.SetLoggedIn(w, r, email)\n")
	buf.WriteString("\thttp.Redirect(w, r, \"/dashboard\", http.StatusSeeOther)\n")
	buf.WriteString("}\n\n")
	buf.WriteString("func (ac *AuthController) Logout(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\tsession.Logout(w, r)\n")
	buf.WriteString("\thttp.Redirect(w, r, \"/\", http.StatusSeeOther)\n")
	buf.WriteString("}\n")
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("create", path)
	return nil
}

// createLoginTemplate generates a basic login template.
func createLoginTemplate() error {
	path := filepath.Join("templates", "login.html.tmpl")
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n    <meta charset=\"UTF-8\">\n    <title>Login</title>\n</head>\n<body>\n    <h1>Login</h1>\n    <form method=\"POST\" action=\"/login\">\n        <label>Email: <input type=\"email\" name=\"email\"></label><br>\n        <label>Password: <input type=\"password\" name=\"password\"></label><br>\n        <button type=\"submit\">Login</button>\n    </form>\n    <a href=\"/signup\">Sign up</a>\n</body>\n</html>\n")
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return err
	}
	fmt.Println("create", path)
	return nil
}

func createSignupTemplate() error {
	path := filepath.Join("templates", "signup.html.tmpl")
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n    <meta charset=\"UTF-8\">\n    <title>Sign Up</title>\n</head>\n<body>\n    <h1>Sign Up</h1>\n    <form method=\"POST\" action=\"/signup\">\n        <label>Email: <input type=\"email\" name=\"email\"></label><br>\n        <label>Password: <input type=\"password\" name=\"password\"></label><br>\n        <button type=\"submit\">Create Account</button>\n    </form>\n    <a href=\"/login\">Login</a>\n</body>\n</html>\n")
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return err
	}
	fmt.Println("create", path)
	return nil
}

// updateRoutesForAuth injects the authentication routes if missing.
func updateRoutesForAuth() error {
	path := filepath.Join("routes", "routes.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	insertIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "staticFileServer") {
			insertIdx = i + 1
			break
		}
	}
	if insertIdx == -1 {
		return fmt.Errorf("could not find insertion point in routes.go")
	}
	indent := leadingWhitespace(lines[insertIdx-1])
	newLines := []string{
		fmt.Sprintf("%smux.HandleFunc(\"GET /login\", controllers.AuthCtrl.ShowLoginForm)", indent),
		fmt.Sprintf("%smux.HandleFunc(\"POST /login\", controllers.AuthCtrl.Login)", indent),
		fmt.Sprintf("%smux.HandleFunc(\"GET /signup\", controllers.AuthCtrl.ShowSignupForm)", indent),
		fmt.Sprintf("%smux.HandleFunc(\"POST /signup\", controllers.AuthCtrl.Signup)", indent),
		fmt.Sprintf("%smux.HandleFunc(\"GET /logout\", controllers.AuthCtrl.Logout)", indent),
		"",
	}

	for _, nl := range newLines {
		found := false
		for _, l := range lines {
			if strings.TrimSpace(l) == strings.TrimSpace(nl) {
				found = true
				break
			}
		}
		if !found {
			lines = append(lines[:insertIdx], append([]string{nl}, lines[insertIdx:]...)...)
			insertIdx++
		}
	}

	out := strings.Join(lines, "\n")
	formatted, err := format.Source([]byte(out))
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("update", path)
	return nil
}

func createJobFunction(name string) error {
	path := filepath.Join("jobs", "jobs.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if strings.Contains(string(data), "func "+name+"(") {
		fmt.Println("exists", path)
		return nil
	}
	lines := strings.Split(string(data), "\n")
	insertIdx := len(lines)
	for i, line := range lines {
		if strings.Contains(line, "example usage") {
			insertIdx = i - 1
			break
		}
	}
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("func %s(payload string) error {\n", name))
	buf.WriteString("\t// TODO: implement job\n")
	buf.WriteString("\treturn nil\n")
	buf.WriteString("}\n\n")
	newLines := append(lines[:insertIdx], append(strings.Split(buf.String(), "\n"), lines[insertIdx:]...)...)
	out := strings.Join(newLines, "\n")
	formatted, err := format.Source([]byte(out))
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("update", path)
	return nil
}

func updateJobTypeEnum(name string) error {
	path := filepath.Join("models", "jobs.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if strings.Contains(string(data), "JobType"+name) {
		fmt.Println("exists", path)
		return nil
	}
	lines := strings.Split(string(data), "\n")
	start, end := -1, -1
	for i, line := range lines {
		if strings.Contains(line, "JobTypePrint") {
			for j := i; j >= 0; j-- {
				if strings.Contains(lines[j], "const (") {
					start = j
					break
				}
			}
			if start != -1 {
				for k := start + 1; k < len(lines); k++ {
					if strings.TrimSpace(lines[k]) == ")" {
						end = k
						break
					}
				}
			}
			break
		}
	}
	if start == -1 || end == -1 {
		return fmt.Errorf("could not find JobType enum in %s", path)
	}
	indent := leadingWhitespace(lines[end-1])
	newLine := indent + "JobType" + name
	lines = append(lines[:end], append([]string{newLine}, lines[end:]...)...)
	out := strings.Join(lines, "\n")
	formatted, err := format.Source([]byte(out))
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("update", path)
	return nil
}

func registerJobInQueue(name string) error {
	path := filepath.Join("jobs", "job_queue.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if strings.Contains(string(data), "JobType"+name) {
		fmt.Println("exists", path)
		return nil
	}
	lines := strings.Split(string(data), "\n")
	insertIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "jobQueue.register(") {
			insertIdx = i + 1
		}
	}
	if insertIdx == -1 {
		return fmt.Errorf("could not find registration point in %s", path)
	}
	indent := leadingWhitespace(lines[insertIdx-1])
	newLine := fmt.Sprintf("%sjobQueue.register(models.JobType%s, %s)", indent, name, name)
	lines = append(lines[:insertIdx], append([]string{newLine}, lines[insertIdx:]...)...)
	out := strings.Join(lines, "\n")
	formatted, err := format.Source([]byte(out))
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("update", path)
	return nil
}

func leadingWhitespace(s string) string {
	i := 0
	for ; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' {
			break
		}
	}
	return s[:i]
}

func toCamelCase(s string) string {
	if s == "" {
		return s
	}
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		r := []rune(p)
		r[0] = unicode.ToUpper(r[0])
		parts[i] = string(r)
	}
	return strings.Join(parts, "")
}

func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
