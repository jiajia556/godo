package mdw

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMiddlewareNameValidation(t *testing.T) {
	for _, name := range []string{"Auth", "Logging2", "权限", "func"} {
		if err := validateMiddlewareName(name); err != nil {
			t.Fatalf("validateMiddlewareName(%q) error = %v", name, err)
		}
	}
	for _, name := range []string{"../Auth", "Auth-Middleware", "123Auth"} {
		if err := validateMiddlewareName(name); err == nil {
			t.Fatalf("validateMiddlewareName(%q) succeeded", name)
		}
	}
}

func TestGenMiddlewareCreatesFilesAndSkipsExisting(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "godoconfig.json"), []byte(`{"project_name":"example.com/project"}`), 0o644); err != nil {
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

	if err := genMiddleware([]string{"Auth", "RequestLog"}); err != nil {
		t.Fatalf("genMiddleware() error = %v", err)
	}
	for _, name := range []string{"auth", "requestlog"} {
		path := filepath.Join(root, "internal", "common", "transport", "http", "middleware", name+".go")
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), "package middleware") {
			t.Fatalf("generated middleware %s is invalid: %s", name, content)
		}
	}
	if len(formatted) != 2 {
		t.Fatalf("formatted files = %v", formatted)
	}
	if err := genMiddleware([]string{"Auth"}); err != nil {
		t.Fatalf("duplicate genMiddleware() error = %v", err)
	}
	if len(formatted) != 2 {
		t.Fatalf("existing middleware was formatted again: %v", formatted)
	}
	if err := genMiddleware([]string{"../bad"}); err == nil {
		t.Fatal("genMiddleware() accepted invalid name")
	}
}
