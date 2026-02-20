package build

import (
	"github.com/jiajia556/godo/internal/cmd/gen/rt"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:     "build [app-name]",
	Short:   "",
	Long:    "",
	Example: "  god build default-api\n  god build default-api --version v1.2.0\n  god build payment-service --goos linux --goarch amd64",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cmdName := args[0]
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
