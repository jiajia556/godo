package cmd

import (
	"github.com/jiajia556/godo/internal/service"
	"github.com/spf13/cobra"
)

var cmdCmd = &cobra.Command{
	Use:     "cmd [cmd-name]",
	Short:   "Create a new cmd",
	Long:    "Generates a new cmd",
	Example: "  godo gen cmd home-api\n  godo gen cmd order-worker --type worker",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmdName := args[0]
		cmdType, _ := cmd.Flags().GetString("type")
		return genCmd(cmdName, cmdType)
	},
}

func GetCommand() *cobra.Command {
	return cmdCmd
}

func init() {
	cmdCmd.Flags().StringP("type", "t", service.CmdTypeAPI, "Command type: api or worker")
}
