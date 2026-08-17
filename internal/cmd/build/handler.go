package build

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jiajia556/godo/internal/service"
	"github.com/jiajia556/godo/internal/utils"
)

var versionPattern = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z._+\-]*$`)

type buildOptions struct {
	CmdName   string
	CmdType   string
	Version   string
	GOOS      string
	GOARCH    string
	BuildPath string
}

func resolveBuildOptions(cmdName, version, goos, goarch string) (buildOptions, error) {
	var err error
	if cmdName == "" {
		cmdName, err = service.GetDefaultCmd()
		if err != nil {
			return buildOptions{}, fmt.Errorf("get default command: %w", err)
		}
	}
	if err := service.ValidateCmdName(cmdName); err != nil {
		return buildOptions{}, fmt.Errorf("validate command name: %w", err)
	}
	buildPath, err := service.GetAbsPath(filepath.Join("cmd", cmdName))
	if err != nil {
		return buildOptions{}, fmt.Errorf("resolve build package: %w", err)
	}
	info, err := os.Stat(buildPath)
	if err != nil {
		return buildOptions{}, fmt.Errorf("inspect build package %s: %w", buildPath, err)
	}
	if !info.IsDir() {
		return buildOptions{}, fmt.Errorf("build package is not a directory: %s", buildPath)
	}
	cmdType, err := service.GetCmdType(cmdName)
	if err != nil {
		return buildOptions{}, fmt.Errorf("get command type: %w", err)
	}
	if goos == "" {
		goos, err = service.GetDefaultGOOS()
		if err != nil {
			return buildOptions{}, fmt.Errorf("get default GOOS: %w", err)
		}
	}
	if goarch == "" {
		goarch, err = service.GetDefaultGOARCH()
		if err != nil {
			return buildOptions{}, fmt.Errorf("get default GOARCH: %w", err)
		}
	}
	goos = strings.ToLower(strings.TrimSpace(goos))
	goarch = strings.ToLower(strings.TrimSpace(goarch))
	if err := validateBuildTarget(goos, goarch); err != nil {
		return buildOptions{}, err
	}

	version, err = normalizeVersion(version)
	if err != nil {
		return buildOptions{}, err
	}
	return buildOptions{CmdName: cmdName, CmdType: cmdType, Version: version, GOOS: goos, GOARCH: goarch, BuildPath: buildPath}, nil
}

func normalizeVersion(version string) (string, error) {
	if version == "" {
		return "", nil
	}
	if strings.TrimSpace(version) != version {
		return "", fmt.Errorf("version must not contain leading or trailing whitespace")
	}
	version = strings.TrimPrefix(version, "v")
	if version == "" || !versionPattern.MatchString(version) {
		return "", fmt.Errorf("invalid version %q", version)
	}
	return version, nil
}

func validateBuildTarget(goos, goarch string) error {
	return service.ValidateBuildTarget(goos, goarch)
}

func build(options buildOptions) error {
	outName, err := service.GetAbsPath(filepath.Join("bin", buildOutputName(options)))
	if err != nil {
		return fmt.Errorf("resolve build output: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outName), 0o755); err != nil {
		return fmt.Errorf("create build output directory: %w", err)
	}

	output, err := utils.NewCommandRunner().
		WithEnv([]string{"GOOS=" + options.GOOS, "GOARCH=" + options.GOARCH}).
		WithDir(filepath.Dir(options.BuildPath)).
		RunCommandOutput("go", "build", "-o", outName, options.BuildPath)
	if err != nil {
		return fmt.Errorf("build %s for %s/%s: %w\n%s", options.CmdName, options.GOOS, options.GOARCH, err, strings.TrimSpace(output))
	}
	return nil
}

func buildOutputName(options buildOptions) string {
	name := options.CmdName
	if options.Version != "" {
		name += "-v" + options.Version
	}
	if options.GOOS == "windows" {
		return name + ".exe"
	}
	return name + ".bin"
}
