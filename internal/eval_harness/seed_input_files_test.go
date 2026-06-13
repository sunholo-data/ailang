package eval_harness

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSeedInputFiles_WritesToWorkspaceRoot guards the agent-mode regression where
// spec.InputFiles were created only by the standard runner, never in the agent
// workspace — so file-reading benchmarks (e.g. cli_args reads numbers.txt) looked
// far harder in agent mode than in standard mode because the agent could not test
// its own solution. Files must land at the workspace root (the agent's cwd and the
// validators' cwd), nested paths included.
func TestSeedInputFiles_WritesToWorkspaceRoot(t *testing.T) {
	ws := t.TempDir()
	spec := &BenchmarkSpec{
		InputFiles: map[string]string{
			"numbers.txt":    "1\n2\n3\n4\n5\n",
			"sub/nested.txt": "nested\n",
		},
	}

	if err := seedInputFiles(ws, spec); err != nil {
		t.Fatalf("seedInputFiles returned error: %v", err)
	}

	for name, want := range spec.InputFiles {
		got, err := os.ReadFile(filepath.Join(ws, name))
		if err != nil {
			t.Fatalf("input file %q not seeded: %v", name, err)
		}
		if string(got) != want {
			t.Errorf("input file %q content = %q, want %q", name, got, want)
		}
	}
}

// TestSeedInputFiles_NilAndEmpty verifies the helper is a no-op for specs without
// input files (the common case) and never panics on a nil spec.
func TestSeedInputFiles_NilAndEmpty(t *testing.T) {
	if err := seedInputFiles(t.TempDir(), nil); err != nil {
		t.Errorf("nil spec should be a no-op, got error: %v", err)
	}
	ws := t.TempDir()
	if err := seedInputFiles(ws, &BenchmarkSpec{}); err != nil {
		t.Errorf("empty InputFiles should be a no-op, got error: %v", err)
	}
	entries, _ := os.ReadDir(ws)
	if len(entries) != 0 {
		t.Errorf("expected empty workspace, found %d entries", len(entries))
	}
}
