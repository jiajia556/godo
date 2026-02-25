package utils

import (
	"fmt"
	"os"
	"os/exec"
)

type CommandRunner struct {
	Env     []string
	Dir     string
	Verbose bool
}

func NewCommandRunner() *CommandRunner {
	return &CommandRunner{
		Verbose: false,
	}
}

func (r *CommandRunner) WithEnv(env []string) *CommandRunner {
	if env == nil {
		r.Env = nil
		return r
	}
	r.Env = append([]string{}, env...)
	return r
}

func (r *CommandRunner) WithDir(dir string) *CommandRunner {
	r.Dir = dir
	return r
}

func (r *CommandRunner) WithVerbose(verbose bool) *CommandRunner {
	r.Verbose = verbose
	return r
}

func (r *CommandRunner) RunCommand(name string, args ...string) {
	out, err := r.RunCommandOutput(name, args...)
	if err != nil {
		// Include command output to make failures easier to diagnose.
		if out != "" {
			OutputFatal(fmt.Sprintf("Command %s failed: %v\nOutput:\n%s", name, err, out))
		} else {
			OutputFatal(fmt.Sprintf("Command %s failed: %v", name, err))
		}
	}
}

func (r *CommandRunner) RunCommandOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if r.Dir != "" {
		cmd.Dir = r.Dir
	}
	if len(r.Env) > 0 {
		cmd.Env = append(os.Environ(), r.Env...)
	}

	if r.Verbose {
		// In verbose/debug scenarios, stream output directly to the console.
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", err
		}
		return "", nil
	}

	// Default mode: capture and return CombinedOutput so callers can attach it to logs/errors.
	outputBytes, err := cmd.CombinedOutput()
	return string(outputBytes), err
}

var defaultRunner = NewCommandRunner()

var GoEnv = []string{}
var CmdDir = ""

func RunCommand(name string, args ...string) {
	defaultRunner.WithEnv(GoEnv).WithDir(CmdDir).RunCommand(name, args...)
}

func RunCommandOutput(name string, args ...string) (string, error) {
	return defaultRunner.WithEnv(GoEnv).WithDir(CmdDir).RunCommandOutput(name, args...)
}
