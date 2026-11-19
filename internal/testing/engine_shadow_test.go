package testing

import "testing"

// TestEngineParity_ShadowedLetInTestBody: a named-test body whose block let
// shadows an outer one. Before #2375 both VM engines failed the first two
// (the inner binding overwrote the outer) and --strict-bytecode the third
// (the local captured the later call of the top-level function r).
func TestEngineParity_ShadowedLetInTestBody(t *testing.T) {
	path := writeEngineSource(t, "shadow.ail", `module shadow

func r() -> int = 7

test "block shadow keeps outer" { { let r = 1; let s = { let r = 2; r * 10 }; r * 100 + s } == 120 }

test "closure after block sees outer" { { let r = 1; let s = { let r = 2; r }; let f = \x. x + r; f(0) + s } == 3 }

test "global after shadow" { { let s = { let r = 2; r }; r() + s } == 9 }
`)
	for _, eng := range []struct {
		name             string
		bytecode, strict bool
	}{{"eval", false, false}, {"bytecode", true, false}, {"strict", true, true}} {
		r := runEngine(t, path, eng.bytecode, eng.strict, 0)
		if r.PassedTests != 3 {
			for _, tr := range r.Tests {
				t.Logf("%s: %q %s %s", eng.name, tr.Name, tr.Status, tr.Error)
			}
			t.Errorf("%s: %d of 3 tests passed", eng.name, r.PassedTests)
		}
	}
}
