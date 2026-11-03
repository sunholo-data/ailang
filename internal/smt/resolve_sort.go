package smt

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// Operand-sort dispatch for operators whose SMT-LIB encoding depends on the
// operand sort (M-SMT-FLOAT-DIV-MOD-SORT-DISPATCH, #2062).
//
// AILANG `/` and `%` mean different things per type: Int truncates toward zero
// (Go), Float is true division and math.Mod. SMT-LIB `div`/`mod` are Int-only,
// so choosing the encoding by operator name alone applied Int division to Real
// terms — Z3 either rejected the script (sort mismatch) or silently coerced it
// with to_int and refuted contracts that hold at runtime. Every mapping below is
// keyed by (operator, operand sort). A sort that cannot be determined skips the
// function (ErrUnresolvableTypes); it is never defaulted to Int.

// smtRealModTrunc stands for AILANG's Float `%` (Go math.Mod). Like the Int
// sentinels it is not an SMT-LIB operator: applyBinaryOp expands it through
// encodeRealMod.
const smtRealModTrunc = "$ail.fmod"

const (
	sortInt    = "Int"
	sortFloat  = "Float"
	sortString = "String"
)

// operandSort returns the AILANG sort ("Int", "Float", "String", "Bool", or a
// normalized type name) of a Core operand, or "" when it is unknown. Evidence,
// in order: literal kind, then the function's Core type info.
func operandSort(expr core.CoreExpr) string {
	if lit, ok := expr.(*core.Lit); ok {
		switch lit.Kind {
		case core.IntLit:
			return sortInt
		case core.FloatLit:
			return sortFloat
		case core.StringLit:
			return sortString
		case core.BoolLit:
			return "Bool"
		}
	}
	if c := activeADTContext; c != nil && c.ti != nil && expr != nil {
		if t, ok := c.ti.Get(expr.ID()); ok && t != nil {
			if name := types.NormalizeTypeName(t); name != "<unknown>" {
				return name
			}
		}
	}
	return ""
}

// operandsSort returns the first known operand sort.
func operandsSort(args ...core.CoreExpr) string {
	for _, a := range args {
		if s := operandSort(a); s != "" {
			return s
		}
	}
	return ""
}

// dictSort returns the instance type of a dictionary application, or "" when
// the dictionary is not a concrete DictRef.
func dictSort(da *core.DictApp) string {
	if ref, ok := da.Dict.(*core.DictRef); ok {
		return ref.TypeName
	}
	return ""
}

// divModOp resolves AILANG `/` or `%` for an operand sort.
func divModOp(isMod bool, sort string) (string, error) {
	switch sort {
	case sortInt:
		if isMod {
			return smtIntModTrunc, nil
		}
		return smtIntDivTrunc, nil
	case sortFloat:
		if isMod {
			return smtRealModTrunc, nil
		}
		return "/", nil
	}
	op := "/"
	if isMod {
		op = "%"
	}
	if sort == "" {
		return "", fmt.Errorf("%w: operand sort of %q is unknown", ErrUnresolvableTypes, op)
	}
	return "", fmt.Errorf("%w: %q on %s has no SMT encoding", ErrUnresolvableTypes, op, sort)
}

// orderOp resolves an ordering comparison for an operand sort. String ordering
// is lexicographic str.< / str.<=; `>`/`>=` flip their operands.
func orderOp(op, sort string) (smtOp string, flip bool, err error) {
	switch sort {
	case sortInt, sortFloat:
		return op, false, nil
	case sortString:
		switch op {
		case "<":
			return "str.<", false, nil
		case "<=":
			return "str.<=", false, nil
		case ">":
			return "str.<", true, nil
		case ">=":
			return "str.<=", true, nil
		}
	}
	if sort == "" {
		return "", false, fmt.Errorf("%w: operand sort of %q is unknown", ErrUnresolvableTypes, op)
	}
	return "", false, fmt.Errorf("%w: %q on %s has no SMT encoding", ErrUnresolvableTypes, op, sort)
}

// encodeRealMod encodes Go math.Mod on Reals: n - d*trunc(n/d), sign of n.
//
//	q = (/ n d)
//	trunc(q) = (ite (>= q 0) (to_int q) (- (to_int (- q))))   ; to_int floors, so negate around it
//	r = (- n (* d (to_real trunc(q))))
//
// n and d are bound by a parallel let (operands encoded once, no capture by a
// nested expansion); q is bound by an inner let so it sees those bindings.
// d = 0 leaves q uninterpreted — the runtime yields NaN there, outside the Real
// model, the same documented stance as division by zero.
func encodeRealMod(left, right string) string {
	const n, d, q = "$ail_n", "$ail_d", "$ail_q"
	return fmt.Sprintf("(let ((%[1]s %[4]s) (%[2]s %[5]s)) (let ((%[3]s (/ %[1]s %[2]s))) (- %[1]s (* %[2]s (to_real (ite (>= %[3]s 0.0) (to_int %[3]s) (- (to_int (- %[3]s)))))))))",
		n, d, q, left, right)
}
