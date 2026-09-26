package eval_harness

import (
	"os"
	"path/filepath"
	"testing"
)

// The harness only pins --stdlib-path to a directory that IS a stdlib: an explicit
// override that resolves nowhere is an error in the child (M-STDLIB-ROOT-RESOLUTION).
func TestStdlibPathArgs_OnlyForARealStdlib(t *testing.T) {
	dir := t.TempDir()
	if got := stdlibPathArgs(dir); got != nil {
		t.Fatalf("no std/: args = %v, want none (the child uses its built-in stdlib)", got)
	}
	stdDir := filepath.Join(dir, "std")
	if err := os.MkdirAll(stdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := stdlibPathArgs(dir); got != nil {
		t.Fatalf("std/ without io.ail: args = %v, want none", got)
	}
	if err := os.WriteFile(filepath.Join(stdDir, "io.ail"), []byte("module std/io\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := stdlibPathArgs(dir)
	if len(got) != 2 || got[0] != "--stdlib-path" || got[1] != stdDir {
		t.Fatalf("args = %v, want [--stdlib-path %s]", got, stdDir)
	}
}
