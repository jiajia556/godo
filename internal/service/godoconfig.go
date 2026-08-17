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
	ProjectName   string            `json:"project_name"`
	DefaultCmd    string            `json:"default_cmd"`
	DefaultGOOS   string            `json:"default_goos"`
	DefaultGOARCH string            `json:"default_goarch"`
	CmdTypes      map[string]string `json:"cmd_types,omitempty"`
}

const (
	CmdTypeAPI    = "api"
	CmdTypeWorker = "worker"

	ConfigKeyDefaultCmd    = "default_cmd"
	ConfigKeyDefaultGOOS   = "default_goos"
	ConfigKeyDefaultGOARCH = "default_goarch"
)

var (
	godoConfig  GodoConfig
	projectRoot string
	mu          sync.Mutex
)

// initGodoConfig locates and loads project configuration once.
func initGodoConfig() error {
	_, _, err := getConfigState()
	return err
}

func getConfigState() (GodoConfig, string, error) {
	mu.Lock()
	defer mu.Unlock()

	if !godoConfig.inited {
		if err := initializeConfigLocked(); err != nil {
			return GodoConfig{}, "", err
		}
	}
	return godoConfig, projectRoot, nil
}

func initializeConfigLocked() error {
	godoConfig = GodoConfig{}
	projectRoot = ""

	if configuredRoot := strings.TrimSpace(os.Getenv("GOD_PROJECT_ROOT")); configuredRoot != "" {
		root, err := filepath.Abs(configuredRoot)
		if err != nil {
			return fmt.Errorf("resolve GOD_PROJECT_ROOT %q: %w", configuredRoot, err)
		}
		info, err := os.Stat(root)
		if err != nil {
			return fmt.Errorf("inspect GOD_PROJECT_ROOT %s: %w", root, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("GOD_PROJECT_ROOT is not a directory: %s", root)
		}
		found, err := loadProjectRootLocked(root)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("GOD_PROJECT_ROOT contains neither godoconfig.json nor a valid go.mod: %s", root)
		}
		return nil
	}

	startDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	var triedPaths []string
	for dir := startDir; ; dir = filepath.Dir(dir) {
		triedPaths = append(triedPaths, filepath.Join(dir, "godoconfig.json"), filepath.Join(dir, "go.mod"))
		found, err := loadProjectRootLocked(dir)
		if err != nil {
			return err
		}
		if found {
			return nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}

	return fmt.Errorf("could not find godoconfig.json or go.mod; attempted: %s", strings.Join(triedPaths, "; "))
}

func loadProjectRootLocked(root string) (bool, error) {
	configPath := filepath.Join(root, "godoconfig.json")
	if found, err := regularFileExists(configPath); err != nil {
		return false, err
	} else if found {
		if err := loadFromFile(configPath); err != nil {
			return false, err
		}
		projectRoot = filepath.Clean(root)
		godoConfig.inited = true
		return true, nil
	}

	modPath := filepath.Join(root, "go.mod")
	if found, err := regularFileExists(modPath); err != nil {
		return false, err
	} else if found {
		if err := loadFromGoMod(modPath); err != nil {
			return false, err
		}
		projectRoot = filepath.Clean(root)
		godoConfig.inited = true
		return true, nil
	}
	return false, nil
}

func GetProjectName() (string, error) {
	cfg, _, err := getConfigState()
	if err != nil {
		return "", err
	}
	if cfg.ProjectName == "" {
		return "", errors.New("project name is empty in godoconfig.json or go.mod")
	}
	return cfg.ProjectName, nil
}

func GetDefaultCmd() (string, error) {
	cfg, _, err := getConfigState()
	if err != nil {
		return "", err
	}
	return effectiveDefaultCmd(cfg), nil
}

func GetDefaultGOOS() (string, error) {
	cfg, _, err := getConfigState()
	if err != nil {
		return "", err
	}
	return cfg.DefaultGOOS, nil
}

func GetDefaultGOARCH() (string, error) {
	cfg, _, err := getConfigState()
	if err != nil {
		return "", err
	}
	return cfg.DefaultGOARCH, nil
}

func GetCmdType(cmdName string) (string, error) {
	if err := ValidateCmdName(cmdName); err != nil {
		return "", err
	}
	cfg, _, err := getConfigState()
	if err != nil {
		return "", err
	}
	cmdType := cfg.CmdTypes[cmdName]
	if cmdType == "" {
		return CmdTypeAPI, nil
	}
	return NormalizeCmdType(cmdType)
}

func RequireCmdType(cmdName, requiredType string) error {
	cmdType, err := GetCmdType(cmdName)
	if err != nil {
		return err
	}
	requiredType, err = NormalizeCmdType(requiredType)
	if err != nil {
		return err
	}
	if cmdType != requiredType {
		return fmt.Errorf("command %q has type %q; this operation requires %q", cmdName, cmdType, requiredType)
	}
	return nil
}

func SetCmdType(cmdName, cmdType string) error {
	if err := ValidateCmdName(cmdName); err != nil {
		return err
	}
	cmdType, err := NormalizeCmdType(cmdType)
	if err != nil {
		return err
	}

	mu.Lock()
	defer mu.Unlock()
	if !godoConfig.inited {
		if err := initializeConfigLocked(); err != nil {
			return err
		}
	}

	updated := godoConfig
	updated.CmdTypes = make(map[string]string, len(godoConfig.CmdTypes)+1)
	for name, existingType := range godoConfig.CmdTypes {
		updated.CmdTypes[name] = existingType
	}
	updated.CmdTypes[cmdName] = cmdType
	if err := writeConfigFile(filepath.Join(projectRoot, "godoconfig.json"), updated); err != nil {
		return err
	}
	godoConfig = updated
	return nil
}

func SetConfigValue(key, value string) (string, error) {
	key = strings.ToLower(strings.TrimSpace(key))

	mu.Lock()
	defer mu.Unlock()
	if !godoConfig.inited {
		if err := initializeConfigLocked(); err != nil {
			return "", err
		}
	}

	updated := godoConfig
	switch key {
	case ConfigKeyDefaultCmd:
		if err := ValidateCmdName(value); err != nil {
			return "", err
		}
		cmdPath := filepath.Join(projectRoot, "cmd", value)
		info, err := os.Stat(cmdPath)
		if err != nil {
			return "", fmt.Errorf("inspect command directory %s: %w", cmdPath, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("command path is not a directory: %s", cmdPath)
		}
		updated.DefaultCmd = value
	case ConfigKeyDefaultGOOS:
		value = strings.ToLower(strings.TrimSpace(value))
		if err := ValidateBuildTarget(value, updated.DefaultGOARCH); err != nil {
			return "", err
		}
		updated.DefaultGOOS = value
	case ConfigKeyDefaultGOARCH:
		value = strings.ToLower(strings.TrimSpace(value))
		if err := ValidateBuildTarget(updated.DefaultGOOS, value); err != nil {
			return "", err
		}
		updated.DefaultGOARCH = value
	default:
		return "", fmt.Errorf("config key %q cannot be modified; allowed keys: %s, %s, %s", key, ConfigKeyDefaultCmd, ConfigKeyDefaultGOOS, ConfigKeyDefaultGOARCH)
	}

	if err := writeConfigFile(filepath.Join(projectRoot, "godoconfig.json"), updated); err != nil {
		return "", err
	}
	godoConfig = updated
	return value, nil
}

func SetBuildTarget(goos, goarch string) (string, string, error) {
	goos = strings.ToLower(strings.TrimSpace(goos))
	goarch = strings.ToLower(strings.TrimSpace(goarch))
	if err := ValidateBuildTarget(goos, goarch); err != nil {
		return "", "", err
	}

	mu.Lock()
	defer mu.Unlock()
	if !godoConfig.inited {
		if err := initializeConfigLocked(); err != nil {
			return "", "", err
		}
	}

	updated := godoConfig
	updated.DefaultGOOS = goos
	updated.DefaultGOARCH = goarch
	if err := writeConfigFile(filepath.Join(projectRoot, "godoconfig.json"), updated); err != nil {
		return "", "", err
	}
	godoConfig = updated
	return goos, goarch, nil
}

func NormalizeCmdType(cmdType string) (string, error) {
	cmdType = strings.ToLower(strings.TrimSpace(cmdType))
	switch cmdType {
	case CmdTypeAPI, CmdTypeWorker:
		return cmdType, nil
	default:
		return "", fmt.Errorf("unsupported command type %q; expected api or worker", cmdType)
	}
}

func GetDefaultCmdCmd() (string, error) {
	cfg, root, err := getConfigState()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "cmd", effectiveDefaultCmd(cfg)), nil
}

func GetDefaultCmdInternal() (string, error) {
	cfg, root, err := getConfigState()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "internal", effectiveDefaultCmd(cfg)), nil
}

// GetAbsPath resolves path relative to the discovered project root.
func GetAbsPath(path string) (string, error) {
	return resolvePath(path)
}

func GetProjectRoot() (string, error) {
	_, root, err := getConfigState()
	if err != nil {
		return "", err
	}
	if root == "" {
		return "", errors.New("project root is unknown")
	}
	return root, nil
}

func loadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var loaded GodoConfig
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if strings.TrimSpace(loaded.ProjectName) == "" {
		return fmt.Errorf("project_name is empty in %s", path)
	}
	ensureDefaults(&loaded)
	godoConfig = loaded
	return nil
}

func loadFromGoMod(modPath string) error {
	file, err := os.Open(modPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", modPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) >= 2 && parts[0] == "module" {
			loaded := GodoConfig{ProjectName: parts[1]}
			ensureDefaults(&loaded)
			godoConfig = loaded
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan %s: %w", modPath, err)
	}
	return fmt.Errorf("module directive not found in %s", modPath)
}

func ensureDefaults(cfg *GodoConfig) {
	if cfg.DefaultGOOS == "" {
		cfg.DefaultGOOS = "linux"
	}
	if cfg.DefaultGOARCH == "" {
		cfg.DefaultGOARCH = "amd64"
	}
	if cfg.CmdTypes == nil {
		cfg.CmdTypes = make(map[string]string)
	}
}

func effectiveDefaultCmd(cfg GodoConfig) string {
	if cfg.DefaultCmd == "" {
		return "default-api"
	}
	return cfg.DefaultCmd
}

func regularFileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect %s: %w", path, err)
	}
	if info.IsDir() {
		return false, fmt.Errorf("expected a file but found a directory: %s", path)
	}
	return true, nil
}

func writeConfigFile(path string, cfg GodoConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	data = append(data, '\n')

	temp, err := os.CreateTemp(filepath.Dir(path), ".godoconfig-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary config file: %w", err)
	}
	tempName := temp.Name()
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(tempName)
	}
	if _, err := temp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write temporary config file: %w", err)
	}
	if err := temp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temporary config file: %w", err)
	}
	if err := os.Chmod(tempName, 0o644); err != nil {
		cleanup()
		return fmt.Errorf("set config permissions: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		cleanup()
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}

func resolvePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	_, root, err := getConfigState()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(root, path)), nil
}
