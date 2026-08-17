package utils

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestFileUtilitiesRoundTrip(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source", "nested", "data.txt")
	if err := WriteFile(source, "hello"); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if !IsFileExists(source) || IsDirExists(source) || IsFileExists("") || IsDirExists("") {
		t.Fatal("file/directory existence checks returned unexpected values")
	}
	content, err := ReadFile(source)
	if err != nil || content != "hello" {
		t.Fatalf("ReadFile() = %q, %v", content, err)
	}

	copiedFile := filepath.Join(root, "single", "copy.txt")
	if err := CopyFile(source, copiedFile); err != nil {
		t.Fatalf("CopyFile() error = %v", err)
	}
	if copied, _ := ReadFile(copiedFile); copied != "hello" {
		t.Fatalf("copied file = %q", copied)
	}

	copyRoot := filepath.Join(root, "copy-tree")
	if err := WriteFile(filepath.Join(root, "source", "root.go"), "package root"); err != nil {
		t.Fatal(err)
	}
	if err := CopyDir(filepath.Join(root, "source"), copyRoot); err != nil {
		t.Fatalf("CopyDir() error = %v", err)
	}
	if !IsDirExists(copyRoot) || !IsFileExists(filepath.Join(copyRoot, "nested", "data.txt")) {
		t.Fatal("copied directory tree is incomplete")
	}

	files, err := ListFiles(copyRoot, ".go")
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}
	sort.Strings(files)
	want := []string{filepath.Join(copyRoot, "root.go")}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("ListFiles() = %v, want %v", files, want)
	}
	allFiles, err := ListFiles(copyRoot, "")
	if err != nil || len(allFiles) != 1 {
		t.Fatalf("ListFiles(all) = %v, %v", allFiles, err)
	}

	if err := RemoveDir(copyRoot); err != nil || IsDirExists(copyRoot) {
		t.Fatalf("RemoveDir() error = %v, still exists = %v", err, IsDirExists(copyRoot))
	}
}

func TestFileUtilitiesReportMissingSources(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing")
	checks := []struct {
		name string
		err  error
	}{
		{"ReadFile", func() error { _, err := ReadFile(missing); return err }()},
		{"CopyFile", CopyFile(missing, filepath.Join(root, "copy"))},
		{"CopyDir", CopyDir(missing, filepath.Join(root, "copy-dir"))},
		{"ListFiles", func() error { _, err := ListFiles(missing, ""); return err }()},
	}
	for _, check := range checks {
		if check.err == nil {
			t.Errorf("%s error = %v", check.name, check.err)
		}
	}
	if IsFileExists(missing) || IsDirExists(missing) {
		t.Fatal("missing path reported as existing")
	}
}
