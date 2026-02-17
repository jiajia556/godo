package gen

import (
	"github.com/jiajia556/godo/internal/cmd/gen/act"
	"github.com/jiajia556/godo/internal/cmd/gen/cmd"
	"github.com/jiajia556/godo/internal/cmd/gen/ctrl"
	"github.com/jiajia556/godo/internal/cmd/gen/rt"
	"github.com/spf13/cobra"
)

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate Go code",
	Long:  `Generate Go code for controllers, models, middleware etc.`,
}

func GetCommand() *cobra.Command {
	return genCmd
}

func init() {
	genCmd.AddCommand(
		cmd.GetCommand(),
		ctrl.GetCommand(),
		act.GetCommand(),
		rt.GetCommand(),
	)
}
