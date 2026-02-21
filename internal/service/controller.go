package service

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
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
	// Validate route format constraints
	if strings.HasPrefix(controllerRoute, "/") || strings.HasSuffix(controllerRoute, "/") {
		err = fmt.Errorf("controllerRoute must not start or end with '/'")
		return "", "", err
	}

	// Locate the last directory separator
	lastSlashPos := strings.LastIndex(controllerRoute, "/")
	if lastSlashPos == -1 {
		// Handle simple case with no subdirectories
		path, err = GetAbsPath(fmt.Sprintf("internal/%s/transport/http/api/controller/%s.go", cmdName, controllerRoute))
		if err != nil {
			return "", "", err
		}
		return path, utils.CapitalizeFirstLetter(controllerRoute) + "Controller", nil
	}

	// Split route into directory and component name
	directory := controllerRoute[:lastSlashPos]
	component := controllerRoute[lastSlashPos+1:]

	// Construct controller file path
	path = fmt.Sprintf("internal/%s/transport/http/api/%s/controller/%s.go", cmdName, directory, component)
	path, err = GetAbsPath(path)
	if err != nil {
		return "", "", err
	}
	return path, utils.CapitalizeFirstLetter(component) + "Controller", nil
}

func ValidateControllerName(s string) error {
	if strings.Contains(s, " ") {
		return errors.New("controller name can not contain spaces")
	}
	if strings.Contains(s, "_") {
		return errors.New("controller name can not contain _")
	}
	if strings.Contains(s, "-") {
		return errors.New("controller name can not contain -")
	}
	return nil
}

func WriteActions(controllerFilePath, controllerStructName string, actions []string) error {
	actionList, err := makeActions(actions)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(controllerFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
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
	for k, mtd := range actions {
		mtdDetail := strings.Split(mtd, ":")
		for i, v := range mtdDetail {
			if i == 0 {
				res[k].Name = utils.CapitalizeFirstLetter(v)
			} else {
				switch strings.ToLower(v) {
				case "post":
					res[k].HTTPMethod = "POST"
				case "get":
					res[k].HTTPMethod = "GET"
				default:
					err = fmt.Errorf("invalid method: %s", v)
					return
				}
			}
			if res[k].HTTPMethod == "" {
				res[k].HTTPMethod = "POST"
			}
		}
	}
	return
}
