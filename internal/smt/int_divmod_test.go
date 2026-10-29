package smt

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// #1960 (M-SMT-WELL-SORTED-PREFLIGHT M0, S1): AILANG evaluates Int `/` and `%`
// with Go semantics, truncating toward zero. SMT-LIB `div`/`mod` are Euclidean
// (`-7 div 2 = -4`, `-7 mod 2 = 1`), so emitting them directly proves contracts
// that are false at runtime for negative operands.

func builtinApp(name string, a, b core.CoreExpr) core.CoreExpr {
	return &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: name}},
		Args: []core.CoreExpr{a, b},
	}
}

// intDivModEmitters is every Core shape that reaches the SMT encoder as an Int
// division or remainder. Each must encode with truncating semantics.
var intDivModEmitters = map[string]struct {
	isMod bool
	build func(a, b core.CoreExpr) core.CoreExpr
}{
	"builtin div_Int": {false, func(a, b core.CoreExpr) core.CoreExpr { return builtinApp("div_Int", a, b) }},
	"builtin mod_Int": {true, func(a, b core.CoreExpr) core.CoreExpr { return builtinApp("mod_Int", a, b) }},
	"curried div_Int": {false, func(a, b core.CoreExpr) core.CoreExpr {
		return &core.App{Func: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "div_Int"}},
			Args: []core.CoreExpr{a},
		}, Args: []core.CoreExpr{b}}
	}},
	"curried mod_Int": {true, func(a, b core.CoreExpr) core.CoreExpr {
		return &core.App{Func: &core.App{Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "mod_Int"}}, Args: []core.CoreExpr{a}}, Args: []core.CoreExpr{b}}
	}},
	"intrinsic OpDiv": {false, func(a, b core.CoreExpr) core.CoreExpr {
		return &core.Intrinsic{Op: core.OpDiv, Args: []core.CoreExpr{a, b}}
	}},
	"intrinsic OpMod": {true, func(a, b core.CoreExpr) core.CoreExpr {
		return &core.Intrinsic{Op: core.OpMod, Args: []core.CoreExpr{a, b}}
	}},
	"binop /": {false, func(a, b core.CoreExpr) core.CoreExpr { return &core.BinOp{Op: "/", Left: a, Right: b} }},
	"binop %": {true, func(a, b core.CoreExpr) core.CoreExpr { return &core.BinOp{Op: "%", Left: a, Right: b} }},
	"dictapp div": {false, func(a, b core.CoreExpr) core.CoreExpr {
		return &core.DictApp{Dict: &core.DictRef{ClassName: "Num", TypeName: "Int"}, Method: "div", Args: []core.CoreExpr{a, b}}
	}},
}

func smtIntLit(v int64) string {
	if v < 0 {
		return fmt.Sprintf("(- %d)", -v)
	}
	return fmt.Sprintf("%d", v)
}

// TestIntDivModEncoding_MatchesGoTruncation asks Z3 to refute the Go result of
// every sign combination, for every emitter. unsat means the encoding agrees with
// the runtime; sat means the verifier models a different operator.
func TestIntDivModEncoding_MatchesGoTruncation(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}
	pairs := [][2]int64{{7, 2}, {-7, 2}, {7, -2}, {-7, -2}, {6, 3}, {-6, 3}, {0, -5}, {0, 5}, {-1, 7}, {1, -7}, {-6, -3}, {6, -3}, {9223372036854775807, -1}}
	for name, em := range intDivModEmitters {
		for _, p := range pairs {
			a, b := p[0], p[1]
			want := a / b
			if em.isMod {
				want = a % b
			}
			enc, err := EncodeExpr(em.build(intLit(a), intLit(b)))
			if err != nil {
				t.Fatalf("%s(%d, %d): encode: %v", name, a, b, err)
			}
			script := fmt.Sprintf("(set-logic ALL)\n(assert (not (= %s %s)))\n(check-sat)\n", enc, smtIntLit(want))
			res, err := Solve(script, DefaultSolverConfig())
			if err != nil {
				t.Fatalf("%s(%d, %d): solve: %v", name, a, b, err)
			}
			if res.Status != StatusVerified {
				t.Errorf("%s(%d, %d): encoding %s does not equal Go's %d (Z3 %v)", name, a, b, enc, want, res.Status)
			}
		}
	}
}

// TestIntDivModEncoding_NoBareEuclideanOperator pins the shape: the operands are
// let-bound once, so a compound operand is encoded once and cannot be captured.
func TestIntDivModEncoding_NoBareEuclideanOperator(t *testing.T) {
	x := &core.Var{Name: "x"}
	for name, em := range intDivModEmitters {
		enc, err := EncodeExpr(em.build(x, intLit(2)))
		if err != nil {
			t.Fatalf("%s: encode: %v", name, err)
		}
		if strings.HasPrefix(enc, "(div ") || strings.HasPrefix(enc, "(mod ") {
			t.Errorf("%s: emitted a bare Euclidean operator: %s", name, enc)
		}
		if !strings.Contains(enc, "abs") || strings.Count(enc, "x") != 1 {
			t.Errorf("%s: want a let-bound truncating form using abs with x encoded once, got %s", name, enc)
		}
	}
}

// TestIntDivModEncoding_NestedOperands: a let-bound operand that itself divides
// must not be captured by the inner binding names.
func TestIntDivModEncoding_NestedOperands(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}
	// ((-7 % 3) / (-7 / 2)) == (-1) / (-3) == 0 under Go; and (-20 / -3) % (-7 % 4) == 6 % -3 == 0
	cases := []struct {
		expr core.CoreExpr
		want int64
	}{
		{builtinApp("div_Int", builtinApp("mod_Int", intLit(-7), intLit(3)), builtinApp("div_Int", intLit(-7), intLit(2))), (-7 % 3) / (-7 / 2)},
		{builtinApp("mod_Int", builtinApp("div_Int", intLit(-20), intLit(-3)), builtinApp("mod_Int", intLit(-7), intLit(4))), (-20 / -3) % (-7 % 4)},
		{builtinApp("div_Int", builtinApp("div_Int", intLit(-100), intLit(7)), intLit(-3)), (-100 / 7) / -3},
	}
	for i, c := range cases {
		enc, err := EncodeExpr(c.expr)
		if err != nil {
			t.Fatalf("case %d: encode: %v", i, err)
		}
		script := fmt.Sprintf("(set-logic ALL)\n(assert (not (= %s %s)))\n(check-sat)\n", enc, smtIntLit(c.want))
		res, err := Solve(script, DefaultSolverConfig())
		if err != nil {
			t.Fatalf("case %d: solve: %v", i, err)
		}
		if res.Status != StatusVerified {
			t.Errorf("case %d: %s != %d (Z3 %v)", i, enc, c.want, res.Status)
		}
	}
}

func TestIntDivModEncoding_InternalBinderCollision(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}
	// Both operands name the encoder's internal binders. Parallel-let RHSs
	// must resolve to the OUTER values, including inside a nested expansion.
	a, b := &core.Var{Name: "$ail_d"}, &core.Var{Name: "$ail_n"}
	for name, em := range intDivModEmitters {
		value, err := EncodeExpr(em.build(a, b))
		if err != nil {
			t.Fatal(err)
		}
		want := int64(-3)
		if em.isMod {
			want = -1
		}
		script := fmt.Sprintf("(set-logic ALL)\n(assert (not (let (($ail_d (- 7)) ($ail_n 2)) (= %s %s))))\n(check-sat)\n", value, smtIntLit(want))
		r, err := Solve(script, DefaultSolverConfig())
		if err != nil {
			t.Fatal(err)
		}
		if r.Status != StatusVerified {
			t.Fatalf("%s binder capture: %s", name, script)
		}
	}
}
