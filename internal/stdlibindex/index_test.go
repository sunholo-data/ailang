package stdlibindex

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/sunholo-data/ailang/internal/stdlibroot"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// outsideRepo runs the test from an empty temp dir with AILANG_STDLIB_PATH cleared
// and a fresh index, so the index must come from the stdlib built into the binary
// (M-STDLIB-ROOT-RESOLUTION) — exactly where agents run.
func outsideRepo(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("AILANG_STDLIB_PATH", "")
	testutil.SetHomeDir(t, filepath.Join(dir, "home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "xdg"))
	t.Setenv("APPDATA", filepath.Join(dir, "appdata"))
	stdlibroot.Configure(stdlibroot.Options{})
	once, idx = sync.Once{}, nil
	t.Cleanup(func() {
		stdlibroot.Configure(stdlibroot.Options{})
		once, idx = sync.Once{}, nil
	})
}

// TestModules (M-AGENT-ERGONOMICS) — the index resolves a stdlib symbol back to its exporting
// module(s) so "undefined variable" errors can suggest the import.
func TestModules(t *testing.T) {
	outsideRepo(t)

	// nth is a list primitive — std/list must be among its exporters.
	if !contains(Modules("nth"), "std/list") {
		t.Errorf("Modules(\"nth\") = %v, expected to include std/list", Modules("nth"))
	}
	// The Audit's R3 case: `length` got no hint outside the repo.
	if !contains(Modules("length"), "std/list") {
		t.Errorf("Modules(\"length\") = %v outside a repo, expected to include std/list", Modules("length"))
	}
	// A name no stdlib module exports yields no suggestion (no false positives).
	if got := Modules("definitely_not_a_stdlib_symbol_xyz"); len(got) != 0 {
		t.Errorf("Modules(unknown) = %v, want empty", got)
	}
}

// TestAllModules (M-DX-AI-DISCOVERY M3) — AllModules lists every std module,
// sorted, with no duplicates. Used by unknown-module recovery.
func TestAllModules(t *testing.T) {
	outsideRepo(t)

	mods := AllModules()
	if len(mods) == 0 {
		t.Fatal("AllModules() returned empty — stdlib not resolved")
	}
	// Known modules present.
	for _, want := range []string{"std/list", "std/clock", "std/string"} {
		if !contains(mods, want) {
			t.Errorf("AllModules() missing %q; got %v", want, mods)
		}
	}
	// Sorted + de-duplicated.
	for i := 1; i < len(mods); i++ {
		if mods[i-1] >= mods[i] {
			t.Errorf("AllModules() not strictly sorted/unique at %d: %q >= %q", i, mods[i-1], mods[i])
		}
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
