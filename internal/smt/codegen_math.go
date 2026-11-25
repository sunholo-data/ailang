package smt

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/core"
)

// std/math in the verifier (M-VERIFY-SOUNDNESS-COVERAGE M5).
//
// The encoded model treats float as Real (disclosed on every run). Within it:
//
//   - floor, ceil, round, absFloat, absInt and the constants PI(), E() are
//     DEFINED exactly (floor = to_real ∘ to_int, which floors; round is half
//     away from zero, as Go's math.Round; PI/E are the exact float64 values);
//   - sqrt, exp, pow, log, sin, cos are UNINTERPRETED functions, and each call
//     contributes facts that the runtime's IEEE-754 result satisfies for every
//     non-NaN input (sqrt(x) >= 0 when x >= 0, exp(x) >= 0, pow(x, 0) == 1, …).
//     Never a value: sqrt(2.0)*sqrt(2.0) is not 2.0 at runtime. Facts are
//     ground, per call site, through the call-facts mechanism — no quantified
//     axioms (quantified nonlinear real arithmetic sends Z3 to `unknown`).
//
// Before this every caller was skipped: the std/math wrappers call _math_*
// builtins with no encoding, so the callee closure refused them.

// StdlibMathToSMT maps std/math functions to their encoder names.
var StdlibMathToSMT = map[string]string{
	"sqrt": mathSqrt, "exp": mathExp, "pow": mathPow, "log": mathLog,
	"sin": mathSin, "cos": mathCos,
	"floor": mathFloor, "ceil": mathCeil, "round": mathRound,
	"absFloat": mathAbsFloat, "abs_Float": mathAbsFloat,
	"absInt": mathAbsInt, "abs_Int": mathAbsInt,
	"PI": mathPI, "E": mathE,
}

const (
	mathSqrt     = "$ail.math_sqrt"
	mathExp      = "$ail.math_exp"
	mathPow      = "$ail.math_pow"
	mathLog      = "$ail.math_log"
	mathSin      = "$ail.math_sin"
	mathCos      = "$ail.math_cos"
	mathFloor    = "$ail.math_floor"
	mathCeil     = "$ail.math_ceil"
	mathRound    = "$ail.math_round"
	mathAbsFloat = "$ail.math_abs_float"
	mathAbsInt   = "$ail.math_abs_int"
	mathPI       = "$ail.math_pi"
	mathE        = "$ail.math_e"
)

// mathUninterpreted lists the uninterpreted functions: SMT symbol and arity.
var mathUninterpreted = map[string]struct {
	symbol string
	arity  int
}{
	mathSqrt: {"$math_sqrt", 1}, mathExp: {"$math_exp", 1}, mathPow: {"$math_pow", 2},
	mathLog: {"$math_log", 1}, mathSin: {"$math_sin", 1}, mathCos: {"$math_cos", 1},
}

// activeUninterpreted records the uninterpreted builtin functions (std/math,
// std/string) used by the function being encoded: symbol → declare-fun line.
// EncodeFunction declares exactly those.
var activeUninterpreted map[string]string

// useUninterpreted registers symbol with its declaration for this encoding.
func useUninterpreted(symbol, decl string) error {
	if activeUninterpreted == nil {
		return fmt.Errorf("%w: %s outside a function encoding", ErrUnsupportedConstruct, symbol)
	}
	activeUninterpreted[symbol] = decl
	return nil
}

// encodeMathCall encodes a std/math call. ok is false when name is not one.
func encodeMathCall(name string, args []core.CoreExpr) (string, bool, error) {
	switch name {
	case mathPI:
		return "3.141592653589793115997963468544185161590576171875", true, nil
	case mathE:
		return "2.718281828459045090795598298427648842334747314453125", true, nil
	}
	known := name == mathFloor || name == mathCeil || name == mathRound || name == mathAbsFloat || name == mathAbsInt
	uf, isUF := mathUninterpreted[name]
	if !known && !isUF {
		return "", false, nil
	}
	want := 1
	if isUF {
		want = uf.arity
	}
	if len(args) != want {
		return "", true, fmt.Errorf("%w: %s expects %d args, got %d", ErrUnsupportedConstruct, name, want, len(args))
	}
	enc := make([]string, len(args))
	for i, a := range args {
		e, err := EncodeExpr(a)
		if err != nil {
			return "", true, err
		}
		enc[i] = e
	}
	x := enc[0]
	switch name { // NaN at runtime (M4, D4 b′)
	case mathSqrt:
		addNaNObligation(fmt.Sprintf("(>= %s 0.0)", x))
	case mathLog:
		addNaNObligation(fmt.Sprintf("(> %s 0.0)", x))
	}
	floor := func(v string) string { return fmt.Sprintf("(to_real (to_int %s))", v) }
	switch name {
	case mathFloor:
		return floor(x), true, nil
	case mathCeil:
		return fmt.Sprintf("(- %s)", floor("(- "+x+")")), true, nil
	case mathRound: // half away from zero (Go math.Round), exact over the reals
		return fmt.Sprintf("(ite (>= %[1]s 0.0) %[2]s (- %[3]s))", x, floor("(+ "+x+" 0.5)"), floor("(+ (- "+x+") 0.5)")), true, nil
	case mathAbsFloat:
		return fmt.Sprintf("(ite (>= %[1]s 0.0) %[1]s (- %[1]s))", x), true, nil
	case mathAbsInt:
		return fmt.Sprintf("(ite (>= %[1]s 0) %[1]s (- %[1]s))", x), true, nil
	}
	params := "Real"
	if uf.arity == 2 {
		params = "Real Real"
	}
	if err := useUninterpreted(uf.symbol, fmt.Sprintf("(declare-fun %s (%s) Real)", uf.symbol, params)); err != nil {
		return "", true, err
	}
	call := fmt.Sprintf("(%s %s)", uf.symbol, x)
	if uf.arity == 2 {
		call = fmt.Sprintf("(%s %s %s)", uf.symbol, x, enc[1])
	}
	for _, fact := range mathFacts(name, call, enc) {
		addCallFact(fact)
	}
	return call, true, nil
}

// mathFacts are properties of the IEEE-754 result for every non-NaN input,
// stated over the encoded (real-valued) model.
func mathFacts(name, r string, a []string) []string {
	x := a[0]
	switch name {
	case mathSqrt: // correctly rounded sqrt is monotone and fixes 0 and 1
		return []string{
			fmt.Sprintf("(=> (>= %s 0.0) (>= %s 0.0))", x, r),
			fmt.Sprintf("(=> (>= %[1]s 1.0) (and (>= %[2]s 1.0) (<= %[2]s %[1]s)))", x, r),
			fmt.Sprintf("(=> (and (>= %[1]s 0.0) (<= %[1]s 1.0)) (and (>= %[2]s %[1]s) (<= %[2]s 1.0)))", x, r),
		}
	case mathExp: // never negative (it underflows to 0, so not > 0); 1 at 0
		return []string{
			fmt.Sprintf("(>= %s 0.0)", r),
			fmt.Sprintf("(=> (>= %s 0.0) (>= %s 1.0))", x, r),
			fmt.Sprintf("(=> (<= %s 0.0) (<= %s 1.0))", x, r),
		}
	case mathPow: // pow(x, 0) = 1 and pow(x, 1) = x exactly; non-negative base
		y := a[1]
		return []string{
			fmt.Sprintf("(=> (>= %s 0.0) (>= %s 0.0))", x, r),
			fmt.Sprintf("(=> (= %s 0.0) (= %s 1.0))", y, r),
			fmt.Sprintf("(=> (= %s 1.0) (= %s %s))", y, r, x),
		}
	case mathLog:
		return []string{
			fmt.Sprintf("(=> (>= %s 1.0) (>= %s 0.0))", x, r),
			fmt.Sprintf("(=> (and (> %[1]s 0.0) (<= %[1]s 1.0)) (<= %[2]s 0.0))", x, r),
		}
	case mathSin, mathCos:
		return []string{fmt.Sprintf("(and (>= %[1]s (- 1.0)) (<= %[1]s 1.0))", r)}
	}
	return nil
}

// dropUnitArgs removes unit arguments: a nullary function (PI()) is applied to
// unit in Core.
func dropUnitArgs(args []core.CoreExpr) []core.CoreExpr {
	out := args[:0:0]
	for _, a := range args {
		if lit, ok := a.(*core.Lit); ok && lit.Kind == core.UnitLit {
			continue
		}
		out = append(out, a)
	}
	return out
}

// usedUninterpreted returns the uninterpreted symbols the encoding used, sorted.
func usedUninterpreted() []string {
	names := make([]string, 0, len(activeUninterpreted))
	for s := range activeUninterpreted {
		names = append(names, s)
	}
	sort.Strings(names)
	return names
}
