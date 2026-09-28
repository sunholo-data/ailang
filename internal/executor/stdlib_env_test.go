package executor

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	ailangstd "github.com/sunholo-data/ailang/std"
)

// copyEmbeddedStdlib writes the stdlib built into this binary to dir.
func copyEmbeddedStdlib(t *testing.T, dir string) {
	t.Helper()
	err := fs.WalkDir(ailangstd.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, rerr := ailangstd.FS.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		dst := filepath.Join(dir, filepath.FromSlash(p))
		if merr := os.MkdirAll(filepath.Dir(dst), 0o755); merr != nil {
			return merr
		}
		return os.WriteFile(dst, b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// The executor exports AILANG_STDLIB_PATH only for a directory that IS a stdlib
// (M-STDLIB-ROOT-RESOLUTION) AND is the stdlib this binary was built with. A
// checkout's live std/ that differs — mid-edit, or a release ahead of the
// installed binary — is not exported, so agents use the binary's own stdlib
// instead of crashing before step 0 on builtins the binary lacks.
func TestChildStdlibPath_OnlyTheBinarysOwnStdlib(t *testing.T) {
	t.Chdir(t.TempDir()) // no ./std here
	ws := t.TempDir()
	opts := EnvironmentOptions{Task: &Task{Workspace: ws}}
	stdDir := filepath.Join(ws, "std")

	if got := childStdlibPath(opts); got != "" {
		t.Fatalf("no std anywhere: got %q, want none", got)
	}
	if err := os.MkdirAll(stdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := childStdlibPath(opts); got != "" {
		t.Fatalf("workspace std/ without io.ail: got %q, want none", got)
	}
	if err := os.WriteFile(filepath.Join(stdDir, "io.ail"), []byte("module std/io\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := childStdlibPath(opts); got != "" {
		t.Fatalf("a stub stdlib that is not the binary's: got %q, want none", got)
	}

	copyEmbeddedStdlib(t, stdDir)
	if got := childStdlibPath(opts); got != stdDir {
		t.Fatalf("an exact copy of the binary's stdlib: got %q, want %q", got, stdDir)
	}

	// An edited module is a stdlib the binary was not built with.
	json := filepath.Join(stdDir, "json.ail")
	orig, err := os.ReadFile(json)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(json, append(orig, []byte("\n-- edited\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := childStdlibPath(opts); got != "" {
		t.Fatalf("an edited module: got %q, want none", got)
	}
	if err := os.WriteFile(json, orig, 0o644); err != nil {
		t.Fatal(err)
	}

	// So is a module the binary does not carry.
	if err := os.WriteFile(filepath.Join(stdDir, "newmod.ail"), []byte("module std/newmod\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := childStdlibPath(opts); got != "" {
		t.Fatalf("an extra module: got %q, want none", got)
	}
}
