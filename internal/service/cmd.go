package service

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/jiajia556/godo/internal/utils"
)

func ValidateCmdName(name string) error {
	if name == "" || strings.TrimSpace(name) != name {
		return fmt.Errorf("command name must not be empty or contain leading/trailing whitespace")
	}
	for i, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || (i > 0 && (r == '-' || r == '_')) {
			continue
		}
		return fmt.Errorf("command name contains invalid character %q", r)
	}
	return nil
}

func IsCmdExists(cmdName string) bool {
	if ValidateCmdName(cmdName) != nil {
		return false
	}
	var err error
	path := "cmd/" + cmdName
	path, err = GetAbsPath(path)
	if err != nil {
		return false
	}
	return utils.IsDirExists(path)
}

func ValidateBuildTarget(goos, goarch string) error {
	if goos == "" || goarch == "" {
		return fmt.Errorf("GOOS and GOARCH must not be empty")
	}
	output, err := utils.NewCommandRunner().RunCommandOutput("go", "tool", "dist", "list")
	if err != nil {
		return fmt.Errorf("list supported Go build targets: %w\n%s", err, strings.TrimSpace(output))
	}
	target := goos + "/" + goarch
	for _, supported := range strings.Fields(output) {
		if supported == target {
			return nil
		}
	}
	return fmt.Errorf("unsupported Go build target %s", target)
}
