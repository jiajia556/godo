package act

import (
	"strings"

	"github.com/jiajia556/godo/internal/service"
	"github.com/jiajia556/godo/internal/utils"
)

func genAction(cmdName, controllerRoute string, actions []string) {
	var err error
	if len(actions) == 0 {
		actionStr, err := utils.InputStr("please enter actions (space separated):")
		if err != nil {
			utils.OutputFatal(err)
		}
		if actionStr != "" {
			actions = strings.Split(actionStr, " ")
		}
	}
	if len(actions) == 0 {
		utils.OutputFatal("Error: actions is empty")
	}
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

	if !utils.IsFileExists(path) {
		utils.OutputFatal("Error: controller is not exists")
	}

	err = service.WriteActions(path, name, actions)
	if err != nil {
		utils.OutputFatal(err)
	}
}
