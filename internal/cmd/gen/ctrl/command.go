package ctrl

import "github.com/spf13/cobra"

var ctrlCmd = &cobra.Command{
	Use:     "ctrl [controller-route] [actions...]",
	Short:   "Create a new controller with optional actions",
	Long:    "Generates a new controller file with specified route and optional initial actions",
	Example: "  godo gen ctrl user\n  god gen ctrl product list create update",
	Args:    cobra.MinimumNArgs(1), // Requires at least 1 argument
	Run: func(cmd *cobra.Command, args []string) {
		// Extract actions from arguments
		var actions []string
		if len(args) > 1 {
			actions = args[1:]
		}

		cmdName, _ := cmd.Flags().GetString("cmd")
		genCtrl(cmdName, args[0], actions)
	},
}

func GetCommand() *cobra.Command {
	return ctrlCmd
}

func init() {
	ctrlCmd.Flags().StringP("cmd", "", "", "The cmd that requires the controller, e.g. 'default-api'")
}
