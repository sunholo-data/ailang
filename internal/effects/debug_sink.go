package effects

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// DebugSink writes collected Debug output to a host (CLI stderr, serve-api).
//
// It is the single place that decides how a Debug.log line reaches the
// outside world, so every host applies one rule: a line that is a bare JSON
// object is written VERBATIM on its own line — no label, no timestamp — so
// JSON-line consumers (Cloud Logging, jq, Loki) can lift `severity` from it.
// Every other line gets the host's decoration exactly as before.
//
// M-DEBUG-SINK-STRUCTURED-LINES: before this type, serve-api wrapped every
// line in log.Printf("[Debug] %s") and `run --batch` prefixed "[input] ",
// turning {"severity":"ERROR",…} into textPayload at DEFAULT severity.
type DebugSink struct {
	// W receives structured lines verbatim. Required.
	W io.Writer
	// Logf receives unstructured lines (and text-mode assertion failures)
	// with the host's decoration — typically log.Printf. nil = write to W.
	Logf func(format string, args ...any)
	// MinLevel is the --log-level threshold: 0=DEBUG … 4=NONE. Only
	// structured lines carry a severity; unstructured lines always pass.
	MinLevel int
	// Label prefixes UNSTRUCTURED lines as "[label] " (batch attribution).
	// A structured line is never modified — attribution is the program's job.
	Label string
	// Structured emits assertion failures as {"severity":"ERROR",…} JSON
	// (serve-api) instead of the human "[ASSERT FAIL] msg at loc" line (CLI).
	Structured bool
}

// IsStructuredLine reports whether msg is a bare JSON object — the same
// predicate Cloud Logging applies when deciding jsonPayload vs textPayload.
// Arrays, scalars and invalid JSON are not structured; they are decorated
// like any other text and never dropped.
func IsStructuredLine(msg string) bool {
	trimmed := strings.TrimSpace(msg)
	return strings.HasPrefix(trimmed, "{") && json.Valid([]byte(trimmed))
}

// Severity returns the "severity" field of a structured line, or "" when the
// line is unstructured or carries no severity.
func Severity(msg string) string {
	if !IsStructuredLine(msg) {
		return ""
	}
	var parsed struct {
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(msg)), &parsed); err != nil {
		return ""
	}
	return parsed.Severity
}

// SeverityLevel maps a severity string to the --log-level scale.
// Unknown or empty severities rank as INFO so a typo is shown, not hidden.
func SeverityLevel(severity string) int {
	switch severity {
	case "DEBUG", "TRACE":
		return 0
	case "INFO":
		return 1
	case "WARNING":
		return 2
	case "ERROR":
		return 3
	default:
		return 1
	}
}

// structuredAssertion is the JSON shape of a failed Debug.check in
// Structured mode. Field order is fixed by the struct so the line is stable.
type structuredAssertion struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Location string `json:"location"`
	Source   string `json:"source"`
}

// Flush collects, filters, writes and resets d. A nil context is a no-op.
func (s DebugSink) Flush(d *DebugContext) {
	if d == nil {
		return
	}
	logf := s.Logf
	if logf == nil {
		logf = func(format string, args ...any) { fmt.Fprintf(s.W, format+"\n", args...) }
	}
	prefix := ""
	if s.Label != "" {
		prefix = "[" + s.Label + "] "
	}

	out := d.Collect()
	for _, l := range out.Logs {
		if IsStructuredLine(l.Message) {
			// A structured line with no severity field always passes, as
			// before — the filter only judges lines that state a level.
			if sev := Severity(l.Message); s.MinLevel > 0 && sev != "" && SeverityLevel(sev) < s.MinLevel {
				continue
			}
			fmt.Fprintln(s.W, l.Message)
			continue
		}
		logf("%s%s", prefix, l.Message)
	}
	for _, a := range out.Assertions {
		if a.Passed {
			continue
		}
		if s.Structured {
			line, err := json.Marshal(structuredAssertion{
				Severity: "ERROR",
				Message:  "assertion failed: " + a.Message,
				Location: a.Location,
				Source:   "Debug.check",
			})
			if err == nil {
				fmt.Fprintln(s.W, string(line))
				continue
			}
			// json.Marshal cannot fail on plain strings; fall through to text
			// rather than lose the failure.
		}
		logf("%s[ASSERT FAIL] %s at %s", prefix, a.Message, a.Location)
	}
	d.Reset()
}
