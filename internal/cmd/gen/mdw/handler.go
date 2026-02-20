package mdw

import (
	"github.com/jiajia556/godo/internal/service"
	"github.com/jiajia556/godo/internal/template"
	"github.com/jiajia556/godo/internal/utils"
	"github.com/jiajia556/godo/templates"
)

func genMiddleware(middlewares []string) {
	content, err := templates.TemplateFS.ReadFile("default/internal/common/transport/http/middleware/middleware.go.templ")
	if err != nil {
		utils.OutputFatal(err)
	}
	for _, middleware := range middlewares {
		middlewareName := utils.CapitalizeFirstLetter(middleware)

		filePath := "internal/common/transport/http/middleware/" + middlewareName + ".go"
		filePath, err = service.GetAbsPath(filePath)
		if err != nil {
			utils.OutputFatal(err)
		}
		if utils.IsFileExists(filePath) {
			utils.OutputErrorf("%s already exists", middleware)
			continue
		}
		err := template.CreateFile(string(content), template.MiddlewareNameData{middlewareName}, filePath)
		if err != nil {
			utils.OutputFatal(err)
		}
	}
}
