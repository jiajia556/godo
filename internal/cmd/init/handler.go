package init

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jiajia556/godo/internal/template"
	"github.com/jiajia556/godo/internal/utils"
	"github.com/jiajia556/godo/templates"
)

func initProject(name string) error {
	if err := generateProject(name); err != nil {
		return err
	}

	projectDir, err := filepath.Abs(filepath.FromSlash(name))
	if err != nil {
		return fmt.Errorf("resolve generated project directory: %w", err)
	}
	if err := utils.FormatGoFiles(projectDir); err != nil {
		return fmt.Errorf("format generated project: %w", err)
	}

	cmdRunner := utils.NewCommandRunner().WithDir(projectDir)
	if output, err := cmdRunner.RunCommandOutput("go", "mod", "tidy"); err != nil {
		return fmt.Errorf("run go mod tidy: %w\n%s", err, output)
	}
	return nil
}

func generateProject(name string) (err error) {
	targetRoot, err := validateProjectTarget(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return fmt.Errorf("create project directory %s: %w", targetRoot, err)
	}

	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(targetRoot)
		}
	}()

	templateDir := templates.DEFAULT_TEMPLATE_DIR
	err = fs.WalkDir(templates.TemplateFS, templateDir, func(originalPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d == nil {
			return fmt.Errorf("walk template %s: missing directory entry", originalPath)
		}
		if d.IsDir() {
			return nil
		}

		relativePath := strings.TrimPrefix(originalPath, templateDir)
		relativePath = strings.TrimPrefix(relativePath, "/")
		path := filepath.Join(targetRoot, filepath.FromSlash(relativePath))
		dirPath := filepath.Dir(path)
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dirPath, err)
		}

		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}
		targetPath := path[:len(path)-5]

		contentByte, err := fs.ReadFile(templates.TemplateFS, originalPath)
		if err != nil {
			return fmt.Errorf("read embedded template %s: %w", originalPath, err)
		}
		content := string(contentByte)

		fileName := filepath.Base(targetPath)

		projectNameTemplates := []string{
			"go.mod", "main.go",
			"godoconfig.json",
			"outputmsg.go",
			"config.go",
		}
		if slices.Contains(projectNameTemplates, fileName) {
			data := template.ProjectNameData{ProjectName: name, CmdName: "default-api"}
			if err := template.CreateFile(content, data, targetPath); err != nil {
				return fmt.Errorf("render template %s: %w", originalPath, err)
			}
		} else {
			if err := os.WriteFile(targetPath, contentByte, 0o644); err != nil {
				return fmt.Errorf("write generated file %s: %w", targetPath, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("generate project: %w", err)
	}

	complete = true
	return nil
}

func validateProjectTarget(name string) (string, error) {
	if name == "" || strings.TrimSpace(name) != name {
		return "", fmt.Errorf("project name must not be empty or contain leading/trailing whitespace")
	}

	target := filepath.FromSlash(name)
	if filepath.IsAbs(target) || filepath.VolumeName(target) != "" {
		return "", fmt.Errorf("project name must be a relative path: %q", name)
	}
	target = filepath.Clean(target)
	if filepath.ToSlash(target) != name {
		return "", fmt.Errorf("project name must use a clean forward-slash path: %q", name)
	}
	if target == "." || target == ".." || strings.HasPrefix(target, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("project name escapes the current directory: %q", name)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current directory: %w", err)
	}
	target = filepath.Join(cwd, target)
	if _, err := os.Lstat(target); err == nil {
		return "", fmt.Errorf("target already exists: %s", target)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect target %s: %w", target, err)
	}
	return target, nil
}
