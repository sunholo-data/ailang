package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-TYPE-NAME-SHADOW M3: the compile-cache key must cover every interface a
// module's compile READS. Pre-fix it held only the DIRECT imports' iface
// digests, and an iface digest covers exports and constructors but not type
// alias bodies — so editing an alias two hops away (ta's Inner, reached by tc
// only through tb's `Outer = {items: [Inner]}`) served tc its old verdict.
// Measured 2026-09-26: cached "No errors", fresh "record field 'x' not found".
func TestCacheKey_TransitiveAliasEditInvalidates(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))

	write := func(name, src string) {
		t.Helper()
		if err := os.WriteFile(name, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("ta.ail", "module ta\n\nexport type Inner = {x: int}\n")
	write("tb.ail", `module tb

import ta (Inner)

export type Outer = {items: [Inner]}

export pure func one() -> int { 1 }
`)
	write("tc.ail", `module tc

import tb (Outer, one)

export pure func firstX(o: Outer) -> int {
  match o.items {
    [] => one(),
    i :: _ => i.x
  }
}
`)
	check := func() error {
		_, err := Run(Config{Mode: ModeCheck}, Source{Filename: "tc.ail"})
		return err
	}
	if err := check(); err != nil {
		t.Fatalf("seed compile should pass: %v", err)
	}

	// Inner loses field x. tc's source and tb's exports are unchanged.
	write("ta.ail", "module ta\n\nexport type Inner = {y: int}\n")
	err := check()
	if err == nil {
		t.Fatal("stale cache: tc was served its pre-edit verdict after ta's Inner lost field x")
	}
	if !strings.Contains(err.Error(), "'x'") {
		t.Fatalf("expected the missing-field error for x, got: %v", err)
	}
}
