package eval_harness

import (
	"fmt"
	"strings"
)

// SourceConstraints defines deterministic, language-neutral checks applied to
// the GENERATED SOURCE (not its output). They unlock the constrained-
// construction benchmark class (M-EVAL-FRONTIER-TIER follow-up): tasks where
// the model itself must count, construct, or avoid tokens at generation time —
// difficulty that cannot be delegated to the program it writes, because the
// constraint is on the program text.
//
// All checks run on normalized source: CRLF/CR converted to LF, then ALL
// trailing newline characters stripped. Benchmarks using byte-count
// constraints must state this normalization in their task_prompt so the model
// is judged against exactly the rule it was told.
type SourceConstraints struct {
	ExactBytes       int      `yaml:"exact_bytes,omitempty"`       // normalized source must be exactly this many bytes (0 = unchecked)
	MaxBytes         int      `yaml:"max_bytes,omitempty"`         // normalized source must be at most this many bytes (0 = unchecked)
	BannedChars      string   `yaml:"banned_chars,omitempty"`      // none of these characters may appear anywhere in the source
	BannedSubstrings []string `yaml:"banned_substrings,omitempty"` // none of these substrings may appear anywhere in the source
}

// NormalizeSource applies the constraint-checking normalization: line endings
// to LF, trailing newlines stripped. Exported so tests and tools can reproduce
// the exact byte count a model is graded against.
func NormalizeSource(code string) string {
	s := strings.ReplaceAll(code, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimRight(s, "\n")
}

// Check returns a list of human-readable violations (empty = compliant).
// Messages are written to double as self-repair feedback: precise, actionable,
// and quantified — the model gets exactly one repair attempt to act on them.
func (sc *SourceConstraints) Check(code string) []string {
	var violations []string
	src := NormalizeSource(code)

	if sc.ExactBytes > 0 && len(src) != sc.ExactBytes {
		violations = append(violations, fmt.Sprintf(
			"source must be EXACTLY %d bytes after normalization (LF line endings, trailing newlines stripped); yours is %d bytes (%+d)",
			sc.ExactBytes, len(src), len(src)-sc.ExactBytes))
	}
	if sc.MaxBytes > 0 && len(src) > sc.MaxBytes {
		violations = append(violations, fmt.Sprintf(
			"source must be at most %d bytes after normalization; yours is %d bytes (%d over)",
			sc.MaxBytes, len(src), len(src)-sc.MaxBytes))
	}
	for _, ch := range sc.BannedChars {
		if idx := strings.IndexRune(src, ch); idx >= 0 {
			line := 1 + strings.Count(src[:idx], "\n")
			violations = append(violations, fmt.Sprintf(
				"source must not contain the character %q anywhere (found on line %d)", ch, line))
		}
	}
	for _, sub := range sc.BannedSubstrings {
		if sub == "" {
			continue
		}
		if idx := strings.Index(src, sub); idx >= 0 {
			line := 1 + strings.Count(src[:idx], "\n")
			violations = append(violations, fmt.Sprintf(
				"source must not contain the substring %q anywhere (found on line %d)", sub, line))
		}
	}
	return violations
}

// Validate rejects meaningless constraint combinations at spec-load time.
func (sc *SourceConstraints) Validate() error {
	if sc.ExactBytes < 0 || sc.MaxBytes < 0 {
		return fmt.Errorf("source_constraints: byte counts must be non-negative")
	}
	if sc.ExactBytes > 0 && sc.MaxBytes > 0 {
		return fmt.Errorf("source_constraints: exact_bytes and max_bytes are mutually exclusive")
	}
	if sc.ExactBytes == 0 && sc.MaxBytes == 0 && sc.BannedChars == "" && len(sc.BannedSubstrings) == 0 {
		return fmt.Errorf("source_constraints: at least one constraint must be set")
	}
	return nil
}
