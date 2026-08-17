package mdw

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/jiajia556/godo/internal/service"
	"github.com/jiajia556/godo/internal/template"
	"github.com/jiajia556/godo/internal/utils"
	"github.com/jiajia556/godo/templates"
)

var formatGoFiles = utils.FormatGoFiles

func genMiddleware(middlewares []string) error {
	for _, middleware := range middlewares {
		if err := validateMiddlewareName(middleware); err != nil {
			return err
		}
	}

	content, err := templates.TemplateFS.ReadFile("default/internal/common/transport/http/middleware/middleware.go.templ")
	if err != nil {
		return fmt.Errorf("read middleware template: %w", err)
	}
	generatedFiles := make([]string, 0, len(middlewares))
	for _, middleware := range middlewares {
		middlewareName := utils.CapitalizeFirstLetter(middleware)
		fileName := strings.ToLower(middlewareName)

		filePath := "internal/common/transport/http/middleware/" + fileName + ".go"
		filePath, err = service.GetAbsPath(filePath)
		if err != nil {
			return fmt.Errorf("resolve middleware path for %q: %w", middleware, err)
		}
		if utils.IsFileExists(filePath) {
			utils.OutputErrorf("%s already exists", middleware)
			continue
		}
		err := template.CreateFile(string(content), template.MiddlewareNameData{MiddlewareName: middlewareName}, filePath)
		if err != nil {
			return fmt.Errorf("write middleware %q: %w", middleware, err)
		}
		generatedFiles = append(generatedFiles, filePath)
	}
	if err := formatGoFiles(generatedFiles...); err != nil {
		return fmt.Errorf("format generated middleware: %w", err)
	}
	return nil
}

func validateMiddlewareName(name string) error {
	if !token.IsIdentifier(utils.CapitalizeFirstLetter(name)) {
		return fmt.Errorf("invalid middleware name %q", name)
	}
	return nil
}
