package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandRunnerHelperProcess(t *testing.T) {
	if os.Getenv("GODO_HELPER_PROCESS") != "1" {
		return
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(2)
	}
	fmt.Printf("%s|%s", os.Getenv("GODO_HELPER_VALUE"), workingDirectory)
	if os.Getenv("GODO_HELPER_FAIL") == "1" {
		os.Exit(3)
	}
	os.Exit(0)
}

func TestCommandRunnerEnvironmentDirectoryAndErrors(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	env := []string{"GODO_HELPER_PROCESS=1", "GODO_HELPER_VALUE=visible"}
	runner := NewCommandRunner().WithEnv(env).WithDir(dir)
	env[1] = "GODO_HELPER_VALUE=mutated"

	output, err := runner.RunCommandOutput(executable, "-test.run=^TestCommandRunnerHelperProcess$")
	if err != nil {
		t.Fatalf("RunCommandOutput() error = %v, output = %s", err, output)
	}
	wantSuffix := "visible|" + dir
	if !strings.Contains(output, wantSuffix) {
		t.Fatalf("output = %q, want substring %q", output, wantSuffix)
	}

	runner.WithEnv([]string{"GODO_HELPER_PROCESS=1", "GODO_HELPER_FAIL=1"})
	if _, err := runner.RunCommandOutput(executable, "-test.run=^TestCommandRunnerHelperProcess$"); err == nil {
		t.Fatal("RunCommandOutput() succeeded for failing helper")
	}
	if runner.WithEnv(nil).Env != nil {
		t.Fatal("WithEnv(nil) did not clear environment overrides")
	}
	if !runner.WithVerbose(true).Verbose || runner.WithVerbose(false).Verbose {
		t.Fatal("WithVerbose() did not update runner")
	}
}

func TestPackageCommandRunnerUsesConfiguredDirectory(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	previousEnv, previousDir := GoEnv, CmdDir
	t.Cleanup(func() {
		GoEnv, CmdDir = previousEnv, previousDir
	})
	GoEnv = []string{"GODO_HELPER_PROCESS=1", "GODO_HELPER_VALUE=package"}
	CmdDir = t.TempDir()

	output, err := RunCommandOutput(executable, "-test.run=^TestCommandRunnerHelperProcess$")
	if err != nil {
		t.Fatalf("RunCommandOutput() error = %v", err)
	}
	if !strings.Contains(output, "package|"+filepath.Clean(CmdDir)) {
		t.Fatalf("output = %q", output)
	}
}
