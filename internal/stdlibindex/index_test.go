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

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
