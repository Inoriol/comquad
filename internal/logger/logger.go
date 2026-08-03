package logger

import (
	"fmt"
	"os"
	"sync"
)

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	blue   = "\033[34m"
)

var (
	mu      sync.Mutex
	verbose = false
	quiet   = false
	noColor = false
)

func init() {
	// Disable colors if NO_COLOR is set
	_, noColor = os.LookupEnv("NO_COLOR")
}

func SetVerbose(v bool) {
	mu.Lock()
	defer mu.Unlock()
	verbose = v
}

func IsVerbose() bool {
	mu.Lock()
	defer mu.Unlock()
	return verbose
}

// SetQuiet suppresses all non-error output. Takes precedence over verbose.
func SetQuiet(q bool) {
	mu.Lock()
	defer mu.Unlock()
	quiet = q
}

func IsQuiet() bool {
	mu.Lock()
	defer mu.Unlock()
	return quiet
}

// Print outputs a plain (uncolored, unprefixed) operational message.
// It is suppressed when quiet mode is active, but does not require verbose.
// Use this for normal runtime messages like "Starting unit: foo".
func Print(msg string) {
	mu.Lock()
	q := quiet
	mu.Unlock()
	if q {
		return
	}
	fmt.Println(msg)
}

// Printf formats and prints a plain message, suppressed when quiet mode is active.
func Printf(format string, args ...interface{}) {
	mu.Lock()
	q := quiet
	mu.Unlock()
	if q {
		return
	}
	fmt.Printf(format, args...)
}

func colorize(colorCode, msg string) string {
	mu.Lock()
	n := noColor
	mu.Unlock()
	if n {
		return msg
	}
	// Check NO_COLOR dynamically in case it was set after init()
	if _, dynamic := os.LookupEnv("NO_COLOR"); dynamic {
		return msg
	}
	return fmt.Sprintf("%s%s%s", colorCode, msg, reset)
}

func Info(msg string) {
	if !IsVerbose() || IsQuiet() {
		return
	}
	fmt.Println(colorize(cyan, "comquad: "+msg))
}

func Success(msg string) {
	if IsQuiet() {
		return
	}
	fmt.Println(colorize(green, "comquad: "+msg))
}

func Warn(msg string) {
	if IsQuiet() {
		return
	}
	fmt.Println(colorize(yellow, "comquad: "+msg))
}

// Error always prints to stderr regardless of verbose or quiet mode.
func Error(msg string) {
	fmt.Fprintln(os.Stderr, colorize(red, "comquad: "+msg))
}

func Action(msg string) {
	if IsQuiet() {
		return
	}
	fmt.Println(colorize(blue, "comquad: "+msg))
}
