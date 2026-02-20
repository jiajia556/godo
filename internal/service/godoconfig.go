package service

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type GodoConfig struct {
	inited        bool
	ProjectName   string `json:"project_name"`
	DefaultCmd    string `json:"default_cmd"`
	DefaultGOOS   string `json:"default_goos"`
	DefaultGOARCH string `json:"default_goarch"`
}

var (
	godoConfig  GodoConfig
	projectRoot string // the directory where godoconfig.json or go.mod was found
	mu          sync.Mutex
)

// initGodoConfig locates and loads godoconfig.json or falls back to go.mod module.
// Behavior:
// 1. If env GOD_PROJECT_ROOT is set, try that directory first.
// 2. Otherwise, walk up from cwd searching for godoconfig.json; if not found, use go.mod to derive module.
// On success, sets godoConfig and projectRoot.
func initGodoConfig() error {
	mu.Lock()
	defer mu.Unlock()

	if godoConfig.inited {
		return nil
	}

	// 2) Walk up from cwd to root looking for godoconfig.json or go.mod
	startDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot get working directory: %w", err)
	}

	var triedPaths []string
	dir := startDir
	for {
		tryPkg := filepath.Join(dir, "godoconfig.json")
		triedPaths = append(triedPaths, tryPkg)
		if err := loadFromFileIfExists(tryPkg); err == nil {
			projectRoot = filepath.Clean(dir)
			godoConfig.inited = true
			return nil
		}

		// If godoconfig.json not found, check go.mod as fallback
		tryMod := filepath.Join(dir, "go.mod")
		triedPaths = append(triedPaths, tryMod)
		if exists(tryMod) {
			if err := loadFromGoMod(tryMod); err == nil {
				projectRoot = filepath.Clean(dir)
				godoConfig.inited = true
				return nil
			}
			// continue walking up if parsing fails
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}

	return fmt.Errorf("could not find godoconfig.json nor parse go.mod module; attempted: %s", strings.Join(triedPaths, "; "))
}

func GetProjectName() (string, error) {
	if !godoConfig.inited {
		if err := initGodoConfig(); err != nil {
			return "", err
		}
	}
	if godoConfig.ProjectName == "" {
		return "", errors.New("project name is empty in godoconfig.json or go.mod")
	}
	return godoConfig.ProjectName, nil
}

func GetDefaultCmd() (string, error) {
	if !godoConfig.inited {
		if err := initGodoConfig(); err != nil {
			return "", err
		}
	}
	if godoConfig.DefaultCmd == "" {
		return "default-api", nil
	}
	return godoConfig.DefaultCmd, nil
}

func GetDefaultGOOS() (string, error) {
	if !godoConfig.inited {
		if err := initGodoConfig(); err != nil {
			return "", err
		}
	}
	if godoConfig.DefaultGOOS == "" {
		return "linux", nil
	}
	return godoConfig.DefaultGOOS, nil
}

func GetDefaultGOARCH() (string, error) {
	if !godoConfig.inited {
		if err := initGodoConfig(); err != nil {
			return "", err
		}
	}
	if godoConfig.DefaultGOARCH == "" {
		return "amd64", nil
	}
	return godoConfig.DefaultGOARCH, nil
}

// GetDefaultCmdCmd returns the absolute path to the default cmd cmd directory as specified in godoconfig.json.
func GetDefaultCmdCmd() (string, error) {
	if !godoConfig.inited {
		if err := initGodoConfig(); err != nil {
			return "", err
		}
	}
	return resolvePath("cmd/" + godoConfig.DefaultCmd)
}

// GetDefaultCmdInternal returns the absolute path to the default cmd internal directory as specified in godoconfig.json.
func GetDefaultCmdInternal() (string, error) {
	if !godoConfig.inited {
		if err := initGodoConfig(); err != nil {
			return "", err
		}
	}
	return resolvePath("internal/" + godoConfig.DefaultCmd)
}

// GetAbsPath resolves the given path to an absolute path based on projectRoot if it's not already absolute.
func GetAbsPath(path string) (string, error) {
	if !godoConfig.inited {
		if err := initGodoConfig(); err != nil {
			return "", err
		}
	}
	return resolvePath(path)
}

// GetProjectRoot returns the absolute path of the discovered project root (where godoconfig.json or go.mod was found).
// If not yet initialized, it will attempt initialization.
func GetProjectRoot() (string, error) {
	if !godoConfig.inited {
		if err := initGodoConfig(); err != nil {
			return "", err
		}
	}
	if projectRoot == "" {
		return "", errors.New("project root is unknown; godoconfig.json and go.mod not found during initialization")
	}
	return projectRoot, nil
}

// loadFromFileIfExists tries to read and unmarshal the given path if it exists.
// On success, it fills godoConfig (but does not set projectRoot - caller must set it).
func loadFromFileIfExists(path string) error {
	if !exists(path) {
		return fmt.Errorf("not found: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &godoConfig); err != nil {
		return fmt.Errorf("unmarshal %s: %w", path, err)
	}
	ensureDefaults(&godoConfig)
	return nil
}

// loadFromGoMod parses module name from go.mod and fills godoConfig.ProjectName.
// Caller should set projectRoot on success.
func loadFromGoMod(modPath string) error {
	f, err := os.Open(modPath)
	if err != nil {
		return fmt.Errorf("open go.mod %s: %w", modPath, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				godoConfig.ProjectName = parts[1]
				ensureDefaults(&godoConfig)
				return nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan go.mod %s: %w", modPath, err)
	}
	return errors.New("module directive not found in go.mod")
}

func ensureDefaults(gp *GodoConfig) {
	if gp.DefaultGOOS == "" {
		gp.DefaultGOOS = "linux"
	}
	if gp.DefaultGOARCH == "" {
		gp.DefaultGOARCH = "amd64"
	}
}

// exists reports whether the named file exists (and is not a directory).
func exists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// resolvePath makes cfgPath absolute based on projectRoot if cfgPath is not already absolute.
// If projectRoot is unknown, resolves relative to current working directory.
func resolvePath(cfgPath string) (string, error) {
	if filepath.IsAbs(cfgPath) {
		return filepath.Clean(cfgPath), nil
	}
	// ensure godoconfig initialized so projectRoot may be set
	if !godoConfig.inited {
		if err := initGodoConfig(); err != nil {
			// fallback: join with cwd
			cwd, _ := os.Getwd()
			return filepath.Clean(filepath.Join(cwd, cfgPath)), nil
		}
	}
	base := projectRoot
	if base == "" {
		// fallback to cwd
		cwd, _ := os.Getwd()
		base = cwd
	}
	return filepath.Clean(filepath.Join(base, cfgPath)), nil
}
