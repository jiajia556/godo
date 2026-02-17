package template

import (
	"fmt"
	"os"
	"path/filepath"
	stdtmpl "text/template"
)

// RouterTmplData holds data used to render the router template.
type RouterTmplData struct {
	MiddlewareImportPath  string
	ControllersImportPath string
	ApiRootDirName        string
	HTTPMethodTags        string
	MiddlewareTags        string
	RegisterControllers   string
}

type ProjectNameData struct {
	ProjectName string
	CmdName     string
}

type ControllerStructNameData struct {
	ControllerStructName string
}

type MiddlewareNameData struct {
	MiddlewareName string
}

type ModelData struct {
	ModelPkg        string
	ProjectName     string
	ModelStruct     string
	ModelStructName string
}

type TemplateWriter struct {
	BaseDir     string
	FilePerm    os.FileMode
	DirPerm     os.FileMode
	TempPattern string
}

func NewTemplateWriter() *TemplateWriter {
	return &TemplateWriter{
		FilePerm:    0o644,
		DirPerm:     0o755,
		TempPattern: ".tmp-tmpl-*",
	}
}

func (w *TemplateWriter) CreateFile(tmplContent string, data any, path string) error {
	// Parse template (do not panic on error)
	tmpl, err := stdtmpl.New(filepath.Base(path)).Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	// Resolve base dir if configured
	if w.BaseDir != "" && !filepath.IsAbs(path) {
		path = filepath.Join(w.BaseDir, path)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "" {
		if err := os.MkdirAll(dir, w.DirPerm); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	// Create temporary file in same directory to allow atomic rename
	pattern := w.TempPattern
	if pattern == "" {
		pattern = ".tmp-tmpl-*"
	}
	tmpFile, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}

	// Ensure cleanup of temp file on error
	tmpName := tmpFile.Name()
	cleanup := func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
	}
	// Execute template into temp file
	if err := tmpl.Execute(tmpFile, data); err != nil {
		cleanup()
		return fmt.Errorf("execute template to %s: %w", tmpName, err)
	}

	// Close before rename
	if err := tmpFile.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file %s: %w", tmpName, err)
	}

	// Set file permission to 0644
	perm := w.FilePerm
	if perm == 0 {
		perm = 0o644
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("chmod temp file %s: %w", tmpName, err)
	}

	// Atomically replace the target file
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename temp file %s -> %s: %w", tmpName, path, err)
	}

	return nil
}

// CreateFile renders the provided template content with data and writes it to path.
// The write is performed atomically by writing to a temp file in the same directory
// and then renaming it into place. Returns any parse/execute/io error instead of panicking.
func CreateFile(tmplContent string, data any, path string) error {
	return NewTemplateWriter().CreateFile(tmplContent, data, path)
}
