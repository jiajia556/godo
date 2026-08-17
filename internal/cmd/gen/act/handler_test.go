package act

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenActionUpdatesExistingController(t *testing.T) {
	root := t.TempDir()
	config := `{
  "project_name": "example.com/project",
  "default_cmd": "api",
  "cmd_types": {"api": "api", "worker": "worker"}
}`
	if err := os.WriteFile(filepath.Join(root, "godoconfig.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"api", "worker"} {
		if err := os.MkdirAll(filepath.Join(root, "cmd", name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	controllerPath := filepath.Join(root, "internal", "api", "transport", "http", "api", "controller", "user.go")
	if err := os.MkdirAll(filepath.Dir(controllerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(controllerPath, []byte("package user\n\ntype UserController struct{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOD_PROJECT_ROOT", root)
	previousFormatter := formatGoFiles
	formatGoFiles = func(...string) error { return nil }
	t.Cleanup(func() { formatGoFiles = previousFormatter })

	if err := genAction("api", "user", []string{"Get:GET", "Delete:DELETE"}); err != nil {
		t.Fatalf("genAction() error = %v", err)
	}
	content, err := os.ReadFile(controllerPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"func (ctrl *UserController) Get", "@http_method GET", "@http_method DELETE"} {
		if !strings.Contains(string(content), expected) {
			t.Errorf("controller does not contain %q:\n%s", expected, content)
		}
	}

	if err := genAction("api", "missing", []string{"Get"}); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("missing controller error = %v", err)
	}
	if err := genAction("worker", "user", []string{"Get"}); err == nil || !strings.Contains(err.Error(), "requires \"api\"") {
		t.Fatalf("worker action error = %v", err)
	}
	if err := genAction("missing", "user", []string{"Get"}); err == nil || !strings.Contains(err.Error(), "command \"missing\" does not exist") {
		t.Fatalf("missing command error = %v", err)
	}
}
