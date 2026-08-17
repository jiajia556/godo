package utils

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// CapitalizeFirstLetter uppercases the first character of s.
func CapitalizeFirstLetter(s string) string {
	if len(s) == 0 {
		return s
	}

	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size == 1 {
		return s
	}
	return string(unicode.ToUpper(r)) + s[size:]
}

// LowercaseFirstLetter lowercases the first character of s.
func LowercaseFirstLetter(s string) string {
	if len(s) == 0 {
		return s
	}

	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size == 1 {
		return s
	}
	return string(unicode.ToLower(r)) + s[size:]
}

func CamelToSnake(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}

	return strings.ToLower(string(result))
}
