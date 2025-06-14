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

// Run dispatches to the specific generator based on args.
// args should not include the leading "generator" argument.
func Run(args []string) error {
	if len(args) == 0 {
		return errors.New("missing generator type")
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
	default:
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

	if err := createModelTestFile(name); err != nil {
		return err
	}
	if err := createControllerTestFile(ctrlName); err != nil {
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
	if err := createAuthControllerFile(); err != nil {
		return err
	}
	if err := createLoginTemplate(); err != nil {
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
	base := "/" + toSnakeCase(name)

	var newLines []string
	newLines = append(newLines, fmt.Sprintf("%s// %sController handles %s actions", indent, ctrlVar, name))
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
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("package models\n\n")
	buf.WriteString("import \"gorm.io/gorm\"\n\n")
	buf.WriteString("// User represents an application user\n")
	buf.WriteString("type User struct {\n")
	buf.WriteString("\tgorm.Model\n")
	buf.WriteString("\tEmail        string `gorm:\"unique;not null\"`\n")
	buf.WriteString("\tPasswordHash []byte\n")
	buf.WriteString("}\n")
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("create", path)
	return updateDBFile("User")
}

// createSessionFile sets up cookie sessions and Google OAuth config.
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
	buf.WriteString("\t\"golang.org/x/oauth2\"\n")
	buf.WriteString("\t\"golang.org/x/oauth2/google\"\n")
	buf.WriteString(")\n\n")
	buf.WriteString("const SESSION_NAME_KEY = \"session\"\n")
	buf.WriteString("const LOGGED_IN_KEY = \"logged_in\"\n")
	buf.WriteString("const EMAIL_KEY = \"email\"\n\n")
	buf.WriteString("var store = sessions.NewCookieStore([]byte(\"super-secret-key\"))\n\n")
	buf.WriteString("var googleOAuthConfig = &oauth2.Config{\n")
	buf.WriteString("\tClientID:     \"YOUR_GOOGLE_CLIENT_ID\",\n")
	buf.WriteString("\tClientSecret: \"YOUR_GOOGLE_CLIENT_SECRET\",\n")
	buf.WriteString("\tRedirectURL:  \"http://localhost:8080/auth/google/callback\",\n")
	buf.WriteString("\tScopes:       []string{\"profile\", \"email\"},\n")
	buf.WriteString("\tEndpoint:     google.Endpoint,\n")
	buf.WriteString("}\n\n")
	buf.WriteString("func GetGoogleOAuthConfig() *oauth2.Config {\n")
	buf.WriteString("\treturn googleOAuthConfig\n")
	buf.WriteString("}\n\n")
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

// createAuthControllerFile creates auth controller handling OAuth callbacks.
func createAuthControllerFile() error {
	path := filepath.Join("controllers", "auth_controller.go")
	if _, err := os.Stat(path); err == nil {
		fmt.Println("exists", path)
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("package controllers\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"encoding/json\"\n")
	buf.WriteString("\t\"errors\"\n")
	buf.WriteString("\t\"net/http\"\n\n")
	buf.WriteString("\t\"monolith/db\"\n")
	buf.WriteString("\t\"monolith/models\"\n")
	buf.WriteString("\t\"monolith/session\"\n")
	buf.WriteString("\t\"monolith/templates\"\n")
	buf.WriteString("\n\t\"gorm.io/gorm\"\n")
	buf.WriteString(")\n\n")
	buf.WriteString("type AuthController struct{}\n\n")
	buf.WriteString("var AuthCtrl = &AuthController{}\n\n")
	buf.WriteString("func (ac *AuthController) ShowLoginForm(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\ttemplates.ExecuteTemplate(w, \"login.html.tmpl\", nil)\n")
	buf.WriteString("}\n\n")
	buf.WriteString("func (ac *AuthController) Logout(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\tsession.Logout(w, r)\n")
	buf.WriteString("\thttp.Redirect(w, r, \"/\", http.StatusSeeOther)\n")
	buf.WriteString("}\n\n")
	buf.WriteString("func (ac *AuthController) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\tconf := session.GetGoogleOAuthConfig()\n")
	buf.WriteString("\turl := conf.AuthCodeURL(\"random-state\")\n")
	buf.WriteString("\thttp.Redirect(w, r, url, http.StatusTemporaryRedirect)\n")
	buf.WriteString("}\n\n")
	buf.WriteString("func (ac *AuthController) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {\n")
	buf.WriteString("\tconf := session.GetGoogleOAuthConfig()\n")
	buf.WriteString("\tcode := r.URL.Query().Get(\"code\")\n\n")
	buf.WriteString("\ttoken, err := conf.Exchange(context.Background(), code)\n")
	buf.WriteString("\tif err != nil {\n")
	buf.WriteString("\t\thttp.Error(w, \"Failed to exchange token\", http.StatusInternalServerError)\n")
	buf.WriteString("\t\treturn\n")
	buf.WriteString("\t}\n\n")
	buf.WriteString("\tclient := conf.Client(context.Background(), token)\n")
	buf.WriteString("\tresp, err := client.Get(\"https://www.googleapis.com/oauth2/v2/userinfo\")\n")
	buf.WriteString("\tif err != nil {\n")
	buf.WriteString("\t\thttp.Error(w, \"Failed to get user info\", http.StatusInternalServerError)\n")
	buf.WriteString("\t\treturn\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\tdefer resp.Body.Close()\n\n")
	buf.WriteString("\tvar userInfo struct {\n")
	buf.WriteString("\t\tEmail     string `json:\"email\"`\n")
	buf.WriteString("\t\tName      string `json:\"name\"`\n")
	buf.WriteString("\t\tAvatarURL string `json:\"picture\"`\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\tjson.NewDecoder(resp.Body).Decode(&userInfo)\n\n")
	buf.WriteString("\tuser, err := models.GetUser(db.GetDB(), userInfo.Email)\n")
	buf.WriteString("\tif errors.Is(err, gorm.ErrRecordNotFound) {\n")
	buf.WriteString("\t\tuser, err = models.CreateUser(db.GetDB(), userInfo.Email, userInfo.Name, userInfo.AvatarURL)\n")
	buf.WriteString("\t\tif err != nil {\n")
	buf.WriteString("\t\t\thttp.Error(w, \"Failed to create user\", http.StatusInternalServerError)\n")
	buf.WriteString("\t\t\treturn\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t} else if err != nil {\n")
	buf.WriteString("\t\thttp.Error(w, \"Database error\", http.StatusInternalServerError)\n")
	buf.WriteString("\t\treturn\n")
	buf.WriteString("\t}\n\n")
	buf.WriteString("\tsession.SetLoggedIn(w, r, user.Email)\n\n")
	buf.WriteString("\thttp.Redirect(w, r, \"/dashboard\", http.StatusSeeOther)\n")
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
	buf.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n    <meta charset=\"UTF-8\">\n    <title>Login</title>\n</head>\n<body>\n    <h1>Login</h1>\n    <a href=\"/auth/google\">\n        <button>Login with Google</button>\n    </a>\n</body>\n</html>\n")
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
		fmt.Sprintf("%smux.HandleFunc(\"GET /auth/google\", controllers.AuthCtrl.HandleGoogleLogin)", indent),
		fmt.Sprintf("%smux.HandleFunc(\"GET /auth/google/callback\", controllers.AuthCtrl.HandleGoogleCallback)", indent),
		fmt.Sprintf("%smux.HandleFunc(\"GET /login\", controllers.AuthCtrl.ShowLoginForm)", indent),
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
