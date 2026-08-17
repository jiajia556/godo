package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestMalformedNearestConfigIsNotIgnored(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/fallback\n")
	writeTestFile(t, filepath.Join(root, "godoconfig.json"), "{")
	prepareConfigTest(t, root, "")

	_, err := GetProjectName()
	if err == nil {
		t.Fatal("GetProjectName() ignored a malformed godoconfig.json")
	}
	if !strings.Contains(err.Error(), "godoconfig.json") {
		t.Fatalf("error does not identify godoconfig.json: %v", err)
	}
}

func TestGODProjectRootIsAuthoritative(t *testing.T) {
	workingRoot := t.TempDir()
	configuredRoot := t.TempDir()
	writeTestFile(t, filepath.Join(workingRoot, "go.mod"), "module example.com/working\n")
	writeTestFile(t, filepath.Join(configuredRoot, "godoconfig.json"), `{"project_name":"example.com/configured"}`)
	prepareConfigTest(t, workingRoot, configuredRoot)

	name, err := GetProjectName()
	if err != nil {
		t.Fatalf("GetProjectName() error = %v", err)
	}
	if name != "example.com/configured" {
		t.Fatalf("project name = %q, want configured root project", name)
	}
	root, err := GetProjectRoot()
	if err != nil {
		t.Fatalf("GetProjectRoot() error = %v", err)
	}
	if root != filepath.Clean(configuredRoot) {
		t.Fatalf("project root = %q, want %q", root, configuredRoot)
	}
}

func TestDefaultCommandPathsUseEffectiveDefault(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/project\n")
	prepareConfigTest(t, root, "")

	cmdPath, err := GetDefaultCmdCmd()
	if err != nil {
		t.Fatalf("GetDefaultCmdCmd() error = %v", err)
	}
	want := filepath.Join(root, "cmd", "default-api")
	if cmdPath != want {
		t.Fatalf("default command path = %q, want %q", cmdPath, want)
	}
}

func TestConcurrentConfigReads(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/concurrent\n")
	prepareConfigTest(t, root, "")

	var wait sync.WaitGroup
	errors := make(chan error, 32)
	for i := 0; i < 32; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, err := GetProjectName(); err != nil {
				errors <- err
			}
		}()
	}
	wait.Wait()
	close(errors)
	for err := range errors {
		t.Errorf("concurrent GetProjectName() error = %v", err)
	}
}

func TestCmdTypesDefaultAndPersistence(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "godoconfig.json"), `{
  "project_name": "example.com/project",
  "default_cmd": "default-api",
  "cmd_types": {"default-api": "api"}
}`)
	prepareConfigTest(t, root, "")

	legacyType, err := GetCmdType("legacy-api")
	if err != nil || legacyType != CmdTypeAPI {
		t.Fatalf("legacy command type = %q, %v; want api", legacyType, err)
	}
	if err := SetCmdType("order-worker", CmdTypeWorker); err != nil {
		t.Fatalf("SetCmdType() error = %v", err)
	}
	cmdType, err := GetCmdType("order-worker")
	if err != nil || cmdType != CmdTypeWorker {
		t.Fatalf("worker command type = %q, %v; want worker", cmdType, err)
	}

	data, err := os.ReadFile(filepath.Join(root, "godoconfig.json"))
	if err != nil {
		t.Fatal(err)
	}
	var persisted GodoConfig
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("parse persisted config: %v", err)
	}
	if persisted.CmdTypes["default-api"] != CmdTypeAPI || persisted.CmdTypes["order-worker"] != CmdTypeWorker {
		t.Fatalf("persisted cmd_types = %v", persisted.CmdTypes)
	}
}

func TestNormalizeCmdType(t *testing.T) {
	for input, want := range map[string]string{"api": CmdTypeAPI, " API ": CmdTypeAPI, "worker": CmdTypeWorker} {
		got, err := NormalizeCmdType(input)
		if err != nil || got != want {
			t.Errorf("NormalizeCmdType(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	if _, err := NormalizeCmdType("cron"); err == nil {
		t.Fatal("NormalizeCmdType() accepted an unsupported type")
	}
}

func TestSetConfigValue(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "godoconfig.json"), `{
  "project_name": "example.com/project",
  "default_cmd": "default-api",
  "default_goos": "linux",
  "default_goarch": "amd64",
  "cmd_types": {"default-api": "api", "jobs-worker": "worker"}
}`)
	if err := os.MkdirAll(filepath.Join(root, "cmd", "jobs-worker"), 0o755); err != nil {
		t.Fatal(err)
	}
	prepareConfigTest(t, root, "")

	value, err := SetConfigValue(ConfigKeyDefaultCmd, "jobs-worker")
	if err != nil || value != "jobs-worker" {
		t.Fatalf("SetConfigValue(default_cmd) = %q, %v", value, err)
	}
	value, err = SetConfigValue(ConfigKeyDefaultGOOS, " WINDOWS ")
	if err != nil || value != "windows" {
		t.Fatalf("SetConfigValue(default_goos) = %q, %v", value, err)
	}
	value, err = SetConfigValue(ConfigKeyDefaultGOARCH, "arm64")
	if err != nil || value != "arm64" {
		t.Fatalf("SetConfigValue(default_goarch) = %q, %v", value, err)
	}

	data, err := os.ReadFile(filepath.Join(root, "godoconfig.json"))
	if err != nil {
		t.Fatal(err)
	}
	var persisted GodoConfig
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("parse persisted config: %v", err)
	}
	if persisted.ProjectName != "example.com/project" || persisted.DefaultCmd != "jobs-worker" || persisted.DefaultGOOS != "windows" || persisted.DefaultGOARCH != "arm64" {
		t.Fatalf("persisted config = %+v", persisted)
	}
	if persisted.CmdTypes["jobs-worker"] != CmdTypeWorker {
		t.Fatalf("cmd_types was not preserved: %v", persisted.CmdTypes)
	}
}

func TestSetConfigValueRejectsReadOnlyAndInvalidValues(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "godoconfig.json"), `{
  "project_name": "example.com/project",
  "default_cmd": "default-api",
  "default_goos": "linux",
  "default_goarch": "amd64"
}`)
	prepareConfigTest(t, root, "")

	for _, key := range []string{"project_name", "cmd_types", "unknown"} {
		if _, err := SetConfigValue(key, "value"); err == nil {
			t.Errorf("SetConfigValue(%q) modified a read-only or unknown key", key)
		}
	}
	if _, err := SetConfigValue(ConfigKeyDefaultCmd, "missing-api"); err == nil {
		t.Fatal("SetConfigValue(default_cmd) accepted a missing command")
	}
	if _, err := SetConfigValue(ConfigKeyDefaultGOOS, "not-an-os"); err == nil {
		t.Fatal("SetConfigValue(default_goos) accepted an unsupported target")
	}
}

func TestSetBuildTargetAtomically(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "godoconfig.json")
	writeTestFile(t, configPath, `{
  "project_name": "example.com/project",
  "default_cmd": "default-api",
  "default_goos": "linux",
  "default_goarch": "amd64",
  "cmd_types": {"default-api": "api"}
}`)
	prepareConfigTest(t, root, "")

	goos, goarch, err := SetBuildTarget(" JS ", " WASM ")
	if err != nil {
		t.Fatalf("SetBuildTarget(js/wasm) error = %v", err)
	}
	if goos != "js" || goarch != "wasm" {
		t.Fatalf("SetBuildTarget() = %s/%s, want js/wasm", goos, goarch)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var persisted GodoConfig
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("parse persisted config: %v", err)
	}
	if persisted.DefaultGOOS != "js" || persisted.DefaultGOARCH != "wasm" {
		t.Fatalf("persisted target = %s/%s", persisted.DefaultGOOS, persisted.DefaultGOARCH)
	}
	if persisted.ProjectName != "example.com/project" || persisted.CmdTypes["default-api"] != CmdTypeAPI {
		t.Fatalf("unrelated config was changed: %+v", persisted)
	}

	if _, _, err := SetBuildTarget("not-an-os", "not-an-arch"); err == nil {
		t.Fatal("SetBuildTarget() accepted an unsupported target")
	}
	currentGOOS, err := GetDefaultGOOS()
	if err != nil {
		t.Fatal(err)
	}
	currentGOARCH, err := GetDefaultGOARCH()
	if err != nil {
		t.Fatal(err)
	}
	if currentGOOS != "js" || currentGOARCH != "wasm" {
		t.Fatalf("invalid update changed in-memory target to %s/%s", currentGOOS, currentGOARCH)
	}
}

func prepareConfigTest(t *testing.T, workingDirectory, configuredRoot string) {
	t.Helper()
	previousDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	previousConfig := godoConfig
	previousRoot := projectRoot
	godoConfig = GodoConfig{}
	projectRoot = ""
	mu.Unlock()
	previousEnv, hadEnv := os.LookupEnv("GOD_PROJECT_ROOT")

	if err := os.Chdir(workingDirectory); err != nil {
		t.Fatal(err)
	}
	if configuredRoot == "" {
		if err := os.Unsetenv("GOD_PROJECT_ROOT"); err != nil {
			t.Fatal(err)
		}
	} else if err := os.Setenv("GOD_PROJECT_ROOT", configuredRoot); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(previousDirectory)
		if hadEnv {
			_ = os.Setenv("GOD_PROJECT_ROOT", previousEnv)
		} else {
			_ = os.Unsetenv("GOD_PROJECT_ROOT")
		}
		mu.Lock()
		godoConfig = previousConfig
		projectRoot = previousRoot
		mu.Unlock()
	})
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
