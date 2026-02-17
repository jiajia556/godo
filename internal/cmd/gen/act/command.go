package act

import "github.com/spf13/cobra"

var actionCmd = &cobra.Command{
	Use:     "act [actions...]",
	Short:   "Add actions to an existing controller",
	Long:    "Adds one or more action methods to a specified controller",
	Example: "  godo gen act getInfo --ctrl user/user\n  god gen act search list",
	Run: func(cmd *cobra.Command, args []string) {
		cmdName, _ := cmd.Flags().GetString("cmd")
		controllerRoute, _ := cmd.Flags().GetString("ctrl")
		genAction(cmdName, controllerRoute, args)
	},
}

func GetCommand() *cobra.Command {
	return actionCmd
}

func init() {
	actionCmd.Flags().StringP("cmd", "", "", "The cmd that requires the action, e.g. 'default-api'")
	actionCmd.Flags().StringP("ctrl", "c", "", "The controller that requires the action, e.g. 'user/user'")
}
