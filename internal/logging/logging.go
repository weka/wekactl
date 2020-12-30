package logging

import (
	"fmt"
	"os"
	"strings"
)

const (
	ColorBlue   = "\033[1;34m"
	ColorGreen  = "\033[1;36m"
	ColorYellow = "\033[1;33m"
	ColorRed    = "\033[1;31m"
	ColorReset  = "\033[0m"

	ColorSuccess  = ColorGreen
	ColorWarning  = ColorYellow
	ColorProgress = ColorBlue
	ColorError    = ColorRed
	ColorFailure  = ColorRed
)

func Colorize(color, text string) string {
	return strings.Join([]string{color, text, ColorReset}, "")
}

// UserSuccess prints a colorized success message
func UserSuccess(msg string, format ...interface{}) {
	msg = fmt.Sprintf(msg, format...)
	fmt.Println(Colorize(ColorSuccess, msg))
}

// UserWarning prints a colorized warning message
func UserWarning(msg string, format ...interface{}) {
	msg = fmt.Sprintf("WARNING: "+msg, format...)
	fmt.Println(Colorize(ColorWarning, msg))
}

// UserProgress prints a colorized progress message
func UserProgress(msg string, format ...interface{}) {
	msg = fmt.Sprintf(msg, format...)
	fmt.Println(Colorize(ColorProgress, msg))
}

// UserFailure prints a colorized failure message
func UserFailure(msg string, format ...interface{}) {
	msg = fmt.Sprintf("ERROR: "+msg, format...)
	fmt.Println(Colorize(ColorFailure, msg))
}

// UserError prints a colorized error message and terminates with a non-zero exit code
func UserError(msg string, format ...interface{}) {
	msg = fmt.Sprintf("ERROR: "+msg, format...)
	fmt.Println(Colorize(ColorError, msg))
	os.Exit(2)
}
