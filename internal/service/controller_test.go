package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteActionsAppendsActionOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "user.go")
	initial := "package controller\n\ntype UserController struct{}\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteActions(path, "UserController", []string{"List:GET"}); err != nil {
		t.Fatalf("WriteActions() error = %v", err)
	}
	if err := WriteActions(path, "UserController", []string{"List:GET"}); err != nil {
		t.Fatalf("second WriteActions() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	generated := string(content)
	if strings.Count(generated, "func (ctrl *UserController) List") != 1 {
		t.Fatalf("action was not generated exactly once:\n%s", generated)
	}
	if !strings.Contains(generated, "// @http_method GET") {
		t.Fatalf("HTTP method annotation is missing:\n%s", generated)
	}
}

func TestWriteActionsSupportsDELETE(t *testing.T) {
	path := filepath.Join(t.TempDir(), "user.go")
	if err := os.WriteFile(path, []byte("package controller\n\ntype UserController struct{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteActions(path, "UserController", []string{"Remove:DELETE"}); err != nil {
		t.Fatalf("WriteActions() error = %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "// @http_method DELETE") {
		t.Fatalf("DELETE annotation is missing:\n%s", content)
	}
}

func TestWriteActionsRejectsUnsupportedHTTPMethod(t *testing.T) {
	path := filepath.Join(t.TempDir(), "user.go")
	if err := os.WriteFile(path, []byte("package controller\n\ntype UserController struct{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteActions(path, "UserController", []string{"List:TRACE"}); err == nil {
		t.Fatal("WriteActions() accepted an unsupported HTTP method")
	}
}

func TestControllerRouteValidation(t *testing.T) {
	valid := []string{"user", "admin/user_profile", "权限/用户"}
	for _, route := range valid {
		if _, err := validateControllerRoute(route); err != nil {
			t.Errorf("validateControllerRoute(%q) error = %v", route, err)
		}
	}

	invalid := []string{"", "/user", "user/", "user//profile", "../user", "admin/../../user", `admin\user`, "123user", "user-name"}
	for _, route := range invalid {
		if _, err := validateControllerRoute(route); err == nil {
			t.Errorf("validateControllerRoute(%q) succeeded", route)
		}
	}
}

func TestGetControllerPathStaysWithinAPIRoot(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/project\n")
	prepareConfigTest(t, root, "")

	controllerPath, name, err := GetControllerPathAndNameByRoute("default-api", "admin/user_profile")
	if err != nil {
		t.Fatalf("GetControllerPathAndNameByRoute() error = %v", err)
	}
	want := filepath.Join(root, "internal", "default-api", "transport", "http", "api", "admin", "controller", "user_profile.go")
	if controllerPath != want || name != "UserProfileController" {
		t.Fatalf("controller = %q, %q; want %q, UserProfileController", controllerPath, name, want)
	}
}

func TestMakeActionsValidatesNamesAndFormat(t *testing.T) {
	actions, err := makeActions([]string{"get_list", "Remove:delete"})
	if err != nil {
		t.Fatalf("makeActions() error = %v", err)
	}
	if actions[0].Name != "GetList" || actions[0].HTTPMethod != "POST" {
		t.Fatalf("default action = %+v", actions[0])
	}
	if actions[1].Name != "Remove" || actions[1].HTTPMethod != "DELETE" {
		t.Fatalf("DELETE action = %+v", actions[1])
	}

	for _, action := range []string{"", "123List", "Bad-Name", "bad__name", "List:GET:extra", "List:TRACE"} {
		if _, err := makeActions([]string{action}); err == nil {
			t.Errorf("makeActions(%q) succeeded", action)
		}
	}
}
