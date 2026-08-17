package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigCommandsUpdateWritableValues(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "godoconfig.json")
	config := `{
  "project_name": "example.com/project",
  "default_cmd": "api",
  "default_goos": "linux",
  "default_goarch": "amd64",
  "cmd_types": {"api": "api", "worker": "worker"}
}`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd", "worker"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOD_PROJECT_ROOT", root)

	var output bytes.Buffer
	setCmd.SetOut(&output)
	if err := setCmd.RunE(setCmd, []string{"default_cmd", "worker"}); err != nil {
		t.Fatalf("config set error = %v", err)
	}
	if output.String() != "default_cmd=worker\n" {
		t.Fatalf("config set output = %q", output.String())
	}
	if err := setCmd.RunE(setCmd, []string{"project_name", "changed"}); err == nil || !strings.Contains(err.Error(), "cannot be modified") {
		t.Fatalf("read-only config error = %v", err)
	}

	output.Reset()
	setTargetCmd.SetOut(&output)
	if err := setTargetCmd.RunE(setTargetCmd, []string{"js", "wasm"}); err != nil {
		t.Fatalf("config set-target error = %v", err)
	}
	if output.String() != "default_goos=js\ndefault_goarch=wasm\n" {
		t.Fatalf("config set-target output = %q", output.String())
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"default_cmd": "worker"`, `"default_goos": "js"`, `"default_goarch": "wasm"`, `"project_name": "example.com/project"`} {
		if !strings.Contains(string(data), expected) {
			t.Errorf("persisted config does not contain %s:\n%s", expected, data)
		}
	}
	if GetCommand() != configCmd {
		t.Fatal("GetCommand() returned unexpected command")
	}
}
