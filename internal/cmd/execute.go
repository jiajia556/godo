package cmd

import "github.com/jiajia556/godo/internal/utils"

// Execute is the CLI entrypoint.
func Execute() {
	if err := GetRootCmd().Execute(); err != nil {
		utils.OutputFatal(err)
	}
}
