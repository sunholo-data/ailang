package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// TestCollectPatternVars_ListPatternTail verifies that collectPatternVars
// handles the Tail field of ListPattern (cons patterns like x :: rest).
func TestCollectPatternVars_ListPatternTail(t *testing.T) {
	locals := make(map[string]bool)

	// Simulate the Core AST for: x :: rest => ...
	// Elaborated as: ListPattern{Elements: [VarPattern("x")], Tail: &VarPattern("rest")}
	tailPat := core.CorePattern(&core.VarPattern{Name: "rest"})
	pat := &core.ListPattern{
		Elements: []core.CorePattern{&core.VarPattern{Name: "x"}},
		Tail:     &tailPat,
	}

	collectPatternVars(pat, locals)

	if !locals["x"] {
		t.Error("expected 'x' to be in locals from ListPattern.Elements")
	}
	if !locals["rest"] {
		t.Error("expected 'rest' to be in locals from ListPattern.Tail")
	}
}

// TestFindUnresolvedVars_ConsPatternNotFalsePositive verifies that a match
// expression using cons patterns does not produce false "unresolved" warnings
// for the tail binding.
func TestFindUnresolvedVars_ConsPatternNotFalsePositive(t *testing.T) {
	// Build Core AST equivalent to:
	//   match xs { x :: rest => rest }
	tailPat := core.CorePattern(&core.VarPattern{Name: "rest"})
	matchExpr := &core.Match{
		Scrutinee: &core.Var{Name: "xs"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.ListPattern{
					Elements: []core.CorePattern{&core.VarPattern{Name: "x"}},
					Tail:     &tailPat,
				},
				Body: &core.Var{Name: "rest"}, // references tail binding
			},
		},
	}

	// "xs" is in outer scope
	scope := map[string]bool{"xs": true}
	unresolved := findUnresolvedVars(matchExpr, scope)

	for _, name := range unresolved {
		if name == "rest" {
			t.Error("'rest' should NOT be unresolved — it's bound by the cons pattern tail")
		}
		if name == "x" {
			t.Error("'x' should NOT be unresolved — it's bound by the cons pattern head")
		}
	}
	if len(unresolved) > 0 {
		t.Errorf("expected no unresolved vars, got %v", unresolved)
	}
}
