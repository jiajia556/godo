package cmd

import "github.com/jiajia556/godo/internal/utils"

// Execute 程序主入口，执行 CLI 命令
func Execute() {
	if err := GetRootCmd().Execute(); err != nil {
		utils.OutputFatal(err)
	}
}
