package main

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/builtins"
)

// #1551: `builtins list` narrows by --module and --query so a --json listing
// fits under a policy-tool output cap.
func TestFilterBuiltinSpecs(t *testing.T) {
	all := builtins.AllSpecs()
	if len(all) == 0 {
		t.Skip("no builtins registered")
	}

	fs := filterBuiltinSpecs(all, "std/fs", "")
	if len(fs) == 0 || len(fs) >= len(all) {
		t.Fatalf("--module std/fs: %d of %d", len(fs), len(all))
	}
	for name, s := range fs {
		if s.Module != "std/fs" {
			t.Errorf("--module std/fs kept %s from %s", name, s.Module)
		}
	}

	q := filterBuiltinSpecs(all, "", "READFILE")
	if len(q) == 0 || len(q) >= len(all) {
		t.Fatalf("--query readfile: %d of %d", len(q), len(all))
	}
	for name, s := range q {
		desc := ""
		if s.Metadata != nil {
			desc = s.Metadata.Description
		}
		if !strings.Contains(strings.ToLower(name+" "+s.Module+" "+desc), "readfile") {
			t.Errorf("--query readfile kept %s", name)
		}
	}

	if got := filterBuiltinSpecs(all, "", ""); len(got) != len(all) {
		t.Fatalf("no filter must keep everything: %d of %d", len(got), len(all))
	}
	if got := filterBuiltinSpecs(all, "std/no-such-module", ""); len(got) != 0 {
		t.Fatalf("unknown module kept %d", len(got))
	}
}

// Grouped listings take their groups from the global registry; restricting
// them to the filtered specs must drop the names (and empty groups) the
// filter removed instead of indexing a nil spec.
func TestRestrictGroups(t *testing.T) {
	all := builtins.AllSpecs()
	fs := filterBuiltinSpecs(all, "std/fs", "")
	g := restrictGroups(builtins.GroupByModule(), fs)
	if len(g) != 1 || len(g["std/fs"]) != len(fs) {
		t.Fatalf("restricted groups: %v", g)
	}
}
