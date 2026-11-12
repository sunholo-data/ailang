package main

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/elaborate"
)

// Pipeline warnings in the machine-readable check outputs
// (M-V1-MATCH-EXHAUSTIVENESS M3). `ailang check` printed warnings only to a
// human's stderr; `--json` returned `"errors": []` and `--format agent`
// returned `ok: <file>`, so an AI author could not see them at all. Both
// formats now carry them, additively: a `warnings` array (omitted when
// empty) and one `warn:` line per warning after the verdict line.

// warningCoder and warningPositioner are the optional structure a pipeline
// warning may expose beyond elaborate.Warning's String().
type warningCoder interface{ Code() string }
type warningPositioner interface{ Position() (line, col int) }

// warningFiler names the warning's own source file when it may sit in an
// imported module rather than the checked file; warningSuggester gives a
// one-line fix; warningMessager gives the message without the location and
// fix the structured fields already carry.
type warningFiler interface{ WarningFile() string }
type warningSuggester interface{ Suggestion() string }
type warningMessager interface{ Message() string }

// checkWarningsJSON converts pipeline warnings to check-output entries; nil
// when there are none, so the field is omitted.
func checkWarningsJSON(ws []elaborate.Warning, filename string) []checkJSONError {
	var out []checkJSONError
	for _, w := range ws {
		if w == nil {
			continue
		}
		out = append(out, warningToCheckJSON(w, filename))
	}
	return out
}

func warningToCheckJSON(w elaborate.Warning, filename string) checkJSONError {
	e := checkJSONError{Code: "WARNING", Message: oneLine(w.String()), File: filename}
	if c, ok := w.(warningCoder); ok {
		e.Code = c.Code()
		// The rendered text leads with "<CODE> at <loc>"; the fields say both.
		e.Message = strings.TrimPrefix(e.Message, e.Code+" ")
	}
	if p, ok := w.(warningPositioner); ok {
		e.Line, e.Column = p.Position()
	}
	if f, ok := w.(warningFiler); ok && f.WarningFile() != "" {
		e.File = f.WarningFile()
	}
	if s, ok := w.(warningSuggester); ok {
		e.Suggestion = s.Suggestion()
	}
	if m, ok := w.(warningMessager); ok {
		e.Message = m.Message()
	}
	return e
}

// oneLine collapses a multi-line warning rendering into one line and drops a
// leading "warning: ", which the output format already conveys.
func oneLine(s string) string {
	fields := strings.Fields(s)
	out := strings.Join(fields, " ")
	return strings.TrimPrefix(out, "warning: ")
}

// agentWarnLine renders one warning for --format agent.
func agentWarnLine(w checkJSONError, fallbackFile string) string {
	return fmt.Sprintf("warn: %s", distillAgentLine(w, fallbackFile))
}
