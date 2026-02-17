package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type NumInt interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// OutputFatal prints a formatted message to stderr and exits with code 1.
// It accepts variadic arguments similar to fmt.Println; they will be joined into a single line.
func OutputFatal(msg ...any) {
	// join into one line with spaces and a newline
	s := fmt.Sprintln(msg...)
	_, _ = fmt.Fprint(os.Stderr, s)
	os.Exit(1)
}

// OutputErrorf prints a formatted error message to stderr but does NOT exit.
func OutputErrorf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// OutputInfof prints a formatted informational message to stdout.
func OutputInfof(format string, args ...any) {
	_, _ = fmt.Printf(format+"\n", args...)
}

func InputNum[T comparable](msg string) (num T, err error) {
	fmt.Println(msg)
	_, err = fmt.Scanf("%d", &num)
	if err != nil {
		return
	}
	return
}

func InputStr(msg string) (str string, err error) {
	fmt.Println(msg)
	reader := bufio.NewReader(os.Stdin)
	str, err = reader.ReadString('\n')
	if err != nil {
		return
	}
	str = strings.Trim(str, "\r\n")
	str = strings.Trim(str, "\n")
	return
}
