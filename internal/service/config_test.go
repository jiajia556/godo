package service

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type managerTestConfig struct {
	Name    string `json:"name" yaml:"name"`
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Ports   []int  `json:"ports" yaml:"ports"`
}

func TestConfigManagerJSONAndYAMLRoundTrip(t *testing.T) {
	want := managerTestConfig{Name: "demo", Enabled: true, Ports: []int{80, 443}}
	for _, extension := range []string{".json", ".yaml", ".yml"} {
		t.Run(extension, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "nested", "config"+extension)
			manager := NewManager[managerTestConfig]()
			if got := manager.Get(); !reflect.DeepEqual(got, managerTestConfig{}) {
				t.Fatalf("new manager data = %+v", got)
			}
			manager.Set(want)
			if err := manager.Save(path); err != nil {
				t.Fatalf("Save() error = %v", err)
			}
			if manager.Path() != path {
				t.Fatalf("Path() = %q, want %q", manager.Path(), path)
			}

			loaded := NewManager[managerTestConfig]()
			if err := loaded.Load(path); err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got := loaded.Get(); !reflect.DeepEqual(got, want) {
				t.Fatalf("loaded data = %+v, want %+v", got, want)
			}
		})
	}
}

func TestConfigManagerReportsInvalidInput(t *testing.T) {
	root := t.TempDir()
	manager := NewManager[managerTestConfig]()

	if err := manager.Load(filepath.Join(root, "missing.json")); err == nil || !strings.Contains(err.Error(), "read config") {
		t.Fatalf("Load(missing) error = %v", err)
	}
	invalidJSON := filepath.Join(root, "invalid.json")
	if err := os.WriteFile(invalidJSON, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := manager.Load(invalidJSON); err == nil || !strings.Contains(err.Error(), "parse json") {
		t.Fatalf("Load(invalid JSON) error = %v", err)
	}
	invalidYAML := filepath.Join(root, "invalid.yaml")
	if err := os.WriteFile(invalidYAML, []byte("name: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := manager.Load(invalidYAML); err == nil || !strings.Contains(err.Error(), "parse yaml") {
		t.Fatalf("Load(invalid YAML) error = %v", err)
	}
	unsupported := filepath.Join(root, "config.toml")
	if err := os.WriteFile(unsupported, []byte("name='demo'"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := manager.Load(unsupported); err == nil || !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("Load(unsupported) error = %v", err)
	}
	if err := manager.Save(unsupported); err == nil || !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("Save(unsupported) error = %v", err)
	}
}

func TestConfigManagerReportsMarshalError(t *testing.T) {
	manager := NewManager[map[string]any]()
	manager.Set(map[string]any{"unsupported": func() {}})
	if err := manager.Save(filepath.Join(t.TempDir(), "config.json")); err == nil || !strings.Contains(err.Error(), "marshal") {
		t.Fatalf("Save() error = %v", err)
	}
}

func TestGlobalConfigLoadAndGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := "mysql:\n  host: localhost\n  user: root\n  db_name: app\n  port: 3306\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadConfig(path); err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	got := GetConfig()
	if got.Mysql.Host != "localhost" || got.Mysql.Port != 3306 || got.Mysql.DBName != "app" {
		t.Fatalf("GetConfig() = %+v", got)
	}
}
