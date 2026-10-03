package eval

import (
	"errors"

	"github.com/sunholo-data/ailang/internal/core"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
)

// attachDivZeroPos gives an RT001 error the source position of the expression
// that raised it (M-INT-DIV-ZERO-ERROR, #1449).
//
// The dictionary method and the div_Int/mod_Int builtins that detect a zero
// divisor do not know their call site. evalCoreT calls this on every error it
// returns, so the first Core node on the unwinding path — the DictApp or App
// that performed the division — fills Pos; outer nodes see it set and leave it
// alone. errors.As looks through builtin-callback and contract wrapping.
// Only the error path pays for it.
func attachDivZeroPos(err error, expr core.CoreExpr) {
	var dz *ailerrors.DivByZeroError
	if !errors.As(err, &dz) || dz.Pos != "" {
		return
	}
	pos := expr.OriginalSpan()
	if pos.Line == 0 {
		pos = expr.Span()
	}
	if pos.Line > 0 {
		dz.Pos = pos.String()
	}
}

// checkIntDivisorAt is CheckIntDivisor for evaluators that already know the
// position (TypedEvaluator); pos may be empty, in which case evalCoreT fills it.
func checkIntDivisorAt(op string, divisor int, pos string) error {
	err := ailerrors.CheckIntDivisor(op, int64(divisor))
	if dz, ok := err.(*ailerrors.DivByZeroError); ok {
		dz.Pos = pos
	}
	return err
}
