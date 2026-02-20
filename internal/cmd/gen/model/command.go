package model

import "github.com/spf13/cobra"

var modelCmd = &cobra.Command{
	Use:     "model",
	Short:   "Generate database model files",
	Long:    "Generate Go model files from SQL schema definitions or from existing database.\nCreates record and list type files based on SQL CREATE TABLE statements.",
	Example: "  godo gen model config.json\n  godo gen model schema.sql",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		genModel(args[0])
	},
}

func GetCommand() *cobra.Command {
	return modelCmd
}
