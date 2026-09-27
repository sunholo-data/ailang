package loader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// M-V1-SIMPLIFY-S4 M1: resolvePath used to carry its own stdlib lookup —
// AILANG_STDLIB_PATH else "." — so with the variable unset a std/ path was
// `./std/<module>.ail` relative to whatever cwd the process had: a stale cwd
// meant the wrong stdlib, silently. It now goes through the same
// StdlibResolver Load uses (then the embedded copy), and an unresolvable
// stdlib module is an ERROR, never a cwd-relative guess.
func TestResolvePath_StdGoesThroughTheResolverNeverCwd(t *testing.T) {
	testutil.SetHomeDir(t, t.TempDir()) // no ~/.ailang/std on this run
	stdDir := t.TempDir()
	want := filepath.Join(stdDir, "zz_probe.ail")
	writeStdFile(t, stdDir, "io.ail", "module std/io\n") // the stdlib-root marker
	if err := os.WriteFile(want, []byte("module std/zz_probe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", stdDir)

	ml := NewModuleLoader(t.TempDir())
	got, err := ml.resolvePath("std/zz_probe")
	if err != nil || got != want {
		t.Fatalf("resolvePath(std/zz_probe) = (%q, %v), want %q via AILANG_STDLIB_PATH", got, err, want)
	}
	if strings.HasPrefix(got, "std/") || strings.HasPrefix(got, "./") {
		t.Fatalf("resolved path %q is cwd-relative — the fallback this test retires", got)
	}

	// A module the chosen root lacks is an error, not "./std/…" and not the
	// embedded copy: the resolver's error names the root.
	got, err = ml.resolvePath("std/zz_nowhere")
	if err == nil || got != "" {
		t.Fatalf("resolvePath(std/zz_nowhere) = (%q, %v), want an error", got, err)
	}
	if !strings.Contains(err.Error(), stdDir) {
		t.Fatalf("the error must name the paths tried (expected %s in %q)", stdDir, err)
	}

	// Embedded stdlib modules resolve even with no std/ on disk.
	isolateStdlib(t)
	ml = NewModuleLoader(t.TempDir())
	got, err = ml.resolvePath("std/io")
	if err != nil || !strings.HasPrefix(got, "<embedded>/std/") {
		t.Fatalf("resolvePath(std/io) with no std on disk = (%q, %v), want the embedded copy", got, err)
	}

	// CanonicalPath propagates the error instead of canonicalising a guess.
	if p, err := ml.CanonicalPath("std/zz_nowhere"); err == nil || p != "" {
		t.Fatalf("CanonicalPath(std/zz_nowhere) = (%q, %v), want an error", p, err)
	}
}
