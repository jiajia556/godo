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

type GoDoPackage struct {
	inited        bool
	ProjectName   string `json:"project_name"`
	DefaultApi    string `json:"default_api"`
	DefaultGOOS   string `json:"default_goos"`
	DefaultGOARCH string `json:"default_goarch"`
}

var (
	godoPackage GoDoPackage
	projectRoot string // the directory where gopackage.json or go.mod was found
	mu          sync.Mutex
)

// initGoPackage locates and loads gopackage.json or falls back to go.mod module.
// Behavior:
// 1. If env GOD_PROJECT_ROOT is set, try that directory first.
// 2. Otherwise, walk up from cwd searching for gopackage.json; if not found, use go.mod to derive module.
// On success, sets godoPackage and projectRoot.
func initGoPackage() error {
	mu.Lock()
	defer mu.Unlock()

	if godoPackage.inited {
		return nil
	}

	// 2) Walk up from cwd to root looking for gopackage.json or go.mod
	startDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot get working directory: %w", err)
	}

	var triedPaths []string
	dir := startDir
	for {
		tryPkg := filepath.Join(dir, "gopackage.json")
		triedPaths = append(triedPaths, tryPkg)
		if err := loadFromFileIfExists(tryPkg); err == nil {
			projectRoot = filepath.Clean(dir)
			godoPackage.inited = true
			return nil
		}

		// If gopackage.json not found, check go.mod as fallback
		tryMod := filepath.Join(dir, "go.mod")
		triedPaths = append(triedPaths, tryMod)
		if exists(tryMod) {
			if err := loadFromGoMod(tryMod); err == nil {
				projectRoot = filepath.Clean(dir)
				godoPackage.inited = true
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

	return fmt.Errorf("could not find gopackage.json nor parse go.mod module; attempted: %s", strings.Join(triedPaths, "; "))
}

func GetProjectName() (string, error) {
	if !godoPackage.inited {
		if err := initGoPackage(); err != nil {
			return "", err
		}
	}
	if godoPackage.ProjectName == "" {
		return "", errors.New("project name is empty in gopackage.json or go.mod")
	}
	return godoPackage.ProjectName, nil
}

// GetDefaultApiCmd returns the absolute path to the default API command directory as specified in gopackage.json.
func GetDefaultApiCmd() (string, error) {
	if !godoPackage.inited {
		if err := initGoPackage(); err != nil {
			return "", err
		}
	}
	return resolvePath("cmd/" + godoPackage.DefaultApi)
}

// GetDefaultApiInternal returns the absolute path to the default API internal directory as specified in gopackage.json.
func GetDefaultApiInternal() (string, error) {
	if !godoPackage.inited {
		if err := initGoPackage(); err != nil {
			return "", err
		}
	}
	return resolvePath("internal/" + godoPackage.DefaultApi)
}

// GetAbsPath resolves the given path to an absolute path based on projectRoot if it's not already absolute.
func GetAbsPath(path string) (string, error) {
	if !godoPackage.inited {
		if err := initGoPackage(); err != nil {
			return "", err
		}
	}
	return resolvePath(path)
}

// loadFromFileIfExists tries to read and unmarshal the given path if it exists.
// On success, it fills godoPackage (but does not set projectRoot - caller must set it).
func loadFromFileIfExists(path string) error {
	if !exists(path) {
		return fmt.Errorf("not found: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &godoPackage); err != nil {
		return fmt.Errorf("unmarshal %s: %w", path, err)
	}
	ensureDefaults(&godoPackage)
	return nil
}

// loadFromGoMod parses module name from go.mod and fills godoPackage.ProjectName.
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
				godoPackage.ProjectName = parts[1]
				ensureDefaults(&godoPackage)
				return nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan go.mod %s: %w", modPath, err)
	}
	return errors.New("module directive not found in go.mod")
}

func ensureDefaults(gp *GoDoPackage) {
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
	// ensure gopackage initialized so projectRoot may be set
	if !godoPackage.inited {
		if err := initGoPackage(); err != nil {
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
