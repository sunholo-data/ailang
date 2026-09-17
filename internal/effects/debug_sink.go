package effects

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	// W receives structured lines verbatim. nil means the CURRENT os.Stderr,
	// resolved at each write — a sink attached at start-up must follow a
	// host (or test) that swaps stderr later.
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
// line is unstructured or carries no severity. One decode over the string —
// the previous json.Valid([]byte) + json.Unmarshal([]byte) copied the line
// twice per call.
func Severity(msg string) string {
	trimmed := strings.TrimSpace(msg)
	if !strings.HasPrefix(trimmed, "{") {
		return ""
	}
	var parsed struct {
		Severity string `json:"severity"`
	}
	dec := json.NewDecoder(strings.NewReader(trimmed))
	if err := dec.Decode(&parsed); err != nil {
		return ""
	}
	// A trailing non-whitespace token means it was not ONE object.
	if _, err := dec.Token(); err != io.EOF {
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

// Attach makes d stream through s: every line and failed check is written on
// arrival under s.MinLevel and nothing is retained (M-V1-MEMORY-FOOTPRINT M2).
// Flush remains valid afterwards and finds nothing to drain.
func (s DebugSink) Attach(d *DebugContext) {
	if d == nil {
		return
	}
	sink := s
	d.sink = &sink
	d.minLevel = s.MinLevel
}

func (s DebugSink) w() io.Writer {
	if s.W != nil {
		return s.W
	}
	return os.Stderr
}

func (s DebugSink) logf() func(format string, args ...any) {
	if s.Logf != nil {
		return s.Logf
	}
	return func(format string, args ...any) { fmt.Fprintf(s.w(), format+"\n", args...) }
}

func (s DebugSink) prefix() string {
	if s.Label != "" {
		return "[" + s.Label + "] "
	}
	return ""
}

// writeLine writes one already level-filtered log line: a structured line
// verbatim, anything else decorated.
func (s DebugSink) writeLine(l LogEntry) {
	if IsStructuredLine(l.Message) {
		fmt.Fprintln(s.w(), l.Message)
		return
	}
	s.logf()("%s%s", s.prefix(), l.Message)
}

// writeAssertion writes one FAILED check.
func (s DebugSink) writeAssertion(a AssertionResult) {
	if s.Structured {
		line, err := json.Marshal(structuredAssertion{
			Severity: "ERROR",
			Message:  "assertion failed: " + a.Message,
			Location: a.Location,
			Source:   "Debug.check",
		})
		if err == nil {
			fmt.Fprintln(s.w(), string(line))
			return
		}
		// json.Marshal cannot fail on plain strings; fall through to text
		// rather than lose the failure.
	}
	s.logf()("%s[ASSERT FAIL] %s at %s", s.prefix(), a.Message, a.Location)
}

// Flush collects, filters, writes and resets d. A nil context is a no-op.
// With a sink attached the context holds nothing and this is a no-op too.
func (s DebugSink) Flush(d *DebugContext) {
	if d == nil {
		return
	}
	out := d.Collect()
	for _, l := range out.Logs {
		if !passesLevel(l.Message, s.MinLevel) {
			continue
		}
		s.writeLine(l)
	}
	for _, a := range out.Assertions {
		if a.Passed {
			continue
		}
		s.writeAssertion(a)
	}
	d.Reset()
}
