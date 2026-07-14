package stdlibindex

import (
	"path/filepath"
	"testing"
)

// TestModules (M-AGENT-ERGONOMICS) — the index resolves a stdlib symbol back to its exporting
// module(s) so "undefined variable" errors can suggest the import. Scans the repo's std/.
func TestModules(t *testing.T) {
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join("..", "..", "std"))

	// nth is a list primitive — std/list must be among its exporters.
	if !contains(Modules("nth"), "std/list") {
		t.Errorf("Modules(\"nth\") = %v, expected to include std/list", Modules("nth"))
	}
	// A name no stdlib module exports yields no suggestion (no false positives).
	if got := Modules("definitely_not_a_stdlib_symbol_xyz"); len(got) != 0 {
		t.Errorf("Modules(unknown) = %v, want empty", got)
	}
}

// TestAllModules (M-DX-AI-DISCOVERY M3) — AllModules lists every std module,
// sorted, with no duplicates. Used by unknown-module recovery.
func TestAllModules(t *testing.T) {
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join("..", "..", "std"))

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
