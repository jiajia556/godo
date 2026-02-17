package cmd

import "github.com/spf13/cobra"

var cmdCmd = &cobra.Command{
	Use:     "cmd [cmd-name]",
	Short:   "Create a new cmd",
	Long:    "Generates a new cmd",
	Example: "  godo gen cmd home-api",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cmdName := args[0]
		genCmd(cmdName)
	},
}

func GetCommand() *cobra.Command {
	return cmdCmd
}
