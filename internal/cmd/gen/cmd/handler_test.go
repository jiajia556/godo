package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	texttemplate "text/template"

	"github.com/jiajia556/godo/internal/service"
	projecttemplate "github.com/jiajia556/godo/internal/template"
	"github.com/jiajia556/godo/templates"
)

func TestValidateCmdName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "admin-api"},
		{name: "worker_v2"},
		{name: "", wantErr: true},
		{name: "../outside", wantErr: true},
		{name: "nested/worker", wantErr: true},
		{name: " worker", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateCmdName(test.name)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateCmdName(%q) error = %v, wantErr %v", test.name, err, test.wantErr)
			}
		})
	}
}

func TestTemplateForCmdType(t *testing.T) {
	api := templateForCmdType(service.CmdTypeAPI)
	if api.placeholder != "default-api" || len(api.directories) != 2 {
		t.Fatalf("API template spec = %+v", api)
	}
	worker := templateForCmdType(service.CmdTypeWorker)
	if worker.placeholder != "default-worker" || len(worker.directories) != 2 {
		t.Fatalf("worker template spec = %+v", worker)
	}
}

func TestWorkerGoTemplatesRenderValidSyntax(t *testing.T) {
	paths := []string{
		"worker/cmd/default-worker/main.go.tmpl",
		"worker/internal/default-worker/config/config.go.tmpl",
		"worker/internal/default-worker/worker/execute.go.tmpl",
	}
	data := projecttemplate.ProjectNameData{ProjectName: "example.com/project", CmdName: "order-worker"}
	for _, templatePath := range paths {
		t.Run(filepath.Base(templatePath), func(t *testing.T) {
			content, err := templates.TemplateFS.ReadFile(templatePath)
			if err != nil {
				t.Fatal(err)
			}
			tmpl, err := texttemplate.New(templatePath).Parse(string(content))
			if err != nil {
				t.Fatalf("parse template: %v", err)
			}
			var rendered bytes.Buffer
			if err := tmpl.Execute(&rendered, data); err != nil {
				t.Fatalf("render template: %v", err)
			}
			if _, err := parser.ParseFile(token.NewFileSet(), strings.TrimSuffix(templatePath, ".tmpl"), rendered.Bytes(), parser.AllErrors); err != nil {
				t.Fatalf("generated Go syntax is invalid: %v\n%s", err, rendered.String())
			}
		})
	}
}

func TestGenCmdGeneratesAPIAndWorkerAndPersistsTypes(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "godoconfig.json")
	config := `{
  "project_name": "example.com/project",
  "default_cmd": "default-api",
  "default_goos": "linux",
  "default_goarch": "amd64",
  "cmd_types": {}
}`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOD_PROJECT_ROOT", root)
	previousFormatter := formatGoFiles
	var formatted []string
	formatGoFiles = func(paths ...string) error {
		formatted = append(formatted, paths...)
		return nil
	}
	t.Cleanup(func() { formatGoFiles = previousFormatter })

	if err := genCmd("orders-worker", service.CmdTypeWorker); err != nil {
		t.Fatalf("genCmd(worker) error = %v", err)
	}
	workerMain := filepath.Join(root, "cmd", "orders-worker", "main.go")
	workerExecute := filepath.Join(root, "internal", "orders-worker", "worker", "execute.go")
	for _, path := range []string{workerMain, workerExecute, filepath.Join(root, "internal", "orders-worker", "config", "config.go")} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("generated worker file %s: %v", path, err)
		}
	}
	mainContent, err := os.ReadFile(workerMain)
	if err != nil || !strings.Contains(string(mainContent), "runner.New(workerInterval, worker.Execute)") {
		t.Fatalf("worker main content is incomplete: %v\n%s", err, mainContent)
	}

	if err := genCmd("admin-api", service.CmdTypeAPI); err != nil {
		t.Fatalf("genCmd(api) error = %v", err)
	}
	apiMain := filepath.Join(root, "cmd", "admin-api", "main.go")
	apiConfig := filepath.Join(root, "internal", "admin-api", "config", "config.go")
	for _, path := range []string{apiMain, apiConfig} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("generated API file %s: %v", path, err)
		}
	}
	if len(formatted) == 0 {
		t.Fatal("generated Go files were not passed to formatter")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var persisted service.GodoConfig
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.CmdTypes["orders-worker"] != service.CmdTypeWorker || persisted.CmdTypes["admin-api"] != service.CmdTypeAPI {
		t.Fatalf("persisted command types = %v", persisted.CmdTypes)
	}

	if err := genCmd("orders-worker", service.CmdTypeWorker); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate genCmd() error = %v", err)
	}
	if err := genCmd("bad", "cron"); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unsupported genCmd() error = %v", err)
	}

	formatGoFiles = func(...string) error { return fmt.Errorf("format failed") }
	err = genCmd("rollback-worker", service.CmdTypeWorker)
	if err == nil || !strings.Contains(err.Error(), "format failed") {
		t.Fatalf("genCmd() error = %v", err)
	}
	for _, path := range []string{filepath.Join(root, "cmd", "rollback-worker"), filepath.Join(root, "internal", "rollback-worker")} {
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Fatalf("generated path was not rolled back: %s, error = %v", path, statErr)
		}
	}
}
