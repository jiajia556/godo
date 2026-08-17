package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNormalizeVersion(t *testing.T) {
	for input, want := range map[string]string{
		"":       "",
		"1.2.0":  "1.2.0",
		"v1.2.0": "1.2.0",
		"dev-42": "dev-42",
	} {
		got, err := normalizeVersion(input)
		if err != nil {
			t.Fatalf("normalizeVersion(%q) error = %v", input, err)
		}
		if got != want {
			t.Fatalf("normalizeVersion(%q) = %q, want %q", input, got, want)
		}
	}
	for _, input := range []string{"v", "1/2", `1\\2`, " 1.2.0", "1.2.0 "} {
		if _, err := normalizeVersion(input); err == nil {
			t.Errorf("normalizeVersion(%q) succeeded", input)
		}
	}
}

func TestBuildOutputName(t *testing.T) {
	tests := []struct {
		options buildOptions
		want    string
	}{
		{buildOptions{CmdName: "api", Version: "1.2.0", GOOS: "linux"}, "api-v1.2.0.bin"},
		{buildOptions{CmdName: "api", Version: "1.2.0", GOOS: "windows"}, "api-v1.2.0.exe"},
		{buildOptions{CmdName: "api", GOOS: "linux"}, "api.bin"},
	}
	for _, test := range tests {
		if got := buildOutputName(test.options); got != test.want {
			t.Errorf("buildOutputName(%+v) = %q, want %q", test.options, got, test.want)
		}
	}
}

func TestValidateBuildTarget(t *testing.T) {
	if err := validateBuildTarget("linux", "amd64"); err != nil {
		t.Fatalf("validateBuildTarget(linux/amd64) error = %v", err)
	}
	if err := validateBuildTarget("not-an-os", "not-an-arch"); err == nil {
		t.Fatal("validateBuildTarget() accepted an unsupported target")
	}
}

func TestResolveBuildOptionsAndBuild(t *testing.T) {
	root := t.TempDir()
	config := `{
  "project_name": "example.com/buildtest",
  "default_cmd": "demo",
  "default_goos": "` + runtime.GOOS + `",
  "default_goarch": "` + runtime.GOARCH + `",
  "cmd_types": {"demo": "worker"}
}`
	if err := os.WriteFile(filepath.Join(root, "godoconfig.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/buildtest\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmdDir := filepath.Join(root, "cmd", "demo")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOD_PROJECT_ROOT", root)

	options, err := resolveBuildOptions("", "v1.2.3", "", "")
	if err != nil {
		t.Fatalf("resolveBuildOptions() error = %v", err)
	}
	if options.CmdName != "demo" || options.CmdType != "worker" || options.Version != "1.2.3" || options.GOOS != runtime.GOOS || options.GOARCH != runtime.GOARCH {
		t.Fatalf("options = %+v", options)
	}
	if err := build(options); err != nil {
		t.Fatalf("build() error = %v", err)
	}
	outputPath := filepath.Join(root, "bin", buildOutputName(options))
	if info, err := os.Stat(outputPath); err != nil || info.IsDir() {
		t.Fatalf("build output %s: info=%v err=%v", outputPath, info, err)
	}

	if _, err := resolveBuildOptions("missing", "", "", ""); err == nil || !strings.Contains(err.Error(), "inspect build package") {
		t.Fatalf("missing package error = %v", err)
	}
	if _, err := resolveBuildOptions("demo", "bad/version", "", ""); err == nil || !strings.Contains(err.Error(), "invalid version") {
		t.Fatalf("invalid version error = %v", err)
	}
	if _, err := resolveBuildOptions("demo", "", "not-an-os", "amd64"); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("invalid target error = %v", err)
	}
}
