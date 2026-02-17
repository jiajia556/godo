package service

import "github.com/jiajia556/godo/internal/utils"

func IsCmdExists(cmdName string) bool {
	var err error
	path := "cmd/" + cmdName
	path, err = GetAbsPath(path)
	if err != nil {
		return false
	}
	return utils.IsDirExists(path)
}
