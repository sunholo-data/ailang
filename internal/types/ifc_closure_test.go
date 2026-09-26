package types

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// M-TAINT-TYPES: a labelled value must not reach a {not secret} sink by being
// routed through a closure.
//
// Before this fix the guarantee was bypassable in three tokens:
//
//	let f = \u. s in sink(f(0))      -- s : string<secret>
//
// Two independent gaps combined. ifc_check.go's Lambda case returned
// LabelBottom() ("a closure value carries no label"), and labelOfCall's fallback
// for a callee with no declared signature — which is the path every local
// closure takes — joined only the ARGUMENT labels, discarding the callee's own.
// Fixing either alone leaves the hole open; the second test below is what caught
// that during development.
//
// Both fixes over-approximate: a closure is labelled with what its body returns,
// whether or not a caller uses the result. For a security control that is the
// correct direction to err — over-approximating rejects safe programs loudly,
// under-approximating admits leaks silently.

func ifcErrsFor(t *testing.T, body string) []*TypeCheckError {
	t.Helper()
	src := "module p\n" +
		"export func sink(body: string{not secret}) -> string ! {} = body\n" +
		"export func attempt(s: string<secret>) -> string ! {} = " + body + "\n"
	p := parser.New(lexer.New(src, "p.ail"))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse %q: %v", body, errs)
	}
	return CheckModuleIFC(prog.File)
}

func hasIFCViolation(errs []*TypeCheckError) bool {
	for _, e := range errs {
		if strings.Contains(e.Error(), "information-flow violation") {
			return true
		}
	}
	return false
}

func TestIFCClosureCannotLaunderALabel(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		wantBlocked bool
		note        string
	}{
		{"direct call is the control", `sink(s)`, true,
			"if this ever passes, the fixture is broken and every row below proves nothing"},
		{"let binding", `let copied = s in sink(copied)`, true, ""},
		{"record field", `let r = { payload: s } in sink(r.payload)`, true, ""},
		{"list element", `let xs = [s] in match xs { [] => "", x :: _ => sink(x) }`, true, ""},
		{"sink inside the lambda", `let f = \u. sink(s) in f(0)`, true,
			"was already caught: the checker walks lambda bodies"},
		{"label returned from a lambda", `let f = \u. s in sink(f(0))`, true,
			"THE THREE-TOKEN BYPASS this fix closes"},
		{"lambda applied inline", `(\u. sink(s))(0)`, true, ""},
		{"nested let over a closure call", `let f = \u. s in let g = f(0) in sink(g)`, true, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := hasIFCViolation(ifcErrsFor(t, tc.body))
			if got != tc.wantBlocked {
				verb := "was not blocked"
				if !tc.wantBlocked {
					verb = "was blocked"
				}
				t.Errorf("%s: a <secret> value %s reaching a {not secret} sink. %s", tc.body, verb, tc.note)
			}
		})
	}
}

// TestIFCAnnotatedLetIsAlsoClosed covers the annotated-binding form, which is a
// separate code path from the bare `let` above.
//
// During development this case appeared to survive the fix. It did not: the
// scratchpad's compile cache was serving a result built before the change. The
// cache key includes the compiler version, which for two builds of the same
// (dirty) tree is identical — so iterating on the checker and re-running through
// the CLI can silently grade a stale binary. Clear .ailang/cache, or use these
// unit tests, when measuring a checker change.
func TestIFCAnnotatedLetIsAlsoClosed(t *testing.T) {
	cases := []struct{ name, body string }{
		{"annotated plain value", `let g: string<secret> = s in sink(g)`},
		{"annotated, unlabelled annotation", `let g: string = s in sink(g)`},
		{"annotated binding of a closure call", `let f = \u. s in let g: string<secret> = f(0) in sink(g)`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !hasIFCViolation(ifcErrsFor(t, tc.body)) {
				t.Errorf("%s: a <secret> value reached a {not secret} sink", tc.body)
			}
		})
	}
}
