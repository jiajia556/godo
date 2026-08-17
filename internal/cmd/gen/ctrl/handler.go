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

var formatGoFiles = utils.FormatGoFiles

func genCtrl(cmdName, controllerRoute string, actions []string) error {
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

	if utils.IsFileExists(path) {
		return fmt.Errorf("controller already exists: %s", path)
	}

	if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create controller directory: %w", err)
	}
	tmplContent, err := templates.TemplateFS.ReadFile("default/internal/default-api/transport/http/api/controller.templ")
	if err != nil {
		return fmt.Errorf("read controller template: %w", err)
	}
	err = template.CreateFile(string(tmplContent),
		template.ControllerStructNameData{ControllerStructName: name},
		path,
	)
	if err != nil {
		return fmt.Errorf("write controller file: %w", err)
	}

	if len(actions) > 0 {
		err = service.WriteActions(path, name, actions)
		if err != nil {
			return fmt.Errorf("write controller actions: %w", err)
		}
	}
	if err = formatGoFiles(path); err != nil {
		return fmt.Errorf("format controller file: %w", err)
	}
	return nil
}
