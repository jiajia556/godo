package utils

import (
	"bytes"
	"strings"
	"testing"
)

func TestFirstLetterCaseConversionSupportsUnicode(t *testing.T) {
	if got := CapitalizeFirstLetter("éclair"); got != "Éclair" {
		t.Fatalf("CapitalizeFirstLetter() = %q, want %q", got, "Éclair")
	}
	if got := LowercaseFirstLetter("Éclair"); got != "éclair" {
		t.Fatalf("LowercaseFirstLetter() = %q, want %q", got, "éclair")
	}
	if got := CapitalizeFirstLetter("权限"); got != "权限" {
		t.Fatalf("CapitalizeFirstLetter() corrupted Unicode: %q", got)
	}
	if CapitalizeFirstLetter("") != "" || LowercaseFirstLetter("") != "" {
		t.Fatal("empty string conversion changed the value")
	}
}

func TestCamelToSnake(t *testing.T) {
	tests := map[string]string{
		"UserID":     "user_i_d",
		"userName":   "user_name",
		"Already_OK": "already__o_k",
		"":           "",
	}
	for input, want := range tests {
		if got := CamelToSnake(input); got != want {
			t.Errorf("CamelToSnake(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestLoggerWritesExpectedLevels(t *testing.T) {
	var output bytes.Buffer
	logger := NewLogger(false)
	logger.writer = &output
	logger.Info("hello %s", "world")
	logger.Warn("careful")
	logger.Debug("hidden")
	logger.Success("done")
	text := output.String()
	for _, expected := range []string{"INFO: hello world", "WARN: careful", "SUCCESS: done"} {
		if !strings.Contains(text, expected) {
			t.Errorf("logger output does not contain %q: %s", expected, text)
		}
	}
	if strings.Contains(text, "hidden") {
		t.Fatalf("disabled debug log was written: %s", text)
	}

	output.Reset()
	logger = NewLogger(true)
	logger.writer = &output
	logger.Debug("visible %d", 1)
	if !strings.Contains(output.String(), "DEBUG: visible 1") {
		t.Fatalf("debug output = %s", output.String())
	}
}
