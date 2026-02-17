package ctrl

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jiajia556/godo/internal/service"
	"github.com/jiajia556/godo/internal/template"
	"github.com/jiajia556/godo/internal/utils"
	"github.com/jiajia556/godo/templates"
)

func genCtrl(cmdName, controllerRoute string, actions []string) {
	var err error
	if cmdName == "" {
		cmdName, err = service.GetDefaultCmd()
		if err != nil {
			utils.OutputFatal(err)
		}
	}

	if !service.IsCmdExists(cmdName) {
		utils.OutputFatal("Error: cmd '" + cmdName + "' does not exist")
	}

	if controllerRoute == "" {
		controllerRoute, err = utils.InputStr("please enter controller route:")
		if err != nil {
			utils.OutputFatal(err)
		}
	}
	if controllerRoute == "" {
		utils.OutputFatal("Error: controller route is empty")
	}

	path, name, err := service.GetControllerPathAndNameByRoute(cmdName, controllerRoute)
	if err != nil {
		utils.OutputFatal(err)
	}
	err = service.ValidateControllerName(name)
	if err != nil {
		utils.OutputFatal(err)
	}

	if utils.IsFileExists(path) {
		utils.OutputFatal("Error: controller already exists")
	}

	err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		fmt.Println("Error creating dir:", err)
		return
	}
	tmplContent, err := templates.TemplateFS.ReadFile("default/internal/default-api/transport/http/api/controller.templ")
	if err != nil {
		utils.OutputFatal(fmt.Errorf("read template file: %w", err))
	}
	err = template.CreateFile(string(tmplContent),
		template.ControllerStructNameData{name},
		path,
	)
	if err != nil {
		utils.OutputFatal(err)
	}

	if len(actions) > 0 {
		err = service.WriteActions(path, name, actions)
		if err != nil {
			utils.OutputFatal(err)
		}
	}
}
