package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const GoImportsVersion = "v0.33.0"

const goImportsBatchSize = 100

// FormatGoFiles runs goimports on Go files found at the supplied paths.
// File paths are formatted directly; directory paths are searched recursively.
func FormatGoFiles(paths ...string) error {
	files, err := collectGoFiles(paths)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}

	goimports, err := ensureGoImports()
	if err != nil {
		return err
	}
	for start := 0; start < len(files); start += goImportsBatchSize {
		end := min(start+goImportsBatchSize, len(files))
		args := append([]string{"-w"}, files[start:end]...)
		output, err := NewCommandRunner().RunCommandOutput(goimports, args...)
		if err != nil {
			return fmt.Errorf("run goimports: %w\n%s", err, strings.TrimSpace(output))
		}
	}
	return nil
}

func collectGoFiles(paths []string) ([]string, error) {
	seen := make(map[string]struct{})
	for _, path := range paths {
		if path == "" {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("inspect goimports path %s: %w", path, err)
		}
		if !info.IsDir() {
			if strings.EqualFold(filepath.Ext(path), ".go") {
				absolute, err := filepath.Abs(path)
				if err != nil {
					return nil, fmt.Errorf("resolve go file %s: %w", path, err)
				}
				seen[absolute] = struct{}{}
			}
			continue
		}

		err = filepath.WalkDir(path, func(current string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() && current != path && (entry.Name() == ".git" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".go") {
				absolute, err := filepath.Abs(current)
				if err != nil {
					return err
				}
				seen[absolute] = struct{}{}
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("find Go files under %s: %w", path, err)
		}
	}

	files := make([]string, 0, len(seen))
	for file := range seen {
		files = append(files, file)
	}
	sort.Strings(files)
	return files, nil
}

func ensureGoImports() (string, error) {
	if executable, err := exec.LookPath("goimports"); err == nil {
		return executable, nil
	}

	module := "golang.org/x/tools/cmd/goimports@" + GoImportsVersion
	output, err := NewCommandRunner().RunCommandOutput("go", "install", module)
	if err != nil {
		return "", fmt.Errorf("install %s: %w\n%s", module, err, strings.TrimSpace(output))
	}

	executable, err := installedGoImportsPath()
	if err != nil {
		return "", err
	}
	return executable, nil
}

func installedGoImportsPath() (string, error) {
	if executable, err := exec.LookPath("goimports"); err == nil {
		return executable, nil
	}

	goBin, err := goEnv("GOBIN")
	if err != nil {
		return "", err
	}
	if goBin == "" {
		goPath, err := goEnv("GOPATH")
		if err != nil {
			return "", err
		}
		goPaths := filepath.SplitList(goPath)
		if len(goPaths) == 0 || goPaths[0] == "" {
			return "", fmt.Errorf("goimports was installed but GOPATH is empty")
		}
		goBin = filepath.Join(goPaths[0], "bin")
	}

	name := "goimports"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	candidate := filepath.Join(goBin, name)
	if info, err := os.Stat(candidate); err != nil || info.IsDir() {
		return "", fmt.Errorf("goimports was installed but executable was not found at %s", candidate)
	}
	return candidate, nil
}

func goEnv(name string) (string, error) {
	output, err := NewCommandRunner().RunCommandOutput("go", "env", name)
	if err != nil {
		return "", fmt.Errorf("read go env %s: %w", name, err)
	}
	return strings.TrimSpace(output), nil
}
