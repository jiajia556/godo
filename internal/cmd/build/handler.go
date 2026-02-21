package build

import (
	"path/filepath"

	"github.com/jiajia556/godo/internal/service"
	"github.com/jiajia556/godo/internal/utils"
)

func build(cmdName, version, goos, goarch string) {
	var err error
	if cmdName == "" {
		cmdName, err = service.GetDefaultCmd()
		if err != nil {
			utils.OutputFatal(err)
		}
	}
	if goos == "" {
		goos, err = service.GetDefaultCmd()
		if err != nil {
			utils.OutputFatal(err)
		}
	}
	if goarch == "" {
		goarch, err = service.GetDefaultCmd()
		if err != nil {
			utils.OutputFatal(err)
		}
	}

	buildPath := filepath.Join("cmd", cmdName)
	buildPath, err = service.GetAbsPath(buildPath)
	if err != nil {
		utils.OutputFatal(err)
	}

	outName := filepath.Join("bin", cmdName)
	outName, err = service.GetAbsPath(outName)
	if err != nil {
		utils.OutputFatal(err)
	}
	if version != "" {
		outName += "-v" + version
	}
	if goos == "windows" {
		outName += ".exe"
	}

	runner := utils.NewCommandRunner()
	runner.WithEnv([]string{"GOOS=" + goos, "GOARCH=" + goarch}).RunCommand("go", "build", "-o", outName, buildPath)
}
