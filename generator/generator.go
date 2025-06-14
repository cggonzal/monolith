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
