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
	mu       sync.Mutex
	verbose  = false
	noColor  = false
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

func colorize(colorCode, msg string) string {
	if noColor {
		return msg
	}
	return fmt.Sprintf("%s%s%s", colorCode, msg, reset)
}

func Info(msg string) {
	if !IsVerbose() {
		return
	}
	fmt.Println(colorize(cyan, "comquad: "+msg))
}

func Success(msg string) {
	if !IsVerbose() {
		return
	}
	fmt.Println(colorize(green, "comquad: "+msg))
}

func Warn(msg string) {
	if !IsVerbose() {
		return
	}
	fmt.Println(colorize(yellow, "comquad: "+msg))
}

// Error always prints to stderr regardless of verbose mode.
func Error(msg string) {
	fmt.Fprintln(os.Stderr, colorize(red, "comquad: "+msg))
}

func Action(msg string) {
	if !IsVerbose() {
		return
	}
	fmt.Println(colorize(blue, "comquad: "+msg))
}
