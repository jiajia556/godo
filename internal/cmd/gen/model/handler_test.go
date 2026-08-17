package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateModelFromSQLReturnsParseError(t *testing.T) {
	_, err := generateModelFromSQL("not a CREATE TABLE statement", "", "", "")
	if err == nil {
		t.Fatal("generateModelFromSQL() succeeded for invalid SQL")
	}
}

func TestGenerateModelFromSQLCreatesFilesAndSkipsExisting(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "godoconfig.json"), []byte(`{"project_name":"example.com/project"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOD_PROJECT_ROOT", root)
	sql := "CREATE TABLE `users` (`id` bigint NOT NULL, `name` varchar(64) NOT NULL, PRIMARY KEY (`id`));"
	recordTemplate := "package {{.ModelPkg}}\n\n{{.ModelStruct}}\n"
	listTemplate := "package {{.ModelPkg}}\n\ntype {{.ModelStructName}}List []{{.ModelStructName}}\n"
	modelTemplate := "package {{.ModelPkg}}\n\nconst TableName = {{printf \"%q\" .TableName}}\n"

	files, err := generateModelFromSQL(sql, recordTemplate, listTemplate, modelTemplate)
	if err != nil {
		t.Fatalf("generateModelFromSQL() error = %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("generated files = %v", files)
	}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), "package users") {
			t.Fatalf("generated model is invalid: %s", content)
		}
	}
	record, err := os.ReadFile(filepath.Join(root, "internal", "common", "models", "users", "record.go"))
	if err != nil || !strings.Contains(string(record), "type Users struct") {
		t.Fatalf("record content = %s, err = %v", record, err)
	}

	files, err = generateModelFromSQL(sql, recordTemplate, listTemplate, modelTemplate)
	if err != nil || len(files) != 0 {
		t.Fatalf("second generation = %v, %v", files, err)
	}
}

func TestGenModelReportsInputErrors(t *testing.T) {
	root := t.TempDir()
	emptySQL := filepath.Join(root, "empty.sql")
	if err := os.WriteFile(emptySQL, []byte("SELECT 1;"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := genModel(emptySQL); err == nil || !strings.Contains(err.Error(), "no CREATE TABLE") {
		t.Fatalf("genModel(empty SQL) error = %v", err)
	}
	if err := genModel(filepath.Join(root, "missing.sql")); err == nil || !strings.Contains(err.Error(), "read SQL file") {
		t.Fatalf("genModel(missing SQL) error = %v", err)
	}
	invalidConfig := filepath.Join(root, "invalid.json")
	if err := os.WriteFile(invalidConfig, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := genModel(invalidConfig); err == nil || !strings.Contains(err.Error(), "parse json") {
		t.Fatalf("genModel(invalid config) error = %v", err)
	}
}
