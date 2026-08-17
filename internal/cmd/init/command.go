package init

import "github.com/spf13/cobra"

// initCmd handles project initialization
var initCmd = &cobra.Command{
	Use:     "init [project-name]",
	Short:   "Create a new project",
	Long:    "Initialize a new project with the specified name and basic structure",
	Example: "  godo init myproject\n  god init example.com/myapp",
	Args:    cobra.ExactArgs(1), // Requires exactly 1 argument
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		return initProject(projectName)
	},
}

func GetCommand() *cobra.Command {
	return initCmd
}
