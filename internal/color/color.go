// Package color provides ANSI color helpers for terminal output.
// Respects the NO_COLOR environment variable.
package color

import (
	"fmt"
	"os"
	"strings"
)

// Color represents an ANSI color code.
type Color string

const (
	Reset  Color = "\033[0m"
	Red    Color = "\033[31m"
	Green  Color = "\033[32m"
	Yellow Color = "\033[33m"
	Cyan   Color = "\033[36m"
	Gray   Color = "\033[90m"
)

var disabled = os.Getenv("NO_COLOR") != ""

// Sprint returns the string wrapped in the color code and reset.
func (c Color) Sprint(a ...any) string {
	if disabled {
		return fmt.Sprint(a...)
	}
	return string(c) + fmt.Sprint(a...) + string(Reset)
}

// Sprintf returns the formatted string wrapped in the color code and reset.
func (c Color) Sprintf(format string, a ...any) string {
	if disabled {
		return fmt.Sprintf(format, a...)
	}
	return string(c) + fmt.Sprintf(format, a...) + string(Reset)
}

// Print prints the arguments in the color.
func (c Color) Print(a ...any) {
	if disabled {
		fmt.Print(a...)
		return
	}
	fmt.Print(string(c))
	fmt.Print(a...)
	fmt.Print(string(Reset))
}

// Printf prints the formatted string in the color.
func (c Color) Printf(format string, a ...any) {
	if disabled {
		fmt.Printf(format, a...)
		return
	}
	fmt.Print(string(c))
	fmt.Printf(format, a...)
	fmt.Print(string(Reset))
}

// Disable permanently disables all colors.
func Disable() {
	disabled = true
}

// --- Semantic helpers ---

// DryRun returns the text colored for dry-run previews (yellow).
func DryRun(text string) string {
	return Yellow.Sprint(text)
}

// Action returns the text colored for actual actions (green).
func Action(text string) string {
	return Green.Sprint(text)
}

// Path returns the text colored as a file path (cyan).
func Path(text string) string {
	return Cyan.Sprint(text)
}

// Arrow returns the arrow separator colored in gray.
func Arrow() string {
	return Gray.Sprint(" -> ")
}

// Error returns the text colored as an error (red).
func Error(text string) string {
	return Red.Sprint(text)
}

// Info returns the text colored as an informational message (cyan).
func Info(text string) string {
	return Cyan.Sprint(text)
}

// Strip removes ANSI escape codes from a string.
func Strip(s string) string {
	for {
		start := strings.Index(s, "\033[")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], "m")
		if end == -1 {
			break
		}
		s = s[:start] + s[start+end+1:]
	}
	return s
}
