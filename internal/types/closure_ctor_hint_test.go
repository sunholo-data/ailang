package types

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types/exhaust"
)

func TestClosureCtorCandidatesSortedByModuleThenType(t *testing.T) {
	u := MatchUniverse{
		NominalKey("lib/z", "Z"): {exhaust.CtorSig{Name: "C", Arity: 1}},
		NominalKey("lib/a", "B"): {exhaust.CtorSig{Name: "C", Arity: 0}},
		NominalKey("lib/a", "A"): {exhaust.CtorSig{Name: "C", Arity: 2}, exhaust.CtorSig{Name: "D", Arity: 0}},
	}
	got := ClosureCtorCandidates(u)
	want := []CtorCandidate{{"lib/a", "A", 2}, {"lib/a", "B", 0}, {"lib/z", "Z", 1}}
	if len(got["C"]) != len(want) {
		t.Fatalf("C: got %v, want %v", got["C"], want)
	}
	for i := range want {
		if got["C"][i] != want[i] {
			t.Fatalf("C: got %v, want %v", got["C"], want)
		}
	}
	if len(got["D"]) != 1 || got["D"][0] != (CtorCandidate{"lib/a", "A", 0}) {
		t.Errorf("D: got %v", got["D"])
	}
}

func TestImportCtorHintsListTheTypeAndItsConstructors(t *testing.T) {
	u := MatchUniverse{
		NominalKey("lib/st", "Status"): {exhaust.CtorSig{Name: "Err", Arity: 1}, exhaust.CtorSig{Name: "Ok"}, exhaust.CtorSig{Name: "Pending"}},
		NominalKey("lib/o", "Other"):   {exhaust.CtorSig{Name: "Ok"}},
	}
	one := ImportCtorHint(u, "Ok", CtorCandidate{Module: "lib/st", TypeName: "Status"})
	if one != "if you mean lib/st's Status.Ok, import it: import lib/st (Status, Err, Ok, Pending)" {
		t.Errorf("one hint: %q", one)
	}
	two := ImportCtorHints(u, "Ok", []CtorCandidate{{Module: "lib/o", TypeName: "Other"}, {Module: "lib/st", TypeName: "Status"}})
	if !strings.Contains(two, "import lib/o (Other, Ok) — or if you mean lib/st's Status.Ok") {
		t.Errorf("two hints: %q", two)
	}
	if ImportCtorHints(u, "Ok", nil) != "" {
		t.Error("no candidates must give no hint")
	}
}

// foreignCtorHint picks the closure types named like the scrutinee's type,
// never the type the in-scope constructor belongs to.
func TestForeignCtorHintPicksTheScrutineeType(t *testing.T) {
	tc := NewCoreTypeChecker()
	tc.SetMatchUniverse(MatchUniverse{
		NominalKey("std/option", "Option"): {exhaust.CtorSig{Name: "None"}, exhaust.CtorSig{Name: "Some", Arity: 1}},
		NominalKey("lib/st", "Maybe"):      {exhaust.CtorSig{Name: "Just", Arity: 1}, exhaust.CtorSig{Name: "None"}},
		NominalKey("lib/q", "Other"):       {exhaust.CtorSig{Name: "None"}},
	})
	if h := tc.foreignCtorHint("None", "Option", "Maybe"); h != "if you mean lib/st's Maybe.None, import it: import lib/st (Maybe, Just, None)" {
		t.Errorf("Maybe: %q", h)
	}
	if h := tc.foreignCtorHint("None", "Option", "Unrelated"); h != "" {
		t.Errorf("no closure type named like the scrutinee: want no hint, got %q", h)
	}
	if h := tc.foreignCtorHint("None", "Option", "Option"); h != "" {
		t.Errorf("the in-scope type is never the hint, got %q", h)
	}
}

// #2371: where the prelude is not injected, the six names get the import
// that brings them into scope, without the stdlib index.
func TestImportHintForPreludeNames(t *testing.T) {
	saved := ImportSuggester
	ImportSuggester = nil
	defer func() { ImportSuggester = saved }()
	for name, want := range map[string]string{
		"Some": "import std/option (Option, Some, None)", "None": "import std/option (Option, Some, None)",
		"Option": "import std/option (Option, Some, None)", "Ok": "import std/result (Result, Ok, Err)",
		"Err": "import std/result (Result, Ok, Err)", "Result": "import std/result (Result, Ok, Err)",
	} {
		if got := importHint(name); got != " — add `"+want+"`" {
			t.Errorf("%s: %q", name, got)
		}
	}
	if got := importHint("Just"); got != "" {
		t.Errorf("a non-prelude name with no index: want no hint, got %q", got)
	}
}
