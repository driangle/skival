package color

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

var enabled = func() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	return term.IsTerminal(int(os.Stderr.Fd()))
}()

func wrap(code, s string) string {
	if !enabled || s == "" {
		return s
	}
	return code + s + "\033[0m"
}

func Green(s string) string  { return wrap("\033[32m", s) }
func Red(s string) string    { return wrap("\033[31m", s) }
func Yellow(s string) string { return wrap("\033[33m", s) }
func Cyan(s string) string   { return wrap("\033[36m", s) }
func Dim(s string) string    { return wrap("\033[2m", s) }

// Greenf formats and wraps in green.
func Greenf(format string, a ...any) string { return Green(fmt.Sprintf(format, a...)) }
func Redf(format string, a ...any) string   { return Red(fmt.Sprintf(format, a...)) }
func Yellowf(format string, a ...any) string { return Yellow(fmt.Sprintf(format, a...)) }
func Cyanf(format string, a ...any) string  { return Cyan(fmt.Sprintf(format, a...)) }
func Dimf(format string, a ...any) string   { return Dim(fmt.Sprintf(format, a...)) }

// SetEnabled overrides automatic TTY/NO_COLOR detection. Useful for testing.
func SetEnabled(v bool) { enabled = v }

// Enabled reports whether color output is active.
func Enabled() bool { return enabled }
