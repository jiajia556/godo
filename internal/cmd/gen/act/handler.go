package act

import (
	"fmt"
	"strings"

	"github.com/jiajia556/godo/internal/service"
	"github.com/jiajia556/godo/internal/utils"
)

var formatGoFiles = utils.FormatGoFiles

func genAction(cmdName, controllerRoute string, actions []string) error {
	var err error
	if len(actions) == 0 {
		actionStr, err := utils.InputStr("please enter actions (space separated):")
		if err != nil {
			return fmt.Errorf("read actions: %w", err)
		}
		if actionStr != "" {
			actions = strings.Fields(actionStr)
		}
	}
	if len(actions) == 0 {
		return fmt.Errorf("actions are empty")
	}
	if cmdName == "" {
		cmdName, err = service.GetDefaultCmd()
		if err != nil {
			return fmt.Errorf("get default command: %w", err)
		}
	}
	if err := service.ValidateCmdName(cmdName); err != nil {
		return fmt.Errorf("validate command name: %w", err)
	}

	if !service.IsCmdExists(cmdName) {
		return fmt.Errorf("command %q does not exist", cmdName)
	}
	if err := service.RequireCmdType(cmdName, service.CmdTypeAPI); err != nil {
		return err
	}

	if controllerRoute == "" {
		controllerRoute, err = utils.InputStr("please enter controller route:")
		if err != nil {
			return fmt.Errorf("read controller route: %w", err)
		}
	}
	if controllerRoute == "" {
		return fmt.Errorf("controller route is empty")
	}

	path, name, err := service.GetControllerPathAndNameByRoute(cmdName, controllerRoute)
	if err != nil {
		return fmt.Errorf("resolve controller path: %w", err)
	}
	err = service.ValidateControllerName(name)
	if err != nil {
		return fmt.Errorf("validate controller name: %w", err)
	}

	if !utils.IsFileExists(path) {
		return fmt.Errorf("controller does not exist: %s", path)
	}

	err = service.WriteActions(path, name, actions)
	if err != nil {
		return fmt.Errorf("write controller actions: %w", err)
	}
	if err = formatGoFiles(path); err != nil {
		return fmt.Errorf("format controller file: %w", err)
	}
	return nil
}
