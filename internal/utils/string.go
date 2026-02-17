package utils

import "strings"

func CapitalizeFirstLetter(s string) string {
	if len(s) == 0 {
		return s // Return directly if string is empty
	}

	// Capitalize first letter, keep the rest unchanged
	return strings.ToUpper(string(s[0])) + s[1:]
}
