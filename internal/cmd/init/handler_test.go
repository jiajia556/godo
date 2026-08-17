package init

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateProjectCreatesScaffold(t *testing.T) {
	changeWorkingDirectory(t, t.TempDir())

	if err := generateProject("example.com/myapp"); err != nil {
		t.Fatalf("generateProject() error = %v", err)
	}

	goModPath := filepath.Join("example.com", "myapp", "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("read generated go.mod: %v", err)
	}
	if !strings.Contains(string(content), "module example.com/myapp") {
		t.Fatalf("generated go.mod does not contain the requested module: %s", content)
	}
}

func TestGenerateProjectRejectsExistingTarget(t *testing.T) {
	changeWorkingDirectory(t, t.TempDir())

	if err := os.Mkdir("existing", 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join("existing", "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := generateProject("existing"); err == nil {
		t.Fatal("generateProject() succeeded for an existing target")
	}
	if content, err := os.ReadFile(marker); err != nil || string(content) != "keep" {
		t.Fatalf("existing target was modified: content=%q err=%v", content, err)
	}
}

func TestGenerateProjectRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "work")
	if err := os.Mkdir(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	changeWorkingDirectory(t, workDir)

	if err := generateProject("../outside"); err == nil {
		t.Fatal("generateProject() accepted a path outside the current directory")
	}
	if _, err := os.Stat(filepath.Join(root, "outside")); !os.IsNotExist(err) {
		t.Fatalf("outside path was created: %v", err)
	}
}

func TestGenerateProjectRejectsUncleanPath(t *testing.T) {
	changeWorkingDirectory(t, t.TempDir())

	if err := generateProject("parent/../project"); err == nil {
		t.Fatal("generateProject() accepted an unclean project path")
	}
}

func changeWorkingDirectory(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}
