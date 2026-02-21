package utils

import "strings"

// CapitalizeFirstLetter uppercases the first character of s.
func CapitalizeFirstLetter(s string) string {
	if len(s) == 0 {
		return s
	}

	return strings.ToUpper(string(s[0])) + s[1:]
}

// LowercaseFirstLetter lowercases the first character of s.
func LowercaseFirstLetter(s string) string {
	if len(s) == 0 {
		return s
	}

	return strings.ToLower(string(s[0])) + s[1:]
}
