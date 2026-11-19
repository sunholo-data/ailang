package importhint

import (
	"strings"
	"testing"
)

// After an alias is removed, importing the old spelling names the replacement
// (M-V1-STDLIB-SURFACE-POLISH §6.3) — including the renames normalisation
// cannot find (readFileBytes -> readFileBase64, member -> contains).
func TestIMP010NamesRenamedStdFunctions(t *testing.T) {
	cases := []struct{ sym, mod, want string }{
		{"nth_or", "std/list", "use 'nthOr' from 'std/list'"},
		{"readFileBytes", "std/fs", "use 'readFileBase64' from 'std/fs'"},
		{"member", "std/list", "use 'contains' from 'std/list' (arguments swap"},
		{"slice", "std/bytes", "use 'sliceLen' from 'std/bytes' (the third argument is a length"},
		{"glob", "std/fs", "use 'walkWithSuffix'"},
	}
	for _, c := range cases {
		if got := IMP010(c.sym, c.mod); !strings.Contains(got, c.want) {
			t.Errorf("IMP010(%s, %s) = %q, want it to contain %q", c.sym, c.mod, got, c.want)
		}
	}
	// A same-named export of another module is not a rename.
	if got := RenameHint("member", "std/map"); got != "" {
		t.Errorf("std/map.member is not renamed, got %q", got)
	}
}

// Normalisation (case and underscores) outranks a prefix match: with nth and
// nthOr both exported, nth_or must suggest nthOr, not nth.
func TestClosestExportPrefersNormalisedName(t *testing.T) {
	exports := []string{"nth", "nthOr", "head", "headOr", "last"}
	for want, sym := range map[string]string{"nthOr": "nth_or", "headOr": "HEAD_OR"} {
		if got := closestExport(sym, exports); got != want {
			t.Errorf("closestExport(%q) = %q, want %q", sym, got, want)
		}
	}
	// The ordinary signals still work when nothing normalises.
	if got := closestExport("flushStdout", []string{"flush", "print"}); got != "flush" {
		t.Errorf("prefix match broken: %q", got)
	}
}
