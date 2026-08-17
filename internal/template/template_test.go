package template

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTemplateWriterCreatesAndReplacesFile(t *testing.T) {
	root := t.TempDir()
	writer := NewTemplateWriter()
	writer.BaseDir = root
	writer.FilePerm = 0o600

	path := filepath.Join("nested", "message.txt")
	if err := writer.CreateFile("hello {{.}}", "world", path); err != nil {
		t.Fatalf("CreateFile() error = %v", err)
	}
	fullPath := filepath.Join(root, path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello world" {
		t.Fatalf("content = %q", content)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("mode = %o, want 600", info.Mode().Perm())
		}
	}

	if err := writer.CreateFile("replaced", nil, path); err != nil {
		t.Fatalf("replace CreateFile() error = %v", err)
	}
	content, err = os.ReadFile(fullPath)
	if err != nil || string(content) != "replaced" {
		t.Fatalf("replaced content = %q, err = %v", content, err)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(fullPath), writer.TempPattern))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v, err = %v", matches, err)
	}
}

func TestCreateFileUsesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output.txt")
	if err := CreateFile("{{.ProjectName}}/{{.CmdName}}", ProjectNameData{ProjectName: "demo", CmdName: "api"}, path); err != nil {
		t.Fatalf("CreateFile() error = %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "demo/api" {
		t.Fatalf("content = %q, err = %v", content, err)
	}
}

func TestTemplateWriterReportsParseExecuteAndDirectoryErrors(t *testing.T) {
	root := t.TempDir()
	writer := NewTemplateWriter()
	writer.BaseDir = root

	if err := writer.CreateFile("{{", nil, "parse.txt"); err == nil || !strings.Contains(err.Error(), "parse template") {
		t.Fatalf("parse error = %v", err)
	}
	failing := func() (string, error) { return "", errors.New("render failed") }
	if err := writer.CreateFile("{{call .}}", failing, "execute.txt"); err == nil || !strings.Contains(err.Error(), "execute template") {
		t.Fatalf("execute error = %v", err)
	}
	if matches, _ := filepath.Glob(filepath.Join(root, writer.TempPattern)); len(matches) != 0 {
		t.Fatalf("temporary files left after execute failure: %v", matches)
	}

	blockingPath := filepath.Join(root, "blocking")
	if err := os.WriteFile(blockingPath, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writer.CreateFile("content", nil, filepath.Join("blocking", "output.txt")); err == nil || !strings.Contains(err.Error(), "create dir") {
		t.Fatalf("directory error = %v", err)
	}
}
