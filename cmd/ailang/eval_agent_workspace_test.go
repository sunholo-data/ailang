package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Two runs of the same benchmark, model and trial started in the same second must
// still get different directories: the 2026-11-02 sibling kills came from exactly
// that collision (see newAgentSessionWorkspace).
func TestNewAgentSessionWorkspace_UniqueWithinOneSecond(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		dir, err := newAgentSessionWorkspace("cp_60197", "claude-haiku-4-5", "", 1)
		if err != nil {
			t.Fatal(err)
		}
		if seen[dir] {
			t.Fatalf("workspace %s handed out twice", dir)
		}
		seen[dir] = true
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			t.Fatalf("workspace %s not created: %v", dir, err)
		}
	}
}

// Wire-parity attribution matches an arm by "_<model>_" inside ailang_eval/ in the
// prompt the agent was given (internal/eval_analysis/wire_parity.go). Keep that shape.
func TestNewAgentSessionWorkspace_KeepsWireParityShape(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	dir, err := newAgentSessionWorkspace("cp_60149", "claude-haiku-4-5", "tool_aware", 3)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(filepath.Dir(dir)) != "ailang_eval" {
		t.Errorf("workspace %s not under ailang_eval/", dir)
	}
	base := filepath.Base(dir)
	for _, want := range []string{"cp_60149_", "_claude-haiku-4-5_", "_t3_", "_tool_aware_"} {
		if !strings.Contains(base, want) {
			t.Errorf("workspace name %q missing %q", base, want)
		}
	}
}
