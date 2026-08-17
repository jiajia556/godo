package config

import (
	"fmt"

	"github.com/jiajia556/godo/internal/service"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage godoconfig.json",
}

var setCmd = &cobra.Command{
	Use:     "set [key] [value]",
	Short:   "Update a modifiable project configuration value",
	Long:    "Update one writable field in godoconfig.json. Allowed keys: default_cmd, default_goos, and default_goarch.",
	Example: "  godo config set default_cmd jobs-worker\n  godo config set default_goos windows\n  godo config set default_goarch amd64",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		value, err := service.SetConfigValue(args[0], args[1])
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", args[0], value)
		return err
	},
}

var setTargetCmd = &cobra.Command{
	Use:     "set-target [goos] [goarch]",
	Short:   "Update the default Go build target",
	Long:    "Validate and atomically update default_goos and default_goarch in godoconfig.json.",
	Example: "  godo config set-target linux amd64\n  godo config set-target js wasm",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		goos, goarch, err := service.SetBuildTarget(args[0], args[1])
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "default_goos=%s\ndefault_goarch=%s\n", goos, goarch)
		return err
	},
}

func GetCommand() *cobra.Command {
	return configCmd
}

func init() {
	configCmd.AddCommand(setCmd, setTargetCmd)
}
