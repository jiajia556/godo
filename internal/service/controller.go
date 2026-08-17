package service

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"

	"github.com/jiajia556/godo/internal/utils"
)

const CONTROLLER_ACTION_TMPL = `
// @http_method %s
// @middleware
func (ctrl *%s) %s(c *gin.Context) {
	//TODO: edit
}
`

type method struct {
	Name       string
	HTTPMethod string
}

func GetControllerPathAndNameByRoute(cmdName, controllerRoute string) (path string, name string, err error) {
	if err := ValidateCmdName(cmdName); err != nil {
		return "", "", err
	}
	segments, err := validateControllerRoute(controllerRoute)
	if err != nil {
		return "", "", err
	}

	apiRoot, err := GetAbsPath(filepath.Join("internal", cmdName, "transport", "http", "api"))
	if err != nil {
		return "", "", err
	}
	component := segments[len(segments)-1]
	parts := []string{apiRoot}
	parts = append(parts, segments[:len(segments)-1]...)
	parts = append(parts, "controller", component+".go")
	controllerPath := filepath.Clean(filepath.Join(parts...))
	relative, err := filepath.Rel(apiRoot, controllerPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("controller route escapes API root: %q", controllerRoute)
	}
	return controllerPath, controllerNameFromSegment(component) + "Controller", nil
}

func ValidateControllerName(s string) error {
	if !token.IsIdentifier(s) {
		return fmt.Errorf("invalid controller name %q", s)
	}
	return nil
}

func validateControllerRoute(route string) ([]string, error) {
	if route == "" || strings.TrimSpace(route) != route {
		return nil, fmt.Errorf("controller route must not be empty or contain leading/trailing whitespace")
	}
	if strings.Contains(route, "\\") || pathpkg.Clean(route) != route {
		return nil, fmt.Errorf("controller route must be a clean forward-slash path: %q", route)
	}
	segments := strings.Split(route, "/")
	for _, segment := range segments {
		if segment == "" || !token.IsIdentifier(controllerNameFromSegment(segment)) {
			return nil, fmt.Errorf("invalid controller route segment %q", segment)
		}
		for _, word := range strings.Split(segment, "_") {
			if word == "" {
				return nil, fmt.Errorf("invalid controller route segment %q", segment)
			}
		}
	}
	return segments, nil
}

func controllerNameFromSegment(segment string) string {
	words := strings.Split(segment, "_")
	for index, word := range words {
		words[index] = utils.CapitalizeFirstLetter(word)
	}
	return strings.Join(words, "")
}

func WriteActions(controllerFilePath, controllerStructName string, actions []string) (err error) {
	actionList, err := makeActions(actions)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(controllerFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}()
	for _, v := range actionList {
		actExists, err := ControllerHasMethod(controllerFilePath, controllerStructName, v.Name)
		if err != nil {
			return err
		}
		if actExists {
			fmt.Println("action method already exists: ", v.Name)
			continue
		}
		methodStr := fmt.Sprintf(CONTROLLER_ACTION_TMPL,
			v.HTTPMethod,
			controllerStructName,
			v.Name,
		)
		_, err = file.WriteString(methodStr)
		if err != nil {
			return err
		}
	}
	return nil
}

// ControllerHasMethod reports whether controllerFilePath already defines a method
// with name methodName on receiver type controllerStructName.
// It matches both value receiver (ctrl T) and pointer receiver (ctrl *T).
func ControllerHasMethod(controllerFilePath, controllerStructName, methodName string) (bool, error) {
	if controllerFilePath == "" {
		return false, fmt.Errorf("controllerFilePath is empty")
	}
	if controllerStructName == "" {
		return false, fmt.Errorf("controllerStructName is empty")
	}
	if methodName == "" {
		return false, fmt.Errorf("methodName is empty")
	}

	fset := token.NewFileSet()
	// ParseComments not required for this check, but harmless.
	file, err := parser.ParseFile(fset, controllerFilePath, nil, parser.ParseComments)
	if err != nil {
		return false, fmt.Errorf("parse controller file %s: %w", controllerFilePath, err)
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name == nil {
			continue
		}
		if fn.Name.Name != methodName {
			continue
		}
		if receiverTypeName(fn.Recv) == controllerStructName {
			return true, nil
		}
	}

	return false, nil
}

// receiverTypeName extracts receiver base type name from receiver field list.
// Examples:
//
//	(ctrl T)    -> "T"
//	(ctrl *T)   -> "T"
//
// otherwise -> ""
func receiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}

	// A method receiver list should have exactly 1 entry, but we just take the first.
	t := recv.List[0].Type
	switch x := t.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.StarExpr:
		if id, ok := x.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

func makeActions(actions []string) (res []method, err error) {
	length := len(actions)
	if length == 0 {
		return
	}

	res = make([]method, length)
	for index, action := range actions {
		parts := strings.Split(action, ":")
		if len(parts) > 2 {
			return nil, fmt.Errorf("invalid action format %q; expected Name[:HTTPMethod]", action)
		}
		for _, word := range strings.Split(parts[0], "_") {
			if word == "" {
				return nil, fmt.Errorf("invalid action name %q", parts[0])
			}
		}
		name := controllerNameFromSegment(parts[0])
		if !token.IsIdentifier(name) {
			return nil, fmt.Errorf("invalid action name %q", parts[0])
		}
		res[index] = method{Name: name, HTTPMethod: "POST"}
		if len(parts) == 2 {
			httpMethod := strings.ToUpper(parts[1])
			switch httpMethod {
			case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "ALL":
				res[index].HTTPMethod = httpMethod
			default:
				return nil, fmt.Errorf("invalid HTTP method %q for action %q", parts[1], parts[0])
			}
		}
	}
	return res, nil
}
