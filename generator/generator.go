package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode"
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
	if err := ioutil.WriteFile(path, formatted, 0644); err != nil {
		return err
	}
	fmt.Println("create", path)
	return nil
}

// updateDBFile injects the new model into db/db.go AutoMigrate list.
func updateDBFile(modelName string) error {
	path := filepath.Join("db", "db.go")
	data, err := ioutil.ReadFile(path)
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
	if err := ioutil.WriteFile(path, formatted, 0644); err != nil {
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
