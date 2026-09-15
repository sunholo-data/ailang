package runner

import "github.com/fatih/color"

// Colour helpers for the status lines Run prints. The same fatih/color
// SprintFuncs cmd/ailang uses, so output is byte-identical to the CLI's own
// (color.NoColor is one process-wide switch).
var (
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
	bold   = color.New(color.Bold).SprintFunc()
)
