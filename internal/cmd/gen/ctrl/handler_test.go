package ctrl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenCtrlCreatesControllerAndRejectsInvalidTargets(t *testing.T) {
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
	t.Setenv("GOD_PROJECT_ROOT", root)
	previousFormatter := formatGoFiles
	var formatted []string
	formatGoFiles = func(paths ...string) error {
		formatted = append(formatted, paths...)
		return nil
	}
	t.Cleanup(func() { formatGoFiles = previousFormatter })

	if err := genCtrl("api", "admin/user", []string{"List:GET", "Remove:DELETE"}); err != nil {
		t.Fatalf("genCtrl() error = %v", err)
	}
	controllerPath := filepath.Join(root, "internal", "api", "transport", "http", "api", "admin", "controller", "user.go")
	content, err := os.ReadFile(controllerPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"type UserController", "func (ctrl *UserController) List", "@http_method GET", "@http_method DELETE"} {
		if !strings.Contains(string(content), expected) {
			t.Errorf("controller does not contain %q:\n%s", expected, content)
		}
	}
	if len(formatted) != 1 || formatted[0] != controllerPath {
		t.Fatalf("formatted paths = %v", formatted)
	}

	if err := genCtrl("api", "admin/user", nil); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate controller error = %v", err)
	}
	if err := genCtrl("missing", "user", nil); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("missing command error = %v", err)
	}
	if err := genCtrl("worker", "user", nil); err == nil || !strings.Contains(err.Error(), "requires \"api\"") {
		t.Fatalf("worker controller error = %v", err)
	}
	if err := genCtrl("../bad", "user", nil); err == nil || !strings.Contains(err.Error(), "validate command") {
		t.Fatalf("invalid command error = %v", err)
	}
}
