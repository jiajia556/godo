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
		Verbose: os.Getenv("GOD_VERBOSE") == "0",
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
		// 包含命令输出便于排查
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
		// 在交互式/调试场景下直接把输出流到控制台
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", err
		}
		return "", nil
	}

	// 默认模式：捕获并返回 CombinedOutput，便于日志/错误中包含详细信息
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
