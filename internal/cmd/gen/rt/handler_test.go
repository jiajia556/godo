package rt

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeProjectStructureReturnsMissingRootError(t *testing.T) {
	rg := &routeGenerator{}
	missing := filepath.Join(t.TempDir(), "missing")

	err := rg.analyzeProjectStructure(missing)
	if err == nil {
		t.Fatal("analyzeProjectStructure() succeeded for a missing root")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("error does not identify the missing root: %v", err)
	}
}

func TestRouteFormattingIsDeterministic(t *testing.T) {
	rg := &routeGenerator{
		httpMethods: map[string]string{"z.Method": "POST", "a.Method": "GET"},
		middlewares: map[string]string{"z.Method": "Auth", "a.Method": "Logging"},
	}

	httpMethods := rg.formatHTTPMethods()
	if strings.Index(httpMethods, "a.Method") > strings.Index(httpMethods, "z.Method") {
		t.Fatalf("HTTP methods are not sorted:\n%s", httpMethods)
	}
	middlewares := rg.formatMiddlewares()
	if strings.Index(middlewares, "a.Method") > strings.Index(middlewares, "z.Method") {
		t.Fatalf("middlewares are not sorted:\n%s", middlewares)
	}
}

func TestProcessMethodAnnotationsValidatesAndNormalizes(t *testing.T) {
	method := parseControllerMethod(t, `
// @http_method delete
// @middleware auth RequestLog
func (ctrl *UserController) Remove() {}
`)
	rg := &routeGenerator{httpMethods: map[string]string{}, middlewares: map[string]string{}}
	if err := rg.processMethodAnnotations(method, "users.Remove"); err != nil {
		t.Fatalf("processMethodAnnotations() error = %v", err)
	}
	if got := rg.httpMethods["users.Remove"]; got != "DELETE" {
		t.Fatalf("HTTP method = %q, want DELETE", got)
	}
	if got := rg.middlewares["users.Remove"]; got != "Auth RequestLog" {
		t.Fatalf("middlewares = %q, want normalized names", got)
	}
}

func TestProcessMethodAnnotationsRejectsInvalidValues(t *testing.T) {
	tests := []string{
		"// @http_method TRACE\n",
		"// @middleware ../auth\n",
	}
	for _, annotation := range tests {
		method := parseControllerMethod(t, annotation+"func (ctrl *UserController) Handle() {}\n")
		rg := &routeGenerator{httpMethods: map[string]string{}, middlewares: map[string]string{}}
		if err := rg.processMethodAnnotations(method, "users.Handle"); err == nil {
			t.Fatalf("processMethodAnnotations() accepted %q", annotation)
		}
	}
}

func TestGenRouterAnalyzesControllersAndRejectsWorker(t *testing.T) {
	root := t.TempDir()
	config := `{
  "project_name": "example.com/project",
  "default_cmd": "api",
  "cmd_types": {"api": "api", "worker": "worker"}
}`
	if err := os.WriteFile(filepath.Join(root, "godoconfig.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	controllerDir := filepath.Join(root, "internal", "api", "transport", "http", "api", "user", "controller")
	if err := os.MkdirAll(controllerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	controller := `package controller

type UserController struct{}

// List returns users.
// @http_method get
// @middleware auth requestLog
func (ctrl *UserController) List() {}

func (ctrl UserController) Create() {}
`
	if err := os.WriteFile(filepath.Join(controllerDir, "user.go"), []byte(controller), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOD_PROJECT_ROOT", root)
	previousFormatter := formatGoFiles
	var formatted []string
	formatGoFiles = func(paths ...string) error {
		formatted = append(formatted, paths...)
		return nil
	}
	t.Cleanup(func() { formatGoFiles = previousFormatter })

	if err := GenRouter("api"); err != nil {
		t.Fatalf("GenRouter() error = %v", err)
	}
	routerPath := filepath.Join(root, "internal", "api", "transport", "http", "router", "router.go")
	content, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`controller0 "example.com/project/internal/api/transport/http/api/user/controller"`,
		`"example.com/project/internal/common/transport/http/middleware"`,
		`"example.com/project/internal/api/transport/http/api/user/controller.UserController.List": "GET"`,
		`RegisterController(controller0.NewUserController())`,
	} {
		if !strings.Contains(string(content), expected) {
			t.Errorf("router does not contain %q:\n%s", expected, content)
		}
	}
	if len(formatted) != 1 || formatted[0] != routerPath {
		t.Fatalf("formatted paths = %v", formatted)
	}
	if err := GenRouter("worker"); err == nil || !strings.Contains(err.Error(), "requires \"api\"") {
		t.Fatalf("worker GenRouter() error = %v", err)
	}
}

func TestRouteGeneratorHelpers(t *testing.T) {
	rg := &routeGenerator{middlewares: map[string]string{}}
	if got := rg.middlewareImport(); got != "" {
		t.Fatalf("middlewareImport() = %q", got)
	}
	rg.middlewares["x"] = "Auth"
	rg.projectName = "example.com/project"
	if got := rg.middlewareImport(); !strings.Contains(got, "example.com/project/internal/common") {
		t.Fatalf("middlewareImport() = %q", got)
	}

	root := t.TempDir()
	file := filepath.Join(root, "internal", "api", "controller", "user.go")
	want := "example.com/project/internal/api/controller"
	if got := constructImportPath("example.com/project", root, file); got != want {
		t.Fatalf("constructImportPath() = %q, want %q", got, want)
	}

	fileNode, err := parser.ParseFile(token.NewFileSet(), "receiver.go", "package p\ntype T struct{}\nfunc (T) Value(){}\nfunc (*T) Pointer(){}", 0)
	if err != nil {
		t.Fatal(err)
	}
	var receivers []string
	for _, declaration := range fileNode.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Recv != nil {
			receivers = append(receivers, extractReceiverType(function.Recv.List[0].Type))
		}
	}
	if strings.Join(receivers, ",") != "T,T" {
		t.Fatalf("receiver types = %v", receivers)
	}
}

func parseControllerMethod(t *testing.T, method string) *ast.FuncDecl {
	t.Helper()
	source := "package controller\ntype UserController struct{}\n" + method
	file, err := parser.ParseFile(token.NewFileSet(), "controller.go", source, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse controller source: %v", err)
	}
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok {
			return function
		}
	}
	t.Fatal("controller method not found")
	return nil
}
