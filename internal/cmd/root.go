package cmd

import (
	"github.com/jiajia556/godo/internal/cmd/build"
	"github.com/jiajia556/godo/internal/cmd/gen"
	initproj "github.com/jiajia556/godo/internal/cmd/init"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "godo",
	Short: "Godo - Go Development Accelerator Tool",
	Long: `A CLI tool to accelerate Go web application development with code generation and project scaffolding.

Complete documentation is available at: https://github.com/jiajia556/godo`,
	Version: "1.0.0",
}

// GetRootCmd 获取根命令
func GetRootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	// 注册子命令
	rootCmd.AddCommand(
		initproj.GetCommand(),
		gen.GetCommand(),
		build.GetCommand(),
	)

	// 全局 flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
}
