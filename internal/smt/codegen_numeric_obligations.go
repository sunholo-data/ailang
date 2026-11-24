package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// Numeric obligations (M-VERIFY-SOUNDNESS-COVERAGE M4, decisions D2(a′) and
// D4(b′); design: design_docs/planned/v0_61_0/m-verify-numeric-model.md).
//
// They ride the call-site obligation channel (codegen_obligations.go) under two
// reserved names, so they are path-scoped, emitted for the verified body only,
// and checked by one query each after the postcondition verified.
//
//   - %int64: every int + - * and unary - keeps in-range operands in range:
//     (=> (and (in64 l) (in64 r)) (in64 (op l r))). The operand guard makes the
//     check per operation, whatever the operands are (parameters, fields,
//     list elements, callee results); by induction over the expression every
//     value stays in range. Reported as a caveat by default, a violation under
//     --int-model=int64.
//   - %nan: the operations that make NaN or ±Inf from ordinary floats — real
//     division and remainder by zero, sqrt of a negative, log of a non-positive.

const (
	obligationInt64 = "%int64"
	obligationNaN   = "%nan"
	int64MinSMT     = "(- 9223372036854775808)"
	int64MaxSMT     = "9223372036854775807"
)

// intArithBuiltins are the lowered int operators the %int64 obligation covers.
var intArithBuiltins = map[string]bool{"add_Int": true, "sub_Int": true, "mul_Int": true, "neg_Int": true}

// intArithMethods are the Num-class methods covered on an Int dictionary.
var intArithMethods = map[string]bool{"add": true, "sub": true, "mul": true, "neg": true}

func in64(e string) string {
	return fmt.Sprintf("(and (<= %s %s) (<= %s %s))", int64MinSMT, e, e, int64MaxSMT)
}

// encodeIntArith encodes an int + - * or negation and records that it keeps
// in-range operands in range.
func encodeIntArith(smtOp string, args []core.CoreExpr) (string, error) {
	operands := make([]string, len(args))
	for i, a := range args {
		e, err := EncodeExpr(a)
		if err != nil {
			return "", err
		}
		operands[i] = e
	}
	var result string
	switch len(operands) {
	case 1:
		result = fmt.Sprintf("(%s %s)", smtOp, operands[0])
	case 2:
		result = applyBinaryOp(smtOp, operands[0], operands[1])
	default:
		return "", fmt.Errorf("int operator %q with unexpected arity %d", smtOp, len(operands))
	}
	guards := make([]string, len(operands))
	for i, o := range operands {
		guards[i] = in64(o)
	}
	addObligation(obligationInt64, fmt.Sprintf("(=> %s %s)", conjoinFacts(guards...), in64(result)))
	return result, nil
}

// addNaNObligation records a condition under which a float operation yields an
// ordinary number.
func addNaNObligation(condition string) {
	addObligation(obligationNaN, condition)
}

// isNumericObligation reports whether an obligation name is one of M4's.
func isNumericObligation(name string) bool {
	return strings.HasPrefix(name, "%")
}
