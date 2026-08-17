package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiajia556/godo/internal/service"
	"github.com/jiajia556/godo/internal/template"
	"github.com/jiajia556/godo/internal/utils"
	"github.com/jiajia556/godo/templates"
)

type cmdTemplateSpec struct {
	root        string
	placeholder string
	directories []string
}

var formatGoFiles = utils.FormatGoFiles

func genCmd(cmdName, cmdType string) (err error) {
	if err := validateCmdName(cmdName); err != nil {
		return err
	}
	cmdType, err = service.NormalizeCmdType(cmdType)
	if err != nil {
		return err
	}
	templateSpec := templateForCmdType(cmdType)

	cmdRoot, err := service.GetAbsPath(filepath.Join("cmd", cmdName))
	if err != nil {
		return fmt.Errorf("resolve command directory: %w", err)
	}
	internalRoot, err := service.GetAbsPath(filepath.Join("internal", cmdName))
	if err != nil {
		return fmt.Errorf("resolve internal command directory: %w", err)
	}
	for _, target := range []string{cmdRoot, internalRoot} {
		if _, statErr := os.Lstat(target); statErr == nil {
			return fmt.Errorf("command target already exists: %s", target)
		} else if !os.IsNotExist(statErr) {
			return fmt.Errorf("inspect command target %s: %w", target, statErr)
		}
	}

	projectName, err := service.GetProjectName()
	if err != nil {
		return fmt.Errorf("get project name: %w", err)
	}

	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(cmdRoot)
			_ = os.RemoveAll(internalRoot)
		}
	}()

	var generatedFiles []string
	for _, cmdDir := range templateSpec.directories {
		if err := fs.WalkDir(templates.TemplateFS, cmdDir, func(originalPath string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d == nil {
				return fmt.Errorf("walk template %s: missing directory entry", originalPath)
			}
			if d.IsDir() {
				return nil
			}

			relativePath := strings.TrimPrefix(originalPath, templateSpec.root)
			relativePath = strings.TrimPrefix(relativePath, "/")
			relativePath = strings.ReplaceAll(relativePath, templateSpec.placeholder, cmdName)
			path, err := service.GetAbsPath(filepath.FromSlash(relativePath))
			if err != nil {
				return fmt.Errorf("resolve generated path %s: %w", originalPath, err)
			}
			dirPath := filepath.Dir(path)
			if err = os.MkdirAll(dirPath, 0o755); err != nil {
				return fmt.Errorf("create generated directory %s: %w", dirPath, err)
			}

			if !strings.HasSuffix(path, ".tmpl") {
				return nil
			}
			targetPath := path[:len(path)-5]

			contentByte, err := fs.ReadFile(templates.TemplateFS, originalPath)
			if err != nil {
				return fmt.Errorf("read embedded template %s: %w", originalPath, err)
			}
			data := template.ProjectNameData{ProjectName: projectName, CmdName: cmdName}
			if err = template.CreateFile(string(contentByte), data, targetPath); err != nil {
				return fmt.Errorf("render template %s: %w", originalPath, err)
			}
			if strings.HasSuffix(targetPath, ".go") {
				generatedFiles = append(generatedFiles, targetPath)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("generate command %q: %w", cmdName, err)
		}
	}
	if err := formatGoFiles(generatedFiles...); err != nil {
		return fmt.Errorf("format generated command: %w", err)
	}
	if err := service.SetCmdType(cmdName, cmdType); err != nil {
		return fmt.Errorf("record command type: %w", err)
	}
	complete = true
	return nil
}

func templateForCmdType(cmdType string) cmdTemplateSpec {
	if cmdType == service.CmdTypeWorker {
		return cmdTemplateSpec{
			root:        "worker",
			placeholder: "default-worker",
			directories: []string{"worker/cmd/default-worker", "worker/internal/default-worker"},
		}
	}
	return cmdTemplateSpec{
		root:        templates.DEFAULT_TEMPLATE_DIR,
		placeholder: "default-api",
		directories: []string{"default/cmd/default-api", "default/internal/default-api"},
	}
}

func validateCmdName(name string) error {
	return service.ValidateCmdName(name)
}
