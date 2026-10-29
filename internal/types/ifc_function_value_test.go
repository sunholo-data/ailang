package types

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// M-IFC-FUNCTION-VALUE-SINKS (#1719): a sink's argument requirements travel
// with the FUNCTION VALUE, not with the syntactic call `name(args)`.
//
// Before the fix, Checks A ({not ℓ}) and C (positive <ℓ> coverage) fired only
// when the callee was written as the bare name of a module function. Any other
// route to the same function — a let alias, a higher-order parameter, a record
// field, a list or tuple element, an if branch, a returned function, a
// function literal, or a wrapper lambda — took the label-only fallback and
// checked nothing. Every row below was measured leaking on 772703831 (the
// R/C/N numbering is the design doc's table).

// fvPrelude is written into every row: a source, a refinement sink and a
// positive-label sink.
const fvPrelude = `pure func fromInbox(s: string) -> string<email> { s }
pure func gate(body: string{not email}) -> string { body }
pure func store(q: string<sqlsafe>) -> string { q }
`

const (
	fvApply1  = "pure func apply1(f: string -> string, x: string) -> string { f(x) }\n"
	fvUpper   = "pure func upper(s: string) -> string { s }\n"
	fvSanitze = "pure func sanitize(raw: string<email>) -> string<sanitized> ! {Declassify} { \"[sanitized]\" }\n"
)

type fvRow struct {
	id, name string
	decls    string        // extra declarations
	main     string        // main's body
	mainEff  string        // main's effect row, "" for pure
	want     TypeErrorKind // "" means: must be clean
	direct   bool          // a direct call: message must be the pre-fix wording
}

var fvRows = []fvRow{
	// Controls first: if these ever change, the fixture is broken and every
	// other row proves nothing.
	{id: "R0", name: "direct call (control)", main: `gate(fromInbox("hi"))`, want: SinkRefinementError, direct: true},
	{id: "C0", name: "check C direct (control)", main: `store(fromInbox("hi"))`, want: ParamLabelCoverError, direct: true},

	{id: "R1", name: "let alias", main: `let g = gate in g(fromInbox("hi"))`, want: SinkRefinementError},
	{id: "R2", name: "HOF param", decls: fvApply1, main: `apply1(gate, fromInbox("hi"))`, want: SinkRefinementError},
	{id: "R3", name: "annotated let", main: `let g: string{not email} -> string = gate in g(fromInbox("hi"))`, want: SinkRefinementError},
	{id: "R4", name: "if selection", main: `let g = if true then gate else gate in g(fromInbox("hi"))`, want: SinkRefinementError},
	{id: "R5", name: "record field", main: `let r = { f: gate } in r.f(fromInbox("hi"))`, want: SinkRefinementError},
	{id: "R6", name: "list element", main: `match [gate] { f :: _ => f(fromInbox("hi")), [] => "" }`, want: SinkRefinementError},
	{id: "R7", name: "tuple element", main: `match (gate, 1) { (f, _) => f(fromInbox("hi")) }`, want: SinkRefinementError},
	{id: "R8", name: "returned function", decls: "pure func getGate() -> (string -> string) { gate }\n",
		main: `getGate()(fromInbox("hi"))`, want: SinkRefinementError},
	{id: "R9", name: "composed closure",
		decls: "pure func compose(f: string -> string, g: string -> string) -> (string -> string) { \\x. f(g(x)) }\n" +
			"pure func ident(s: string) -> string { s }\n",
		main: `compose(gate, ident)(fromInbox("hi"))`, want: SinkRefinementError},
	{id: "R10", name: "HOF applies the sink to its own tainted data",
		decls: "pure func h(f: string -> string, x: string) -> string { let _ = f(fromInbox(\"x\")) in x }\n",
		main:  `h(gate, "clean")`, want: SinkRefinementError},
	{id: "R11", name: "HOF with a guarded param type",
		decls: "pure func apply2(f: string{not email} -> string, x: string) -> string { f(x) }\n",
		main:  `apply2(gate, fromInbox("hi"))`, want: SinkRefinementError},
	{id: "R12", name: "named wrapper, unannotated param", decls: "pure func wrap(y: string) -> string { gate(y) }\n",
		main: `wrap(fromInbox("hi"))`, want: SinkRefinementError},
	{id: "R13", name: "lambda wrapper", main: `let w = \y. gate(y) in w(fromInbox("hi"))`, want: SinkRefinementError},
	{id: "R14", name: "function-literal sink", main: `let w = func(y: string{not email}) -> string { y } in w(fromInbox("hi"))`, want: SinkRefinementError},
	{id: "C1", name: "check C via alias", main: `let s = store in s(fromInbox("hi"))`, want: ParamLabelCoverError},
	{id: "C2", name: "check C via HOF", decls: fvApply1, main: `apply1(store, fromInbox("hi"))`, want: ParamLabelCoverError},

	// Negative controls: must stay clean.
	{id: "N1", name: "direct, clean", main: `gate("clean")`},
	{id: "N2", name: "alias + HOF, clean", decls: fvApply1, main: `let g = gate in g(apply1(gate, "clean"))`},
	{id: "N3", name: "non-sink alias to HOF with tainted data", decls: fvApply1 + fvUpper,
		main: `let u = upper in apply1(u, fromInbox("hi"))`},
	{id: "N4", name: "a parameter named gate shadows the sink",
		decls: fvUpper + "pure func k(gate: string -> string, x: string) -> string { gate(x) }\n",
		main:  `k(upper, fromInbox("hi"))`},
	{id: "N5", name: "declassified before the aliased sink", decls: fvSanitze,
		main: `let g = gate in g(sanitize(fromInbox("hi")))`, mainEff: " ! {Declassify}"},
	{id: "N6", name: "sink stored, never applied", main: `let r = { f: gate, n: 1 } in "ok"`},
	{id: "X1", name: "wrapper with multiple forbids", decls: "pure func noSecret(s: string{not secret}) -> string { s }\npure func wrap(y: string) -> string { let _ = gate(y); noSecret(y) }\n", main: `wrap(fromInbox("hi"))`, want: SinkRefinementError},
	{id: "R11b", name: "written HOF guard with unrestricted actual callback", decls: fvUpper + "pure func guarded(f: string{not email} -> string, x: string) -> string { f(x) }\n", main: `guarded(upper, fromInbox("hi"))`, want: SinkRefinementError},
}

func fvSource(r fvRow) string {
	return "module r/fv\n" + fvPrelude + r.decls +
		"export func main() -> string" + r.mainEff + " { " + r.main + " }\n"
}

func fvCheck(t *testing.T, src string) []*TypeCheckError {
	t.Helper()
	p := parser.New(lexer.New(src, "fv.ail"))
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors: %v\n%s", errs, src)
	}
	return CheckModuleIFC(file)
}

func TestIFCFunctionValueSinks(t *testing.T) {
	for _, r := range fvRows {
		t.Run(r.id+"_"+r.name, func(t *testing.T) {
			errs := fvCheck(t, fvSource(r))
			for _, e := range errs {
				if strings.Contains(e.Message, "ρ") {
					t.Errorf("label variable leaked into a diagnostic: %s", e.Message)
				}
			}
			if r.want == "" {
				if len(errs) != 0 {
					t.Fatalf("%s must stay clean, got %v", r.id, errs)
				}
				return
			}
			if len(errs) == 0 {
				t.Fatalf("%s: `%s` leaked: no %s diagnostic", r.id, r.main, r.want)
			}
			for _, e := range errs {
				if e.Kind != r.want {
					t.Fatalf("%s: want only %s, got %s: %s", r.id, r.want, e.Kind, e.Message)
				}
			}
			msg := errs[0].Message
			if r.direct {
				if strings.Contains(msg, " via ") || strings.Contains(msg, " of gate") || strings.Contains(msg, " of store") {
					t.Errorf("%s: direct-call wording changed: %s", r.id, msg)
				}
				return
			}
			if !strings.Contains(msg, "via") {
				t.Errorf("%s: message does not name the path the value took: %s", r.id, msg)
			}
		})
	}
}

// TestIFCFunctionValueDirectMessageUnchanged pins the exact pre-fix wording of
// the direct-call diagnostics (leak_attempt.ail is an expected-fail example
// whose message must not move).
func TestIFCFunctionValueDirectMessageUnchanged(t *testing.T) {
	errs := fvCheck(t, fvSource(fvRows[0]))
	want := `information-flow violation: value labelled <email> reaches parameter "body" which forbids it (string{not email})`
	if len(errs) != 1 || errs[0].Message != want || errs[0].Position != "fv.ail:5:36" {
		t.Fatalf("direct sink message changed:\n got %v\nwant %s", errs, want)
	}
	errs = fvCheck(t, fvSource(fvRows[1]))
	want = `information-flow violation: value labelled <email> reaches parameter "q" declared string<sqlsafe>, which does not cover it`
	if len(errs) != 1 || errs[0].Message != want || errs[0].Position != "fv.ail:5:37" {
		t.Fatalf("direct cover message changed:\n got %v\nwant %s", errs, want)
	}
}

// TestIFCFunctionValueMessageNamesCallee: the via trail names the callee and at
// least the first hop.
func TestIFCFunctionValueMessageNamesCallee(t *testing.T) {
	for _, tc := range []struct{ id, want string }{
		{"R1", "gate"}, {"R2", "apply1"}, {"R12", "wrap"}, {"C1", "store"},
	} {
		for _, r := range fvRows {
			if r.id != tc.id {
				continue
			}
			errs := fvCheck(t, fvSource(r))
			if len(errs) == 0 || !strings.Contains(errs[0].Message, tc.want) {
				t.Errorf("%s: message should name %q: %v", tc.id, tc.want, errs)
			}
		}
	}
}

// Wrappers forwarding to wrappers converge, including mutual recursion, and
// the inferred demand is reported at the call that supplies the label.
func TestIFCFunctionValueMutualRecursionConverges(t *testing.T) {
	src := "module r/mut\n" + fvPrelude + `
pure func ping(n: int, y: string) -> string { if n <= 0 then gate(y) else pong(n - 1, y) }
pure func pong(n: int, y: string) -> string { if n <= 0 then y else ping(n - 1, y) }
export func main() -> string { pong(3, fromInbox("hi")) }
export func clean() -> string { pong(3, "clean") }
`
	errs := fvCheck(t, src)
	if len(errs) != 1 || errs[0].Kind != SinkRefinementError {
		t.Fatalf("want exactly one sink error at main's call, got %v", errs)
	}
}

// Deeper shapes the design's audit implies but does not number: the same
// mechanism must cover them, or the fix is bypassable one hop further out.
func TestIFCFunctionValueDeeperShapes(t *testing.T) {
	for _, tc := range []struct {
		name, decls, main string
		leak              bool
	}{
		{"HOF of HOF", fvApply1 + "pure func twice(f: string -> string, x: string) -> string { apply1(f, x) }\n",
			`twice(gate, fromInbox("hi"))`, true},
		{"lambda passed to HOF", fvApply1, `apply1(\y. gate(y), fromInbox("hi"))`, true},
		{"wrapper of wrapper", "pure func w1(y: string) -> string { gate(y) }\npure func w2(z: string) -> string { w1(z) }\n",
			`w2(fromInbox("hi"))`, true},
		{"wrapper as value", fvApply1 + "pure func wrap(y: string) -> string { gate(y) }\n",
			`apply1(wrap, fromInbox("hi"))`, true},
		{"closure applies captured HOF param", "pure func h(f: string -> string, x: string) -> string { let w = \\y. f(y) in w(x) }\n",
			`h(gate, fromInbox("hi"))`, true},
		{"block let alias", "", `{ let g = gate; g(fromInbox("hi")) }`, true},
		{"clean wrapper", "pure func wrap(y: string) -> string { gate(y) }\n", `wrap("clean")`, false},
		{"HOF of HOF clean", fvApply1 + "pure func twice(f: string -> string, x: string) -> string { apply1(f, x) }\n",
			`twice(gate, "clean")`, false},
		{"composed clean", "pure func compose(f: string -> string, g: string -> string) -> (string -> string) { \\x. f(g(x)) }\n",
			`compose(gate, gate)("clean")`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			errs := fvCheck(t, "module r/deep\n"+fvPrelude+tc.decls+"export func main() -> string { "+tc.main+" }\n")
			if tc.leak && len(errs) == 0 {
				t.Fatalf("`%s` leaked", tc.main)
			}
			if !tc.leak && len(errs) != 0 {
				t.Fatalf("`%s` must stay clean: %v", tc.main, errs)
			}
		})
	}
}

// No label variable may escape the summary pass: not into a diagnostic, not
// into a memoised effective body label.
func TestIFCFunctionValueNoLabelVarEscapes(t *testing.T) {
	src := "module r/esc\n" + fvPrelude + fvApply1 + `
pure func wrap(y: string) -> string { gate(y) }
pure func fwd(y: string) -> string { apply1(upper, y) }
pure func upper(s: string) -> string { s }
pure func compose(f: string -> string, g: string -> string) -> (string -> string) { \x. f(g(x)) }
export func main() -> string { let a = wrap(fromInbox("hi")) in let b = fwd(fromInbox("x")) in compose(gate, upper)(b) }
`
	p := parser.New(lexer.New(src, "esc.ail"))
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatal(errs)
	}
	c := newIFCChecker(file)
	errs := c.run(file)
	if len(errs) == 0 {
		t.Fatal("fixture should produce diagnostics")
	}
	for _, e := range errs {
		if strings.Contains(e.Message, "ρ") {
			t.Errorf("label variable in diagnostic: %s", e.Message)
		}
	}
	for name, l := range c.effLabel {
		if hasLabelVar(l) {
			t.Errorf("effectiveBodyLabel(%s) carries a label variable: %s", name, l)
		}
	}
}

// The join of two callables is at least as strict as either side: anything
// rejected through one alternative is rejected through the join.
func TestIFCCallableJoinIsMonotone(t *testing.T) {
	g := &ifcCallable{alts: []*ifcFn{{kind: fnDeclared, name: "g", params: []ifcDemand{{param: "a", forbid: []string{"email"}}}}}}
	s := &ifcCallable{alts: []*ifcFn{{kind: fnDeclared, name: "s", params: []ifcDemand{{param: "b", covers: []Label{LabelConst("sqlsafe")}}}}}}
	j := joinCallable(g, s)
	if joinCallable(j, nil) != j || joinCallable(nil, j) != j {
		t.Error("nil must be the identity of the join")
	}
	if len(j.alts) != 2 || len(joinCallable(j, g).alts) != 2 {
		t.Errorf("join must union alternatives without duplicates: %d", len(j.alts))
	}
	for _, d := range []struct {
		arg  Label
		fail bool
	}{{LabelConst("email"), true}, {LabelConst("secret"), true}, {LabelConst("sqlsafe"), false}, {LabelBottom(), false}} {
		got := demandViolations(j, d.arg)
		if (got > 0) != d.fail {
			t.Errorf("join on %s: violations=%d, want fail=%v", d.arg, got, d.fail)
		}
		if demandViolations(g, d.arg) > got || demandViolations(s, d.arg) > got {
			t.Errorf("join on %s is less strict than a side", d.arg)
		}
	}
}

func demandViolations(c *ifcCallable, arg Label) int {
	n := 0
	for _, a := range c.alts {
		for _, d := range a.params {
			if d.violation(arg) != "" {
				n++
			}
		}
	}
	return n
}
