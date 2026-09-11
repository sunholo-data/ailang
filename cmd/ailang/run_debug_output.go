package main

import (
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
)

// Debug ghost-effect output for the CLI hosts (run, run --batch): the
// --log-level threshold and the flush that routes collected lines through
// the shared effects.DebugSink.

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

// flushDebugOutput collects Debug effect logs and prints them to stderr via
// the shared effects.DebugSink. Respects debugLogLevel for severity
// filtering. A non-empty label prefixes UNSTRUCTURED lines so batch output
// remains attributable even under --quiet; a structured (JSON-object) line is
// written verbatim so log consumers can parse it (M-DEBUG-SINK-STRUCTURED-LINES).
func flushDebugOutput(effCtx *effects.EffContext, label string) {
	if effCtx == nil {
		return
	}
	effects.DebugSink{W: os.Stderr, MinLevel: debugLogLevel, Label: label}.Flush(effCtx.Debug)
}
