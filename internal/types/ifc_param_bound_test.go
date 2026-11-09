package types_test

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// #2185 review: Checks A and C read only a parameter type's TOP-LEVEL label, so
// `takeOne(x: string<sqlsafe>)` rejected an <arg> value while
// `takeList(xs: [string<sqlsafe>])` accepted `[s]` for the same s. A label or
// refinement anywhere in a covariant data position is a demand
// (ifc_param_bound.go). Cross-module twins: internal/pipeline/ifc_cross_module_test.go.

const ifcBoundDecls = `module app
type Q = { q: string<sqlsafe>, note: string }
type Box[a] = { v: a }
type Safe = Clean(string<sqlsafe>)
pure func src(raw: string) -> string<arg> { raw }
pure func trusted(raw: string) -> string<sqlsafe> { raw }
pure func email(raw: string) -> string<email> { raw }
pure func takeOne(x: string<sqlsafe>) -> int { 0 }
pure func takeList(xs: [string<sqlsafe>]) -> int { 0 }
pure func takeNested(xss: [[string<sqlsafe>]]) -> int { 0 }
pure func takeRec(r: { q: string<sqlsafe> }) -> int { 0 }
pure func takeQ(r: Q) -> int { 0 }
pure func takeTup(t: (string<sqlsafe>, int)) -> int { 0 }
pure func takeOpt(o: Option[string<sqlsafe>]) -> int { 0 }
pure func takeBox(b: Box[string<sqlsafe>]) -> int { 0 }
pure func takeSafe(s: Safe) -> int { 0 }
pure func sinkList(xs: [string{not email}]) -> int { 0 }
pure func takeCb(k: (string<sqlsafe>) -> string) -> int { 0 }
pure func takeTwo(r: { a: string<sqlsafe>, b: string<email> }) -> int { 0 }
pure func noSql(x: string{not sqlsafe}) -> int { 0 }
pure func useNote(r: Q) -> int { noSql(r.note) }
`

func TestIFCParamBoundStructural(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       types.TypeErrorKind // "" = must be accepted
		param      string
	}{
		// The reported asymmetry, both halves.
		{"top-level (control)", `takeOne(src("x"))`, types.ParamLabelCoverError, `"x"`},
		{"list element", `takeList([src("x")])`, types.ParamLabelCoverError, `"xs"`},
		{"nested list", `takeNested([[src("x")]])`, types.ParamLabelCoverError, `"xss"`},
		{"list via let", `{ let ys = [src("x")]; takeList(ys) }`, types.ParamLabelCoverError, `"xs"`},
		{"inline record field", `takeRec({ q: src("x") })`, types.ParamLabelCoverError, `"r"`},
		{"declared record field", `takeQ({ q: src("x"), note: "" })`, types.ParamLabelCoverError, `"r"`},
		{"tuple element", `takeTup((src("x"), 1))`, types.ParamLabelCoverError, `"t"`},
		{"Option argument", `takeOpt(Some(src("x")))`, types.ParamLabelCoverError, `"o"`},
		{"local generic argument", `takeBox({ v: src("x") })`, types.ParamLabelCoverError, `"b"`},
		{"ADT constructor field", `takeSafe(Clean(src("x")))`, types.ParamLabelCoverError, `"s"`},
		{"refinement in list element", `sinkList([email("x")])`, types.SinkRefinementError, `"xs"`},
		{"function value", `{ let f = takeList; f([src("x")]) }`, types.ParamLabelCoverError, `"xs"`},
		// A nested cover bounds the argument as a whole, not per component, so
		// it must not replace the parameter's symbolic label: a covered value in
		// an unlabelled field still reaches the body's sink (topLabel).
		{"covered value in unlabelled field, function", `useNote({ q: "", note: trusted("x") })`, types.SinkRefinementError, `"x"`},
		{"covered value in unlabelled field, literal", `(func(r: Q) -> int { noSql(r.note) })({ q: "", note: trusted("x") })`, types.SinkRefinementError, `"x"`},

		{"list of literals", `takeList(["select 1"])`, "", ""},
		{"list of covered values", `takeList([trusted("x")])`, "", ""},
		{"declared record, covered field", `takeQ({ q: trusted("x"), note: "n" })`, "", ""},
		{"tuple of covered values", `takeTup((trusted("x"), 1))`, "", ""},
		{"Option of covered value", `takeOpt(Some(trusted("x")))`, "", ""},
		{"sibling labels admitted per path", `takeTwo({ a: trusted("x"), b: email("y") })`, "", ""},
		{"unrelated label beside a refinement", `sinkList([src("x")])`, "", ""},
		// Contravariance: a label inside a function-typed parameter's
		// parameter is a demand on the CALLEE's application, never a bound on
		// the caller's function value.
		{"function parameter position", `takeCb(\x. src(x))`, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			errs := types.CheckModuleIFC(parseIFC(t, ifcBoundDecls+"export func main() -> int { "+tc.body+" }\n"))
			if tc.want == "" {
				if len(errs) != 0 {
					t.Fatalf("clean control rejected: %v", errs)
				}
				return
			}
			if len(errs) != 1 || errs[0].Kind != tc.want || !strings.Contains(errs[0].Message, "parameter "+tc.param) {
				t.Fatalf("want one %s on parameter %s, got %v", tc.want, tc.param, errs)
			}
		})
	}
}

// The callee owes a written function type's nested parameter label when it
// applies the function value (contravariant position, checked at the callee).
func TestIFCParamBoundCalleeOwesCallbackList(t *testing.T) {
	errs := types.CheckModuleIFC(parseIFC(t, `module app
pure func apply(k: ([string<sqlsafe>]) -> int, s: string<arg>) -> int { k([s]) }
`))
	if len(errs) != 1 || errs[0].Kind != types.ParamLabelCoverError || !strings.Contains(errs[0].Message, "list[string<sqlsafe>]") {
		t.Fatalf("want the callback's list demand enforced at apply: %v", errs)
	}
}

// The exported summary carries the structural bound, so importers enforce it.
func TestIFCParamBoundExported(t *testing.T) {
	out, err := types.AnalyzeModuleIFC(parseIFC(t, `module lib
export pure func q(xs: [string<sqlsafe>], gs: [(string{not email}, int)], plain: [string]) -> int { 0 }
`), nil)
	if err != nil {
		t.Fatal(err)
	}
	ps := out["q"].Params
	if got := ps[0].Label.Decode(); !types.LabelEqual(got, types.LabelConst("sqlsafe")) {
		t.Errorf("xs exported label = %v, want <sqlsafe>", got)
	}
	if len(ps[1].Forbidden) != 1 || ps[1].Forbidden[0] != "email" {
		t.Errorf("gs exported forbidden = %v, want [email]", ps[1].Forbidden)
	}
	if !types.LabelEqual(ps[2].Label.Decode(), types.LabelBottom()) || len(ps[2].Forbidden) != 0 {
		t.Errorf("plain list exported a demand: %+v", ps[2])
	}
}
