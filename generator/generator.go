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
   or: make generator <command> [arguments]

Available commands:
  model NAME [field:type...]    Create a model struct and update db migrations
  controller NAME [actions]     Create a controller, views and routes
  resource NAME [field:type...] Create model and full REST controller
  authentication                Scaffold basic user authentication
  job NAME                      Create a skeleton background job
  admin                         Add an admin dashboard with profiling helpers

Run "go run main.go generator help COMMAND" for detailed usage of a command.

Examples:
  make generator model Widget name:string price:int
  make generator controller widgets index show
  make generator resource widget name:string price:int
  make generator authentication
  make generator job MyJob
`

const modelHelp = `Usage: go run main.go generator model NAME [field:type...]

Creates models/NAME.go with a struct embedding gorm.Model and updates db/db.go
so the model is migrated automatically. Optional field arguments add struct
fields with the specified Go types.

Example:
  make generator model Widget name:string price:int
`

const controllerHelp = `Usage: go run main.go generator controller NAME [actions]

Generates controllers/NAME_controller.go and matching templates. If actions are
provided, RESTful routes are inserted into routes/routes.go. Use "all" to
generate the full set of CRUD actions.

Example:
  make generator controller widgets index show
`

const resourceHelp = `Usage: go run main.go generator resource NAME [field:type...]

Creates a model and a pluralised controller with all CRUD actions, templates and
routes. Pass the singular model name; the controller will be pluralised.

Example:
  make generator resource widget name:string price:int
`

const authenticationHelp = `Usage: go run main.go generator authentication

Scaffolds a basic user model along with session helpers, login and signup
templates and routes. This generator takes no arguments.

Example:
  make generator authentication
`

const jobHelp = `Usage: go run main.go generator job NAME

Creates jobs/NAME_job.go with a stub function, registers it in the job queue and
adds JobTypeNAME to models/jobs.go.

Example:
  make generator job MyJob
`

const adminHelp = `Usage: go run main.go generator admin

Generates an /admin dashboard wrapped in admin-only middleware. If a User model
does not exist, the authentication generator will be invoked automatically.

Example:
  make generator admin
`

// Run dispatches to the specific generator based on args.
// args should not include the leading "generator" argument.
func Run(args []string) error {
	if len(args) == 0 {
		fmt.Print(helpMessage)
		return errors.New("missing generator type")
	}

	// `help <cmd>` displays detailed help
	if args[0] == "help" {
		cmd := ""
		if len(args) > 1 {
			cmd = args[1]
		}
		printHelp(cmd)
		return nil
	}

	switch args[0] {
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
	case "admin":
		return runAdmin(args[1:])
	default:
		fmt.Print(helpMessage)
		return fmt.Errorf("unknown generator: %s", args[0])
	}
}

func printHelp(cmd string) {
	switch cmd {
	case "model":
		fmt.Print(modelHelp)
	case "controller":
		fmt.Print(controllerHelp)
	case "resource":
		fmt.Print(resourceHelp)
	case "authentication":
		fmt.Print(authenticationHelp)
	case "job":
		fmt.Print(jobHelp)
	case "admin":
		fmt.Print(adminHelp)
	default:
		fmt.Print(helpMessage)
	}
}

// runModel creates a new model struct and updates db migrations.
func runModel(args []string) error {
	if len(args) == 0 {
		fmt.Print(modelHelp)
		return errors.New("model name required")
	}
	modelName := toCamelCase(args[0])
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

// runController creates a new controller with optional REST actions and updates routes and views.
func runController(args []string) error {
	if len(args) == 0 {
		fmt.Print(controllerHelp)
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
// associated views, routes and placeholder tests.
func runResource(args []string) error {
	if len(args) == 0 {
		fmt.Print(resourceHelp)
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
		fmt.Print(jobHelp)
		return errors.New("job name required")
	}
	baseName := toCamelCase(args[0])
	if err := createJobFile(baseName); err != nil {
		return err
	}
	if err := createJobTestFile(baseName); err != nil {
		return err
	}
	if err := updateJobTypeEnum(baseName); err != nil {
		return err
	}
	if err := registerJobInQueue(baseName); err != nil {
		return err
	}
	return nil
}

// runAuthentication scaffolds user authentication helpers, controller,
// views and routes.
func runAuthentication(args []string) error {
	if len(args) != 0 {
		fmt.Print(authenticationHelp)
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

// runAdmin adds an admin dashboard with profiling helpers.
func runAdmin(args []string) error {
	if len(args) != 0 {
		fmt.Print(adminHelp)
		return errors.New("admin generator takes no arguments")
	}
	if _, err := os.Stat(filepath.Join("models", "user.go")); os.IsNotExist(err) {
		if err := runAuthentication([]string{}); err != nil {
			return err
		}
	}
	if err := createSessionEmailHelper(); err != nil {
		return err
	}
	if err := createAdminControllerFile(); err != nil {
		return err
	}
	if err := createAdminTemplate(); err != nil {
		return err
	}
	if err := createAdminMiddlewareFile(); err != nil {
		return err
	}
	if err := createAdminMiddlewareTestFile(); err != nil {
		return err
	}
	if err := updateRoutesForAdmin(); err != nil {
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
	camelName := toCamelCase(modelName)
	buf.WriteString(fmt.Sprintf("type %s struct {\n", camelName))
	buf.WriteString("\tgorm.Model\n")
	buf.WriteString("\tIsActive bool `gorm:\"default:true\"`\n")

	for _, f := range fields {
		parts := strings.SplitN(f, ":", 2)
		name := parts[0]
		typ := "string"
		if len(parts) > 1 {
			typ = parts[1]
		}
		buf.WriteString(fmt.Sprintf("\t%s %s\n", toCamelCase(name), typ))
	}
	buf.WriteString("}\n\n")

	// add blank hook implementations so users can customize validations
	buf.WriteString(fmt.Sprintf("// BeforeSave is called by GORM before persisting a %s.\n", camelName))
	buf.WriteString("// Use this hook to validate or modify the model before saving.\n")
	buf.WriteString(fmt.Sprintf("func (m *%s) BeforeSave(tx *gorm.DB) error {\n", camelName))
	buf.WriteString("\treturn nil\n")
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf("// AfterSave is triggered by GORM after a %s has been saved.\n", camelName))
	buf.WriteString("// This can be used for post-save actions or additional validation.\n")
	buf.WriteString(fmt.Sprintf("func (m *%s) AfterSave(tx *gorm.DB) error {\n", camelName))
	buf.WriteString("\treturn nil\n")
	buf.WriteString("}\n\n")

	pluralName := toCamelCase(inflection.Plural(modelName))
	buf.WriteString(fmt.Sprintf("func Create%s(db *gorm.DB, m *%s) error {\n", camelName, camelName))
	buf.WriteString("\treturn db.Create(m).Error\n")
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf("func Get%sByID(db *gorm.DB, id uint) (*%s, error) {\n", camelName, camelName))
	buf.WriteString(fmt.Sprintf("\tvar m %s\n", camelName))
	buf.WriteString("\tif err := db.First(&m, id).Error; err != nil {\n")
	buf.WriteString("\t\treturn nil, err\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\treturn &m, nil\n")
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf("func GetAll%s(db *gorm.DB) ([]%s, error) {\n", pluralName, camelName))
	buf.WriteString(fmt.Sprintf("\tvar ms []%s\n", camelName))
	buf.WriteString("\tif err := db.Find(&ms).Error; err != nil {\n")
	buf.WriteString("\t\treturn nil, err\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\treturn ms, nil\n")
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf("func Update%s(db *gorm.DB, m *%s) error {\n", camelName, camelName))
	buf.WriteString("\treturn db.Save(m).Error\n")
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf("func Delete%s(db *gorm.DB, id uint) error {\n", camelName))
	buf.WriteString(fmt.Sprintf("\treturn db.Model(&%s{}).Where(\"id = ?\", id).Update(\"is_active\", false).Error\n", camelName))
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
	modelPath := filepath.Join("models", toSnakeCase(modelName)+".go")
	pluralModelName := toCamelCase(inflection.Plural(toSnakeCase(modelName)))
	hasModel := false
	if _, err := os.Stat(modelPath); err == nil {
		hasModel = true
	}
	needDB := false
	needTemplates := false
	needStrconv := false
	for _, a := range actions {
		switch a {
		case "index", "show", "create", "edit", "update", "destroy":
			needDB = true
		}
		switch a {
		case "show", "edit", "update", "destroy":
			needStrconv = true
		}
		switch a {
		case "index", "show", "new", "edit":
			needTemplates = true
		}
	}
	if needDB && !hasModel {
		// without a corresponding model file we cannot use the db package
		needDB = false
	}

	var buf bytes.Buffer
	buf.WriteString("package controllers\n\n")

	imports := []string{"\"net/http\""}
	if needStrconv {
		imports = append(imports, "\"strconv\"")
	}
	if needTemplates {
		imports = append(imports, "\"monolith/views\"")
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
			if hasModel {
				buf.WriteString(fmt.Sprintf("\trecords, _ := models.GetAll%s(db.GetDB())\n", pluralModelName))
				buf.WriteString(fmt.Sprintf("\tviews.Render(w, \"%s_index.html.tmpl\", records)\n", toSnakeCase(name)))
			} else {
				buf.WriteString(fmt.Sprintf("\tviews.Render(w, \"%s_index.html.tmpl\", nil)\n", toSnakeCase(name)))
			}
			buf.WriteString("}\n\n")
		case "show":
			buf.WriteString(fmt.Sprintf("func (c *%s) Show(w http.ResponseWriter, r *http.Request) {\n", ctrlName))
			if hasModel {
				buf.WriteString("\tidStr := r.PathValue(\"id\")\n")
				buf.WriteString("\tid, _ := strconv.Atoi(idStr)\n")
				buf.WriteString(fmt.Sprintf("\trecord, _ := models.Get%sByID(db.GetDB(), uint(id))\n", modelName))
				buf.WriteString(fmt.Sprintf("\tviews.Render(w, \"%s_show.html.tmpl\", record)\n", toSnakeCase(name)))
			} else {
				buf.WriteString(fmt.Sprintf("\tviews.Render(w, \"%s_show.html.tmpl\", nil)\n", toSnakeCase(name)))
			}
			buf.WriteString("}\n\n")
		case "new":
			buf.WriteString(fmt.Sprintf("func (c *%s) New(w http.ResponseWriter, r *http.Request) {\n", ctrlName))
			buf.WriteString(fmt.Sprintf("\tviews.Render(w, \"%s_new.html.tmpl\", nil)\n", toSnakeCase(name)))
			buf.WriteString("}\n\n")
		case "create":
			buf.WriteString(fmt.Sprintf("func (c *%s) Create(w http.ResponseWriter, r *http.Request) {\n", ctrlName))
			if hasModel {
				buf.WriteString(fmt.Sprintf("\tvar record models.%s\n", modelName))
				buf.WriteString("\t// TODO: parse form values into &record\n")
				buf.WriteString(fmt.Sprintf("\t_ = models.Create%s(db.GetDB(), &record)\n", modelName))
			}
			buf.WriteString(fmt.Sprintf("\thttp.Redirect(w, r, \"/%s\", http.StatusSeeOther)\n", toSnakeCase(name)))
			buf.WriteString("}\n\n")
		case "edit":
			buf.WriteString(fmt.Sprintf("func (c *%s) Edit(w http.ResponseWriter, r *http.Request) {\n", ctrlName))
			if hasModel {
				buf.WriteString("\tidStr := r.PathValue(\"id\")\n")
				buf.WriteString("\tid, _ := strconv.Atoi(idStr)\n")
				buf.WriteString(fmt.Sprintf("\trecord, _ := models.Get%sByID(db.GetDB(), uint(id))\n", modelName))
				buf.WriteString(fmt.Sprintf("\tviews.Render(w, \"%s_edit.html.tmpl\", record)\n", toSnakeCase(name)))
			} else {
				buf.WriteString(fmt.Sprintf("\tviews.Render(w, \"%s_edit.html.tmpl\", nil)\n", toSnakeCase(name)))
			}
			buf.WriteString("}\n\n")
		case "update":
			buf.WriteString(fmt.Sprintf("func (c *%s) Update(w http.ResponseWriter, r *http.Request) {\n", ctrlName))
			if hasModel {
				buf.WriteString("\tidStr := r.PathValue(\"id\")\n")
				buf.WriteString("\tid, _ := strconv.Atoi(idStr)\n")
				buf.WriteString(fmt.Sprintf("\trecord, _ := models.Get%sByID(db.GetDB(), uint(id))\n", modelName))
				buf.WriteString("\t// TODO: update record fields\n")
				buf.WriteString(fmt.Sprintf("\t_ = models.Update%s(db.GetDB(), record)\n", modelName))
				buf.WriteString(fmt.Sprintf("\thttp.Redirect(w, r, \"/%s/\"+idStr, http.StatusSeeOther)\n", toSnakeCase(name)))
			} else {
				buf.WriteString(fmt.Sprintf("\thttp.Redirect(w, r, \"/%s\", http.StatusSeeOther)\n", toSnakeCase(name)))
			}
			buf.WriteString("}\n\n")
		case "destroy":
			buf.WriteString(fmt.Sprintf("func (c *%s) Destroy(w http.ResponseWriter, r *http.Request) {\n", ctrlName))
			if hasModel {
				buf.WriteString("\tidStr := r.PathValue(\"id\")\n")
				buf.WriteString("\tid, _ := strconv.Atoi(idStr)\n")
				buf.WriteString(fmt.Sprintf("\t_ = models.Delete%s(db.GetDB(), uint(id))\n", modelName))
			}
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
	lines = ensureControllersImport(lines)
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

// createTemplateFiles generates HTML views for GET actions.
func createTemplateFiles(name string, actions []string) error {
	snake := toSnakeCase(name)
	for _, act := range actions {
		switch act {
		case "index", "show", "new", "edit":
			file := filepath.Join("views", fmt.Sprintf("%s_%s.html.tmpl", snake, act))
			if _, err := os.Stat(file); err == nil {
				fmt.Println("exists", file)
				continue
			}
			var buf bytes.Buffer
			buf.WriteString(`{{template "base.html.tmpl" .}}

{{define "title"}}<title></title>{{end}}

{{block "meta" .}}
{{end}}

 {{block "header" .}}
 {{end}}

{{block "scripts" .}}
{{end}}

{{define "stylesheet"}}
<style>
</style>
{{end}}

{{define "body"}}
{{end}}


{{define "footer"}}
<footer>
</footer>
{{end}}
`)
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
		buf.WriteString(`package models

import (
    "errors"

    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"
)

// User represents a user in the database
type User struct {
    gorm.Model          // Adds ID, CreatedAt, UpdatedAt, DeletedAt fields
    Email        string ` + "`gorm:\"unique;not null\"`" + `
    PasswordHash []byte
    IsActive     bool ` + "`gorm:\"default:true\"`" + `
    IsAdmin      bool ` + "`gorm:\"default:false\"`" + `
}

// BeforeSave is called by GORM before persisting a User.
// Use this hook to validate or modify the model before saving.
func (u *User) BeforeSave(tx *gorm.DB) error {
    return nil
}

// AfterSave is triggered by GORM after a User has been saved.
// This can be used for post-save actions or additional validation.
func (u *User) AfterSave(tx *gorm.DB) error {
    return nil
}

// GetUser fetches a user by email from the database
func GetUser(db *gorm.DB, email string) (*User, error) {
    var user User
    result := db.Where(&User{Email: email}).Take(&user)
    if result.Error != nil {
        return nil, result.Error
    }
    return &user, nil
}

// CreateUser inserts a new user into the database
func CreateUser(db *gorm.DB, email, password string) (*User, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return nil, err
    }
    user := User{
        Email:        email,
        PasswordHash: hash,
    }
    if err := db.Create(&user).Error; err != nil {
        return nil, err
    }
    return &user, nil
}

// AuthenticateUser verifies the provided credentials and returns the user if valid.
func AuthenticateUser(db *gorm.DB, email, password string) (*User, error) {
    user, err := GetUser(db, email)
    if err != nil {
        return nil, err
    }
    if bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)) != nil {
        return nil, errors.New("invalid credentials")
    }
    return user, nil
}
`)
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
	buf.WriteString("// GetSession retrieves the session from the request\n")
	buf.WriteString("func GetSession(r *http.Request) (*sessions.Session, error) {\n")
	buf.WriteString("\treturn store.Get(r, SESSION_NAME_KEY)\n")
	buf.WriteString("}\n\n")
	buf.WriteString("func SetLoggedIn(w http.ResponseWriter, r *http.Request, email string) {\n")
	buf.WriteString("\tsession, _ := GetSession(r)\n")
	buf.WriteString("\tsession.Options = &sessions.Options{MaxAge: 7 * 24 * 60 * 60, SameSite: http.SameSiteLaxMode}\n")
	buf.WriteString("\tsession.Values[LOGGED_IN_KEY] = true\n")
	buf.WriteString("\tsession.Values[EMAIL_KEY] = email\n")
	buf.WriteString("\tsession.Save(r, w)\n")
	buf.WriteString("}\n\n")
	buf.WriteString("func Logout(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\tsession, _ := GetSession(r)\n")
	buf.WriteString("\tdelete(session.Values, LOGGED_IN_KEY)\n")
	buf.WriteString("\tdelete(session.Values, EMAIL_KEY)\n")
	buf.WriteString("\tsession.Save(r, w)\n")
	buf.WriteString("}\n\n")
	buf.WriteString("func IsLoggedIn(r *http.Request) bool {\n")
	buf.WriteString("\tsession, _ := GetSession(r)\n")
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
	buf.WriteString("\t\"monolith/views\"\n")
	buf.WriteString(")\n\n")
	buf.WriteString("type AuthController struct{}\n\n")
	buf.WriteString("var AuthCtrl = &AuthController{}\n\n")
	buf.WriteString("func (ac *AuthController) ShowLoginForm(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\tviews.Render(w, \"login.html.tmpl\", nil)\n")
	buf.WriteString("}\n\n")
	buf.WriteString("func (ac *AuthController) ShowSignupForm(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\tviews.Render(w, \"signup.html.tmpl\", nil)\n")
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
	buf.WriteString("\thttp.Redirect(w, r, \"/\", http.StatusSeeOther)\n")
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
	buf.WriteString("\thttp.Redirect(w, r, \"/\", http.StatusSeeOther)\n")
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
	path := filepath.Join("views", "login.html.tmpl")
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString(`{{template "base.html.tmpl" .}}

{{define "title"}}<title>Login</title>{{end}}

{{block "meta" .}}
{{end}}

 {{block "header" .}}
 {{end}}

{{block "scripts" .}}
{{end}}

{{define "stylesheet"}}
<style>
</style>
{{end}}

{{define "body"}}
    <h1>Login</h1>
    <form method="POST" action="/login">
        <label>Email: <input type="email" name="email"></label><br>
        <label>Password: <input type="password" name="password"></label><br>
        <button type="submit">Login</button>
    </form>
    <a href="/signup">Sign up</a>
{{end}}


{{define "footer"}}
<footer>
</footer>
{{end}}
`)
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return err
	}
	fmt.Println("create", path)
	return nil
}

func createSignupTemplate() error {
	path := filepath.Join("views", "signup.html.tmpl")
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString(`{{template "base.html.tmpl" .}}

{{define "title"}}<title>Sign Up</title>{{end}}

{{block "meta" .}}
{{end}}

 {{block "header" .}}
 {{end}}

{{block "scripts" .}}
{{end}}

{{define "stylesheet"}}
<style>
</style>
{{end}}

{{define "body"}}
    <h1>Sign Up</h1>
    <form method="POST" action="/signup">
        <label>Email: <input type="email" name="email"></label><br>
        <label>Password: <input type="password" name="password"></label><br>
        <button type="submit">Create Account</button>
    </form>
    <a href="/login">Login</a>
{{end}}


{{define "footer"}}
<footer>
</footer>
{{end}}
`)
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
	lines = ensureControllersImport(lines)
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

func createJobFile(name string) error {
	file := filepath.Join("jobs", toSnakeCase(name)+"_job.go")
	if _, err := os.Stat(file); err == nil {
		fmt.Println("exists", file)
		return nil
	}
	funcName := name + "Job"
	var buf bytes.Buffer
	buf.WriteString("package jobs\n\n")
	buf.WriteString(fmt.Sprintf("func %s(payload string) error {\n", funcName))
	buf.WriteString("\t// TODO: implement job\n")
	buf.WriteString("\treturn nil\n")
	buf.WriteString("}\n")
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
	newLine := fmt.Sprintf("%sjobQueue.register(models.JobType%s, %sJob)", indent, name, name)
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

func createJobTestFile(name string) error {
	file := filepath.Join("jobs", toSnakeCase(name)+"_job_test.go")
	if _, err := os.Stat(file); err == nil {
		fmt.Println("exists", file)
		return nil
	}
	funcName := name + "Job"
	var buf bytes.Buffer
	buf.WriteString("package jobs\n\n")
	buf.WriteString("import \"testing\"\n\n")
	buf.WriteString(fmt.Sprintf("func Test%s(t *testing.T) {}\n", funcName))
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

// ensureControllersImport makes sure the routes file imports the controllers package.
func ensureControllersImport(lines []string) []string {
	for _, l := range lines {
		if strings.Contains(l, "\"monolith/controllers\"") {
			return lines
		}
	}

	start, end := -1, -1
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if t == "import (" {
			start = i
			continue
		}
		if start != -1 && t == ")" {
			end = i
			break
		}
	}

	if start != -1 && end != -1 {
		indent := leadingWhitespace(lines[start+1])
		newLine := indent + "\"monolith/controllers\""
		lines = append(lines[:end], append([]string{newLine}, lines[end:]...)...)
		return lines
	}

	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "import ") {
			pkg := strings.Trim(strings.TrimPrefix(t, "import"), " \"")
			block := []string{
				"import (",
				"    " + pkg,
				"    \"monolith/controllers\"",
				")",
			}
			lines = append(lines[:i], append(block, lines[i+1:]...)...)
			return lines
		}
	}

	// no import section found; append one after package line
	for i, l := range lines {
		if strings.HasPrefix(l, "package ") {
			block := []string{"import (", "    \"monolith/controllers\"", ")", ""}
			lines = append(lines[:i+1], append(block, lines[i+1:]...)...)
			break
		}
	}
	return lines
}

// ensureMiddlewareImport makes sure the routes file imports the middleware package.
func ensureMiddlewareImport(lines []string) []string {
	for _, l := range lines {
		if strings.Contains(l, "\"monolith/middleware\"") {
			return lines
		}
	}

	start, end := -1, -1
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if t == "import (" {
			start = i
			continue
		}
		if start != -1 && t == ")" {
			end = i
			break
		}
	}

	if start != -1 && end != -1 {
		indent := leadingWhitespace(lines[start+1])
		newLine := indent + "\"monolith/middleware\""
		lines = append(lines[:end], append([]string{newLine}, lines[end:]...)...)
		return lines
	}

	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "import ") {
			pkg := strings.Trim(strings.TrimPrefix(t, "import"), " \t")
			block := []string{
				"import (",
				"    " + pkg,
				"    \"monolith/middleware\"",
				")",
			}
			lines = append(lines[:i], append(block, lines[i+1:]...)...)
			return lines
		}
	}

	for i, l := range lines {
		if strings.HasPrefix(l, "package ") {
			block := []string{"import (", "    \"monolith/middleware\"", ")", ""}
			lines = append(lines[:i+1], append(block, lines[i+1:]...)...)
			break
		}
	}
	return lines
}

func createSessionEmailHelper() error {
	path := filepath.Join("session", "email.go")
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("package session\n\n")
	buf.WriteString("import \"net/http\"\n\n")
	buf.WriteString("func GetEmail(r *http.Request) string {\n")
	buf.WriteString("\ts, _ := GetSession(r)\n")
	buf.WriteString("\temail, _ := s.Values[EMAIL_KEY].(string)\n")
	buf.WriteString("\treturn email\n")
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

func createAdminControllerFile() error {
	path := filepath.Join("controllers", "admin_controller.go")
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("package controllers\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"net/http\"\n")
	buf.WriteString("\t\"monolith/views\"\n")
	buf.WriteString(")\n\n")
	buf.WriteString("type AdminController struct{}\n\n")
	buf.WriteString("var AdminCtrl = &AdminController{}\n\n")
	buf.WriteString("func (ac *AdminController) Dashboard(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\tif r.Method == http.MethodPost {\n")
	buf.WriteString("\t\tif err := r.ParseForm(); err == nil {\n")
	buf.WriteString("\t\t\ttyp := r.FormValue(\"profile_type\")\n")
	buf.WriteString("\t\t\tdur := r.FormValue(\"seconds\")\n")
	buf.WriteString("\t\t\tswitch typ {\n")
	buf.WriteString("\t\t\tcase \"cpu\":\n")
	buf.WriteString("\t\t\t\tif dur == \"\" { dur = \"30\" }\n")
	buf.WriteString("\t\t\t\thttp.Redirect(w, r, \"/debug/pprof/profile?seconds=\"+dur, http.StatusSeeOther)\n")
	buf.WriteString("\t\t\t\treturn\n")
	buf.WriteString("\t\t\tcase \"heap\":\n")
	buf.WriteString("\t\t\t\thttp.Redirect(w, r, \"/debug/pprof/heap\", http.StatusSeeOther)\n")
	buf.WriteString("\t\t\t\treturn\n")
	buf.WriteString("\t\t\tcase \"mem\":\n")
	buf.WriteString("\t\t\t\thttp.Redirect(w, r, \"/debug/pprof/allocs\", http.StatusSeeOther)\n")
	buf.WriteString("\t\t\t\treturn\n")
	buf.WriteString("\t\t\tcase \"trace\":\n")
	buf.WriteString("\t\t\t\tif dur == \"\" { dur = \"1\" }\n")
	buf.WriteString("\t\t\t\thttp.Redirect(w, r, \"/debug/pprof/trace?seconds=\"+dur, http.StatusSeeOther)\n")
	buf.WriteString("\t\t\t\treturn\n")
	buf.WriteString("\t\t\t}\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\tviews.Render(w, \"admin_dashboard.html.tmpl\", nil)\n")
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

func createAdminTemplate() error {
	path := filepath.Join("views", "admin_dashboard.html.tmpl")
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString(`{{template "base.html.tmpl" .}}

{{define "title"}}<title>Admin Dashboard</title>{{end}}

{{block "meta" .}}
{{end}}

 {{block "header" .}}
 {{end}}

{{block "scripts" .}}
{{end}}

{{define "stylesheet"}}
<style>
    @import url("https://cdn.simplecss.org/simple.css");
</style>
{{end}}

{{define "body"}}
    <h1>Admin Dashboard</h1>
    <div id="chart" style="height:300px;"></div>
    <h2>Download Profile</h2>
    <form method="POST">
        <label>Profile Type:
            <select name="profile_type">
                <option value="cpu">CPU</option>
                <option value="heap">Heap</option>
                <option value="mem">Memory</option>
                <option value="trace">Trace</option>
            </select>
        </label>
        <label>Duration (seconds):
            <input type="number" name="seconds" value="30">
        </label>
        <button type="submit">Download</button>
    </form>
{{end}}


{{define "footer"}}
<footer>
</footer>
{{end}}
`)
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return err
	}
	fmt.Println("create", path)
	return nil
}

func createAdminMiddlewareFile() error {
	path := filepath.Join("middleware", "admin.go")
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("package middleware\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"net/http\"\n")
	buf.WriteString("\t\"monolith/db\"\n")
	buf.WriteString("\t\"monolith/models\"\n")
	buf.WriteString("\t\"monolith/session\"\n")
	buf.WriteString(")\n\n")
	buf.WriteString("// RequireAdmin ensures the user is logged in and an admin before accessing a route\n")
	buf.WriteString("func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {\n")
	buf.WriteString("\treturn func(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\t\tif !session.IsLoggedIn(r) {\n")
	buf.WriteString("\t\t\thttp.Redirect(w, r, \"/login\", http.StatusSeeOther)\n")
	buf.WriteString("\t\t\treturn\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t\temail := session.GetEmail(r)\n")
	buf.WriteString("\t\tuser, err := models.GetUser(db.GetDB(), email)\n")
	buf.WriteString("\t\tif err != nil || !user.IsAdmin {\n")
	buf.WriteString("\t\t\thttp.Redirect(w, r, \"/\", http.StatusSeeOther)\n")
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

func createAdminMiddlewareTestFile() error {
	path := filepath.Join("middleware", "admin_test.go")
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("package middleware\n\n")
	buf.WriteString("import \"testing\"\n\n")
	buf.WriteString("func TestRequireAdminPlaceholder(t *testing.T) {}\n")
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

func updateRoutesForAdmin() error {
	path := filepath.Join("routes", "routes.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	lines = ensureControllersImport(lines)
	lines = ensureMiddlewareImport(lines)
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
	newLines := []string{
		fmt.Sprintf("%smux.HandleFunc(\"GET /admin\", middleware.RequireAdmin(controllers.AdminCtrl.Dashboard))", indent),
		fmt.Sprintf("%smux.HandleFunc(\"POST /admin\", middleware.RequireAdmin(controllers.AdminCtrl.Dashboard))", indent),
	}

	for _, nl := range newLines {
		exists := false
		for _, l := range lines {
			if strings.TrimSpace(l) == strings.TrimSpace(nl) {
				exists = true
				break
			}
		}
		if !exists {
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
