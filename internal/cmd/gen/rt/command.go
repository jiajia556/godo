package rt

import "github.com/spf13/cobra"

var routerCmd = &cobra.Command{
	Use:     "rt",
	Short:   "Generate API router configuration",
	Long:    "Creates or updates the main router file based on existing controllers",
	Example: "  godo gen rt",
	Run: func(cmd *cobra.Command, args []string) {
		cmdName, _ := cmd.Flags().GetString("cmd")
		genRouter(cmdName)
	},
}

func GetCommand() *cobra.Command {
	return routerCmd
}

func init() {
	routerCmd.Flags().StringP("cmd", "", "", "The cmd that requires the router, e.g. 'default-api'")
}
