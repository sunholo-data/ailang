package runner

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
)

func TestBC001Report(t *testing.T) {
	err := BC001Report(bytecode.EvalOnlyEntry{Name: "m.wide", File: "m.ail", Line: 3, Reason: "register allocator: contiguous block of 300 would exceed 256"})
	rep, ok := ailerrors.AsReport(err)
	if !ok {
		t.Fatalf("BC001Report is not a structured report: %v", err)
	}
	if rep.Code != ailerrors.BC001 || rep.Phase != "bytecode" {
		t.Errorf("code/phase = %s/%s, want BC001/bytecode", rep.Code, rep.Phase)
	}
	if rep.Span == nil || rep.Span.Start.File != "m.ail" || rep.Span.Start.Line != 3 {
		t.Errorf("span = %+v, want m.ail:3", rep.Span)
	}
	for _, part := range []string{"BC001: function m.wide is evaluator-only under --strict-bytecode", "contiguous block of 300", "at m.ail:3"} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("message %q lacks %q", err.Error(), part)
		}
	}
	if rep.Data["function"] != "m.wide" {
		t.Errorf("data.function = %v", rep.Data["function"])
	}

	// No position known: no span, no dangling " at ".
	bare := BC001Report(bytecode.EvalOnlyEntry{Name: "f", Reason: "r"})
	if rep, _ := ailerrors.AsReport(bare); rep.Span != nil || strings.Contains(bare.Error(), " at ") {
		t.Errorf("positionless entry rendered a location: %v (span %+v)", bare, rep.Span)
	}
}

func TestStrictEvalOnlyErrorListsEveryEntry(t *testing.T) {
	entries := []bytecode.EvalOnlyEntry{{Name: "a", Reason: "x"}, {Name: "b", Reason: "y", File: "f.ail", Line: 2}}
	msg := StrictEvalOnlyError("main", entries).Error()
	if !strings.Contains(msg, "2 evaluator-only function(s) reachable from main") || strings.Count(msg, "BC001: function ") != 2 {
		t.Errorf("strict error does not name both functions:\n%s", msg)
	}
}

// The non-strict warning always states the exact total, lists at most ten
// names, and is silent for an empty report.
func TestWarnEvalOnly(t *testing.T) {
	var empty bytes.Buffer
	WarnEvalOnly(&empty, "main", nil)
	if empty.Len() != 0 {
		t.Errorf("empty report printed %q", empty.String())
	}

	entries := make([]bytecode.EvalOnlyEntry, 13)
	for i := range entries {
		entries[i] = bytecode.EvalOnlyEntry{Name: fmt.Sprintf("m.f%02d", i), Reason: "why", File: "m.ail", Line: i + 1}
	}
	var b bytes.Buffer
	WarnEvalOnly(&b, "main", entries)
	out := b.String()
	if !strings.Contains(out, "13 evaluator-only function(s) reachable from main") {
		t.Errorf("warning lacks the exact total:\n%s", out)
	}
	if !strings.Contains(out, "m.f09 at m.ail:10: why") || strings.Contains(out, "m.f10") {
		t.Errorf("warning must list exactly the first ten names:\n%s", out)
	}
	if !strings.Contains(out, "... and 3 more") {
		t.Errorf("warning lacks the remainder count:\n%s", out)
	}
}
