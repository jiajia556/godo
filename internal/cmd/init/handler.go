package init

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jiajia556/godo/internal/template"
	"github.com/jiajia556/godo/internal/utils"
	"github.com/jiajia556/godo/templates"
)

func initProject(name string) {
	defer func() {
		cmdRunner := utils.NewCommandRunner().WithDir("./" + name)
		cmdRunner.RunCommand("go", "mod", "tidy")
		_, err := exec.LookPath("goimports")
		if err != nil {
			cmdRunner.RunCommand("go", "install", "golang.org/x/tools/cmd/goimports@latest")
		}
	}()

	templateDir := templates.DEFAULT_TEMPLATE_DIR
	_ = fs.WalkDir(templates.TemplateFS, templateDir, func(originalPath string, d fs.DirEntry, err error) error {
		if err != nil {
			utils.OutputFatal(err)
		}
		if d.IsDir() {
			return nil
		}

		path := name + strings.TrimPrefix(originalPath, templateDir)
		dirPath := filepath.Dir(path)
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

		ProjectNameTmpls := []string{
			"go.mod", "main.go",
			"godoconfig.json",
			"baserecord.go",
			"baselist.go",
			"outputmsg.go",
			"config.go",
		}
		if slices.Contains(ProjectNameTmpls, fileName) {
			data := template.ProjectNameData{ProjectName: name, CmdName: "default-api"}
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
