package mdw

import "github.com/spf13/cobra"

var middlewareCmd = &cobra.Command{
	Use:     "mdw [middleware-name...]",
	Short:   "Create new middleware components",
	Long:    "Generates middleware files with specified names",
	Example: "  godo gen mdw auth\n  god gen mdw logging cache",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		genMiddleware(args)
	},
}

func GetCommand() *cobra.Command {
	return middlewareCmd
}
