package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jiajia556/godo/internal/service"
	"github.com/jiajia556/godo/internal/template"
	"github.com/jiajia556/godo/internal/utils"
	"github.com/jiajia556/godo/templates"
)

func genCmd(cmdName string) {
	templateDir := templates.DEFAULT_TEMPLATE_DIR
	cmdDirs := []string{templateDir + "/cmd", templateDir + "/internal/default-api"}
	for _, cmdDir := range cmdDirs {
		_ = fs.WalkDir(templates.TemplateFS, cmdDir, func(originalPath string, d fs.DirEntry, err error) error {
			if err != nil {
				utils.OutputFatal(err)
			}
			if d.IsDir() {
				return nil
			}

			path := strings.TrimPrefix(originalPath, templateDir)
			path = strings.ReplaceAll(path, "default-api", cmdName)
			path, err = service.GetAbsPath(path)
			if err != nil {
				utils.OutputFatal(err)
			}
			dirPath := filepath.Dir(path)
			if utils.DirExists(dirPath) {
				utils.OutputFatal(cmdName, "already exists")
			}
			err = os.MkdirAll(dirPath, 0755)
			if err != nil {
				utils.OutputFatal(err)
			}

			if !strings.HasSuffix(path, ".tmpl") {
				return nil
			}
			targetPath := path[:len(path)-5]

			contentByte, err := fs.ReadFile(templates.TemplateFS, originalPath)
			if err != nil {
				utils.OutputFatal(err)
			}
			content := string(contentByte)

			fileName := filepath.Base(targetPath)
			projectName, err := service.GetProjectName()
			if err != nil {
				utils.OutputFatal(err)
			}

			ProjectNameTmpls := []string{
				"go.mod", "main.go",
				"godopackage.json",
				"baserecord.go",
				"baselist.go",
				"outputmsg.go",
				"config.go",
			}
			if slices.Contains(ProjectNameTmpls, fileName) {
				data := template.ProjectNameData{ProjectName: projectName, CmdName: cmdName}
				err = template.CreateFile(content, data, targetPath)
				if err != nil {
					utils.OutputFatal(err)
				}
			} else {
				f, _ := os.Create(targetPath)
				_, err = f.WriteString(content)
				if err != nil {
					utils.OutputFatal(err)
				}
				err = f.Close()
				if err != nil {
					utils.OutputFatal(err)
				}
			}
			return nil
		})
	}
}
