package build

import (
	"github.com/jiajia556/godo/internal/cmd/gen/rt"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:     "build [cmd-name]",
	Short:   "Build a cmd module and output the binary to bin/",
	Long:    "Build compiles the specified cmd module (e.g. 'default-api') and writes the binary to the project's bin/ directory.\n\nBefore building, it will regenerate the HTTP router (same as running 'godo gen rt') to keep routes in sync with controllers.\n\nYou can optionally set the build version and cross-compile by specifying --goos/--goarch.",
	Example: "  godo build default-api\n  godo build default-api --version v1.2.0\n  godo build payment-service --goos linux --goarch amd64",
	Run: func(cmd *cobra.Command, args []string) {
		cmdName := ""
		if len(args) > 0 {
			cmdName = args[0]
		}
		version, _ := cmd.Flags().GetString("version")
		goos, _ := cmd.Flags().GetString("goos")
		goarch, _ := cmd.Flags().GetString("goarch")

		rt.GenRouter(cmdName)

		//build.Build(string(content), app, appRoot, apiRoot, version, goos, goarch, isApi)
		build(cmdName, version, goos, goarch)
	},
}

func GetCommand() *cobra.Command {
	return buildCmd
}

func init() {
	buildCmd.Flags().StringP("version", "v", "", "The version of the app")
	buildCmd.Flags().StringP("goos", "", "", "The target OS of the app")
	buildCmd.Flags().StringP("goarch", "", "", "The target architecture of the app")
}
