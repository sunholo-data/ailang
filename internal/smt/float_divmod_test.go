package smt

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// #2062 (M-SMT-FLOAT-DIV-MOD-SORT-DISPATCH): Float `/` and `%` were encoded as
// truncating Int division — Int-only `div`/`abs` and `(>= n 0)` applied to Real
// terms. Z3 rejected the script or coerced it with to_int and refuted contracts
// that hold at runtime. These tests pin the Real encodings against Go's own
// float64 `/` and math.Mod, which is what both AILANG engines evaluate.

func floatLit(v float64) core.CoreExpr { return &core.Lit{Kind: core.FloatLit, Value: v} }

// floatDivModEmitters is every Core shape that reaches the encoder as a Float
// division or remainder. DictApp{Fractional Float, div} is the normal
// post-pipeline shape for `/`; App(mod_Float) is the normal shape for `%`.
var floatDivModEmitters = map[string]struct {
	isMod bool
	build func(a, b core.CoreExpr) core.CoreExpr
}{
	"builtin div_Float": {false, func(a, b core.CoreExpr) core.CoreExpr { return builtinApp("div_Float", a, b) }},
	"builtin mod_Float": {true, func(a, b core.CoreExpr) core.CoreExpr { return builtinApp("mod_Float", a, b) }},
	"curried mod_Float": {true, func(a, b core.CoreExpr) core.CoreExpr {
		return &core.App{Func: &core.App{Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "mod_Float"}}, Args: []core.CoreExpr{a}}, Args: []core.CoreExpr{b}}
	}},
	"intrinsic OpDiv": {false, func(a, b core.CoreExpr) core.CoreExpr {
		return &core.Intrinsic{Op: core.OpDiv, Args: []core.CoreExpr{a, b}}
	}},
	"intrinsic OpMod": {true, func(a, b core.CoreExpr) core.CoreExpr {
		return &core.Intrinsic{Op: core.OpMod, Args: []core.CoreExpr{a, b}}
	}},
	"binop /": {false, func(a, b core.CoreExpr) core.CoreExpr { return &core.BinOp{Op: "/", Left: a, Right: b} }},
	"binop %": {true, func(a, b core.CoreExpr) core.CoreExpr { return &core.BinOp{Op: "%", Left: a, Right: b} }},
	"dictapp Fractional div": {false, func(a, b core.CoreExpr) core.CoreExpr {
		return &core.DictApp{Dict: &core.DictRef{ClassName: "Fractional", TypeName: "Float"}, Method: "div", Args: []core.CoreExpr{a, b}}
	}},
	"dictapp mod": {true, func(a, b core.CoreExpr) core.CoreExpr {
		return &core.DictApp{Dict: &core.DictRef{ClassName: "Num", TypeName: "Float"}, Method: "mod", Args: []core.CoreExpr{a, b}}
	}},
}

func smtRealLit(t *testing.T, v float64) string {
	t.Helper()
	s, err := encodeFloatLit(v)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// TestFloatDivModEncoding_MatchesGoFloat asks Z3 to refute Go's float64 result
// for every sign combination and emitter. Operand pairs are dyadic so the
// float64 result is exact and equals the Real one.
func TestFloatDivModEncoding_MatchesGoFloat(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}
	pairs := [][2]float64{{7.5, 2}, {-7.5, 2}, {7.5, -2}, {-7.5, -2}, {7, 2}, {-7, 2}, {0.5, 0.25}, {6, 3}, {-6, 3}, {0, -5}, {1.5, 0.5}, {-0.75, 4}, {5.25, -0.5}}
	for name, em := range floatDivModEmitters {
		for _, p := range pairs {
			a, b := p[0], p[1]
			want := a / b
			if em.isMod {
				want = math.Mod(a, b)
			}
			enc, err := EncodeExpr(em.build(floatLit(a), floatLit(b)))
			if err != nil {
				t.Fatalf("%s(%v, %v): encode: %v", name, a, b, err)
			}
			script := fmt.Sprintf("(set-logic ALL)\n(assert (not (= %s %s)))\n(check-sat)\n", enc, smtRealLit(t, want))
			res, err := Solve(script, DefaultSolverConfig())
			if err != nil {
				t.Fatalf("%s(%v, %v): solve: %v\n%s", name, a, b, err, script)
			}
			if res.Status != StatusVerified {
				t.Errorf("%s(%v, %v): encoding %s does not equal Go's %v (Z3 %v)", name, a, b, enc, want, res.Status)
			}
		}
	}
}

// TestFloatDivModEncoding_NoIntOperators pins the shape: no Int-only operator,
// no Int numeral compared against a Real, operands encoded once.
func TestFloatDivModEncoding_NoIntOperators(t *testing.T) {
	x := &core.Var{Name: "x"}
	for name, em := range floatDivModEmitters {
		enc, err := EncodeExpr(em.build(x, floatLit(2)))
		if err != nil {
			t.Fatalf("%s: encode: %v", name, err)
		}
		for _, bad := range []string{"(div ", "(mod ", "(abs ", " 0)", " 0 "} {
			if strings.Contains(enc, bad) {
				t.Errorf("%s: emitted Int-sorted %q on Reals: %s", name, bad, enc)
			}
		}
		if strings.Count(enc, "x") != 1 {
			t.Errorf("%s: want x encoded once, got %s", name, enc)
		}
	}
}

// TestFloatModEncoding_InternalBinderCollision: operands named after the
// expansion's own binders must resolve to the OUTER values.
func TestFloatModEncoding_InternalBinderCollision(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}
	for _, names := range [][2]string{{"$ail_d", "$ail_n"}, {"$ail_q", "$ail_d"}, {"$ail_n", "$ail_q"}} {
		value, err := EncodeExpr(builtinApp("mod_Float", &core.Var{Name: names[0]}, &core.Var{Name: names[1]}))
		if err != nil {
			t.Fatal(err)
		}
		// math.Mod(-7.5, 2) = -1.5
		script := fmt.Sprintf("(set-logic ALL)\n(assert (not (let ((%s (- 7.5)) (%s 2.0)) (= %s (- 1.5)))))\n(check-sat)\n", names[0], names[1], value)
		r, err := Solve(script, DefaultSolverConfig())
		if err != nil {
			t.Fatal(err)
		}
		if r.Status != StatusVerified {
			t.Fatalf("binder capture for %v: %s", names, script)
		}
	}
}

// TestDictAppOrdering_StringUsesStrOps: Ord String comparisons are
// lexicographic, never Int `<`; `>`/`>=` flip their operands.
func TestDictAppOrdering_StringUsesStrOps(t *testing.T) {
	a, b := &core.Var{Name: "a"}, &core.Var{Name: "b"}
	want := map[string]string{
		"lt": "(str.< a b)", "le": "(str.<= a b)", "lte": "(str.<= a b)",
		"gt": "(str.< b a)", "ge": "(str.<= b a)", "gte": "(str.<= b a)",
	}
	for method, w := range want {
		got, err := encodeDictApp(&core.DictApp{Dict: &core.DictRef{ClassName: "Ord", TypeName: "String"}, Method: method, Args: []core.CoreExpr{a, b}})
		if err != nil {
			t.Fatalf("%s: %v", method, err)
		}
		if got != w {
			t.Errorf("%s: got %s, want %s", method, got, w)
		}
	}
	// Float ordering keeps the arithmetic operator.
	got, err := encodeDictApp(&core.DictApp{Dict: &core.DictRef{ClassName: "Ord", TypeName: "Float"}, Method: "gt", Args: []core.CoreExpr{a, b}})
	if err != nil || got != "(> a b)" {
		t.Errorf("Float gt: got %q, %v", got, err)
	}
}

// TestSortDispatch_UnknownSortSkips: a sort-dependent method whose instance is
// unknown or unencodable skips the function; it is never defaulted to Int.
func TestSortDispatch_UnknownSortSkips(t *testing.T) {
	a, b := &core.Var{Name: "a"}, &core.Var{Name: "b"}
	cases := map[string]*core.DictApp{
		"div, no dict":       {Method: "div", Args: []core.CoreExpr{a, b}},
		"lt, no dict":        {Method: "lt", Args: []core.CoreExpr{a, b}},
		"div on String":      {Dict: &core.DictRef{ClassName: "Num", TypeName: "String"}, Method: "div", Args: []core.CoreExpr{a, b}},
		"lt on user type":    {Dict: &core.DictRef{ClassName: "Ord", TypeName: "Color"}, Method: "lt", Args: []core.CoreExpr{a, b}},
		"mod, var dict":      {Dict: &core.Var{Name: "dOrd"}, Method: "mod", Args: []core.CoreExpr{a, b}},
		"gt, unknown method": {Dict: &core.DictRef{ClassName: "Ord", TypeName: "Bool"}, Method: "gt", Args: []core.CoreExpr{a, b}},
	}
	for name, da := range cases {
		got, err := encodeDictApp(da)
		if !errors.Is(err, ErrUnresolvableTypes) {
			t.Errorf("%s: want ErrUnresolvableTypes skip, got %q, %v", name, got, err)
		}
	}
}

// TestPreLoweredOrdering_StringEvidence: an Intrinsic comparison with a String
// literal operand uses str.<, not Int `<`.
func TestPreLoweredOrdering_StringEvidence(t *testing.T) {
	s := &core.Lit{Kind: core.StringLit, Value: "m"}
	got, err := EncodeExpr(&core.Intrinsic{Op: core.OpGt, Args: []core.CoreExpr{&core.Var{Name: "a"}, s}})
	if err != nil {
		t.Fatal(err)
	}
	if got != `(str.< "m" a)` {
		t.Errorf("got %s", got)
	}
}
