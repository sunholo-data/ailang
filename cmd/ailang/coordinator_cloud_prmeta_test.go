package main

import (
	"strings"
	"testing"
)

// MU: return "" unconditionally and the first case fails.
func TestMergeWarning(t *testing.T) {
	w := mergeWarning([]string{"sprint-executor"})
	if !strings.Contains(w, "starts **sprint-executor**") || !strings.HasSuffix(w, "\n\n") {
		t.Errorf("warning = %q, want it to name sprint-executor and end its own paragraph", w)
	}
	if got := mergeWarning(nil); got != "" {
		t.Errorf("no handoff produced a warning: %q", got)
	}
}
