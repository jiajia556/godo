package utils

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCollectGoFiles(t *testing.T) {
	root := t.TempDir()
	goFile := filepath.Join(root, "pkg", "file.go")
	otherFile := filepath.Join(root, "pkg", "notes.txt")
	vendorFile := filepath.Join(root, "vendor", "ignored.go")
	for path, content := range map[string]string{
		goFile:     "package pkg\n",
		otherFile:  "not Go",
		vendorFile: "package ignored\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := collectGoFiles([]string{root, goFile})
	if err != nil {
		t.Fatalf("collectGoFiles() error = %v", err)
	}
	absoluteGoFile, err := filepath.Abs(goFile)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{absoluteGoFile}; !reflect.DeepEqual(files, want) {
		t.Fatalf("collectGoFiles() = %v, want %v", files, want)
	}
}

func TestCollectGoFilesReportsMissingPath(t *testing.T) {
	_, err := collectGoFiles([]string{filepath.Join(t.TempDir(), "missing")})
	if err == nil {
		t.Fatal("collectGoFiles() succeeded for a missing path")
	}
}
