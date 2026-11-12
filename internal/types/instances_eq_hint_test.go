package types

import (
	"strings"
	"testing"
)

// The missing-Eq hint names the type, where it is declared, and the exact
// edit; for a type of another package it says that package must make it.
func TestEqTConHint(t *testing.T) {
	env := NewInstanceEnv()
	env.RecordTypeDecl("app/policy", "ToolOffer", "app/policy.ail:5", false)
	env.RecordTypeDecl("app/geo", "Pt", "app/geo.ail:3", true)
	cases := []struct {
		name  string
		typ   *TCon
		want  []string
		avoid []string
	}{
		{"local with site", &TCon{Name: "ToolOffer", Module: "app/policy"},
			[]string{"Add `deriving (Eq)` to type ToolOffer", "declared at app/policy.ail:5", "type ToolOffer = A | B deriving (Eq)"}, []string{"requires/ensures"}},
		{"local record", &TCon{Name: "Pt", Module: "app/geo"},
			[]string{"Add `deriving (Eq)` to type Pt", "declared at app/geo.ail:3", "type Pt = {x: int, y: int} deriving (Eq)"}, []string{"A | B"}},
		{"local without site", &TCon{Name: "Grade", Module: "app/types"},
			[]string{"Add `deriving (Eq)` to type Grade", "declared in module app/types"}, nil},
		{"registry package", &TCon{Name: "Plan", Module: "pkg/sunholo/billing/plan"},
			[]string{"package sunholo/billing", "that package must add `deriving (Eq)` to type Plan", "a contract included"}, []string{"Add `deriving"}},
		{"stdlib", &TCon{Name: "TerminalKey", Module: "std/terminal"},
			[]string{"the standard library", "must add `deriving (Eq)`"}, nil},
		{"unit (#2282)", TUnit, []string{"drop the comparison", "does not apply to ()"}, []string{"Add `deriving"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := env.instanceHint("Eq", c.typ)
			for _, w := range c.want {
				if !strings.Contains(h, w) {
					t.Errorf("hint lacks %q: %s", w, h)
				}
			}
			for _, a := range c.avoid {
				if strings.Contains(h, a) {
					t.Errorf("hint must not say %q: %s", a, h)
				}
			}
		})
	}
	// The lookup error carries it.
	_, err := env.Lookup("Eq", &TCon{Name: "ToolOffer", Module: "app/policy"})
	if err == nil || !strings.Contains(err.Error(), "declared at app/policy.ail:5") {
		t.Fatalf("Lookup error = %v", err)
	}
}
