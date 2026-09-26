package executor

import (
	"os"
	"path/filepath"
	"testing"
)

// The executor only exports AILANG_STDLIB_PATH for a directory that IS a stdlib:
// an explicit one that holds none is an error in the child (M-STDLIB-ROOT-RESOLUTION).
func TestChildStdlibPath_OnlyARealStdlib(t *testing.T) {
	t.Chdir(t.TempDir()) // no ./std here
	ws := t.TempDir()
	opts := EnvironmentOptions{Task: &Task{Workspace: ws}}

	if got := childStdlibPath(opts); got != "" {
		t.Fatalf("no std anywhere: got %q, want none", got)
	}
	stdDir := filepath.Join(ws, "std")
	if err := os.MkdirAll(stdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := childStdlibPath(opts); got != "" {
		t.Fatalf("workspace std/ without io.ail: got %q, want none", got)
	}
	if err := os.WriteFile(filepath.Join(stdDir, "io.ail"), []byte("module std/io\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := childStdlibPath(opts); got != stdDir {
		t.Fatalf("got %q, want the workspace stdlib %q", got, stdDir)
	}
}
