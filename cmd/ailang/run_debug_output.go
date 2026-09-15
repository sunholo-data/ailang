package main

import (
	"strings"
)

// Debug ghost-effect output for the CLI hosts (run, run --batch, serve-api):
// the --log-level threshold. The flush itself is runner.FlushDebugOutput,
// which takes the threshold as a parameter.

// debugLogLevel is the minimum severity level to print. Set by --log-level flag.
// 0=DEBUG/TRACE, 1=INFO, 2=WARNING, 3=ERROR, 4=NONE (suppress all)
var debugLogLevel int

// parseLogLevel converts a log level string to a numeric severity threshold.
func parseLogLevel(level string) int {
	switch strings.ToLower(level) {
	case "debug", "trace":
		return 0
	case "info", "":
		return 1
	case "warn", "warning":
		return 2
	case "error", "err":
		return 3
	case "none", "off":
		return 4
	default:
		return 1 // default to INFO
	}
}
