package rt

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jiajia556/godo/internal/service"
	"github.com/jiajia556/godo/internal/template"
	"github.com/jiajia556/godo/internal/utils"
	"github.com/jiajia556/godo/templates"
)

const (
	generatedFileName    = "router.go"    // Output filename for generated router
	controllerSuffix     = "Controller"   // Suffix for controller type names
	controllerDirName    = "controller"   // Standard directory name for controllers
	httpMethodAnnotation = "@http_method" // Annotation prefix for HTTP methods
	middlewareAnnotation = "@middleware"  // Annotation prefix for middlewares
)

var supportedHTTPMethods = map[string]struct{}{
	"GET": {}, "POST": {}, "PUT": {}, "PATCH": {}, "DELETE": {},
	"HEAD": {}, "OPTIONS": {}, "ALL": {},
}

var formatGoFiles = utils.FormatGoFiles

// routeGenerator maintains state during route generation process
type routeGenerator struct {
	imports           []string          // Import paths for controller packages
	initRegistrations []string          // Controller registration statements
	pkgAliases        map[string]string // Package import aliases
	httpMethods       map[string]string // HTTP method mappings
	middlewares       map[string]string // Middleware configurations
	projectName       string            // Current project module name
	projectRoot       string            // Current project root directory
}

func GenRouter(cmdName string) error {
	var err error
	if cmdName == "" {
		cmdName, err = service.GetDefaultCmd()
		if err != nil {
			return fmt.Errorf("get default command: %w", err)
		}
	}
	if err := service.ValidateCmdName(cmdName); err != nil {
		return fmt.Errorf("validate command name: %w", err)
	}
	if err := service.RequireCmdType(cmdName, service.CmdTypeAPI); err != nil {
		return err
	}

	rg := &routeGenerator{
		pkgAliases:  make(map[string]string),
		httpMethods: make(map[string]string),
		middlewares: make(map[string]string),
	}
	if rg.projectName, err = service.GetProjectName(); err != nil {
		return fmt.Errorf("get project name: %w", err)
	}
	if rg.projectRoot, err = service.GetProjectRoot(); err != nil {
		return fmt.Errorf("get project root: %w", err)
	}
	rootPath, err := service.GetAbsPath(fmt.Sprintf("internal/%s/transport/http/api", cmdName))
	if err != nil {
		return fmt.Errorf("resolve controller root: %w", err)
	}

	tmplData, err := rg.generateTemplateData(rootPath)
	if err != nil {
		return fmt.Errorf("generate router template data: %w", err)
	}

	routePath, err := service.GetAbsPath(fmt.Sprintf("internal/%s/transport/http/router", cmdName))
	if err != nil {
		return fmt.Errorf("resolve router output directory: %w", err)
	}
	outputPath := filepath.Join(routePath, generatedFileName)
	content, err := templates.TemplateFS.ReadFile("default/internal/default-api/transport/http/router/router.go.templ")
	if err != nil {
		return fmt.Errorf("read router template: %w", err)
	}
	err = template.CreateFile(string(content), tmplData, outputPath)
	if err != nil {
		return fmt.Errorf("write router file: %w", err)
	}
	if err = formatGoFiles(outputPath); err != nil {
		return fmt.Errorf("format router file: %w", err)
	}
	return nil
}

// generateTemplateData collects and prepares data for template generation
func (rg *routeGenerator) generateTemplateData(root string) (template.RouterTmplData, error) {
	if err := rg.analyzeProjectStructure(root); err != nil {
		return template.RouterTmplData{}, fmt.Errorf("project analysis failed: %w", err)
	}

	ProjectName, err := service.GetProjectName()
	if err != nil {
		return template.RouterTmplData{}, fmt.Errorf("failed to get project name: %w", err)
	}

	return template.RouterTmplData{
		HTTPMethodTags:        rg.formatHTTPMethods(),
		MiddlewareTags:        rg.formatMiddlewares(),
		RegisterControllers:   strings.Join(rg.initRegistrations, ""),
		MiddlewareImportPath:  rg.middlewareImport(),
		ControllersImportPath: strings.Join(rg.imports, "\n\t"),
		ProjectName:           ProjectName,
	}, nil
}

// analyzeProjectStructure walks through project directories to find controllers
func (rg *routeGenerator) analyzeProjectStructure(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("access %s: %w", path, err)
		}

		if d.IsDir() && d.Name() == controllerDirName {
			if err := rg.processControllerPackage(path); err != nil {
				return fmt.Errorf("controller processing failed: %w", err)
			}
			return filepath.SkipDir
		}
		return nil
	})
}

// processControllerPackage processes all Go files in a controller directory
func (rg *routeGenerator) processControllerPackage(dirPath string) error {
	return filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
			return err
		}
		return rg.analyzeControllerFile(path)
	})
}

// analyzeControllerFile parses a single Go file for controller definitions
func (rg *routeGenerator) analyzeControllerFile(filePath string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("file parsing failed: %w", err)
	}

	pkgPath := constructImportPath(rg.projectName, rg.projectRoot, filePath)
	alias, exists := rg.pkgAliases[pkgPath]

	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || !strings.HasSuffix(typeSpec.Name.Name, controllerSuffix) {
				continue
			}

			controllerName := typeSpec.Name.Name
			if !exists {
				alias = fmt.Sprintf("controller%d", len(rg.imports))
				// ensure alias uniqueness in rare case
				for _, a := range rg.imports {
					if strings.Contains(a, "\""+pkgPath+"\"") && rg.pkgAliases[pkgPath] == alias {
						alias = fmt.Sprintf("controller%d", len(rg.imports)+1)
					}
				}
				rg.pkgAliases[pkgPath] = alias
				rg.imports = append(rg.imports, fmt.Sprintf("\t%s \"%s\"", alias, pkgPath))
			}

			fullTypeName := fmt.Sprintf("%s.New%s", alias, controllerName)
			rg.initRegistrations = append(rg.initRegistrations,
				fmt.Sprintf("\n\tRegisterController(%s())", fullTypeName))

			if err := rg.extractAnnotations(node, controllerName, pkgPath+"."+controllerName); err != nil {
				return err
			}
		}
	}
	return nil
}

// extractAnnotations parses controller method annotations
func (rg *routeGenerator) extractAnnotations(node *ast.File, typeName, pkgPrefix string) error {
	var annotationErr error
	ast.Inspect(node, func(n ast.Node) bool {
		if annotationErr != nil {
			return false
		}
		fnDecl, ok := n.(*ast.FuncDecl)
		if !ok || fnDecl.Recv == nil || len(fnDecl.Recv.List) == 0 {
			return true
		}

		recvType := extractReceiverType(fnDecl.Recv.List[0].Type)
		if recvType != typeName {
			return true
		}

		annotationKey := fmt.Sprintf("%s.%s", pkgPrefix, fnDecl.Name.Name)
		annotationErr = rg.processMethodAnnotations(fnDecl, annotationKey)
		return true
	})
	return annotationErr
}

func (rg *routeGenerator) processMethodAnnotations(fnDecl *ast.FuncDecl, key string) error {
	if fnDecl.Doc == nil {
		rg.httpMethods[key] = "POST"
		return nil
	}
	for _, comment := range fnDecl.Doc.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		switch {
		case strings.HasPrefix(text, httpMethodAnnotation):
			method := strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(text, httpMethodAnnotation)))
			if method == "" {
				rg.httpMethods[key] = "POST"
				continue
			}
			if _, ok := supportedHTTPMethods[method]; !ok {
				return fmt.Errorf("unsupported HTTP method %q on %s", method, key)
			}
			rg.httpMethods[key] = method
		case strings.HasPrefix(text, middlewareAnnotation):
			names := strings.Fields(strings.TrimPrefix(text, middlewareAnnotation))
			for i, name := range names {
				name = utils.CapitalizeFirstLetter(name)
				if !token.IsIdentifier(name) {
					return fmt.Errorf("invalid middleware %q on %s", names[i], key)
				}
				names[i] = name
			}
			if len(names) > 0 {
				rg.middlewares[key] = strings.Join(names, " ")
			}
		}
	}
	return nil
}

func (rg *routeGenerator) formatHTTPMethods() string {
	var builder strings.Builder
	keys := sortedMapKeys(rg.httpMethods)
	for _, k := range keys {
		v := rg.httpMethods[k]
		builder.WriteString(fmt.Sprintf("\t\t\"%s\": \"%s\",\n", k, v))
	}
	return builder.String()
}

func (rg *routeGenerator) formatMiddlewares() string {
	var builder strings.Builder
	keys := sortedMapKeys(rg.middlewares)
	for _, k := range keys {
		v := rg.middlewares[k]
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}

		components := strings.Fields(v)

		for i := 0; i < len(components); i++ {
			components[i] = "middleware." + strings.TrimSpace(components[i])
		}

		formatted := "{" + strings.Join(components, ", ") + "}"
		builder.WriteString(fmt.Sprintf("\t\t\"%s\": %s,\n", k, formatted))
	}
	return builder.String()
}

func sortedMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (rg *routeGenerator) middlewareImport() string {
	if len(rg.middlewares) > 0 {
		return fmt.Sprintf("\t\"%s/internal/common/transport/http/middleware\"", rg.projectName)
	}
	return ""
}

// Helper functions below maintain the same logic with improved readability
func constructImportPath(projectName, projectRoot, filePath string) string {
	// Normalize to absolute slash-separated paths
	absFilePath, _ := filepath.Abs(filePath)
	absFilePath = filepath.ToSlash(absFilePath)

	absProjectRoot := projectRoot
	if absProjectRoot == "" {
		if p, err := filepath.Abs("."); err == nil {
			absProjectRoot = p
		} else {
			absProjectRoot = ""
		}
	}
	absProjectRoot = filepath.ToSlash(absProjectRoot)

	// Directory containing the file
	dir := filepath.ToSlash(filepath.Dir(absFilePath))

	// Compute relative path from project root to the file's directory
	rel := dir
	if absProjectRoot != "" {
		if r, err := filepath.Rel(absProjectRoot, dir); err == nil {
			rel = filepath.ToSlash(r)
		} else {
			// fallback: if Dir contains projectRoot as prefix, trim prefix
			if strings.HasPrefix(dir, absProjectRoot+"/") {
				rel = strings.TrimPrefix(dir, absProjectRoot+"/")
			}
		}
	}

	// Clean and trim
	rel = strings.Trim(rel, "/")

	// If relative path is empty, import is module root
	if rel == "" {
		return projectName
	}

	// Compose module import path
	return strings.TrimRight(projectName, "/") + "/" + rel
}

// extractReceiverType extracts the receiver's base type name from an AST expression.
// Examples:
//
//	(c T)    -> "T"
//	(c *T)   -> "T"
//	otherwise -> ""
func extractReceiverType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}
