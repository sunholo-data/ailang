package errors

// Integer division and modulo by zero (M-INT-DIV-ZERO-ERROR, #1449).
//
// Every place that divides AILANG integers — the Num[int] dictionary method,
// the div_Int/mod_Int builtins, the bytecode VM, the REPL's dictionaries —
// calls CheckIntDivisor rather than carrying its own `if b == 0`, so the
// evaluator and the VM report the same code and text. Before this, `/` was a
// raw Go panic and `%` said "[RT_DIV0] Modulo by zero" with no position.

// Operation names carried by DivByZeroError.Op.
const (
	OpDivision = "division"
	OpModulo   = "modulo"
)

// DivByZeroError is RT001: an integer `/` or `%` whose divisor is zero.
//
// It is a type rather than a formatted string so the evaluator can find it
// with errors.As through any wrapping (builtin callbacks, VM frames) and fill
// in Pos at the innermost expression that performed the division. The code
// that raises it does not know its call site, so Pos starts empty.
type DivByZeroError struct {
	Op  string // OpDivision or OpModulo
	Pos string // "file:line:col" (evaluator) or "file:line" (VM); empty if unknown
}

func (e *DivByZeroError) Error() string {
	msg := RT001 + ": integer " + e.Op + " by zero"
	if e.Pos != "" {
		msg += " at " + e.Pos
	}
	return msg
}

// Code returns the registry code, RT001.
func (e *DivByZeroError) Code() string { return RT001 }

// CheckIntDivisor returns a fresh *DivByZeroError when divisor is zero, nil
// otherwise. Fresh, not a sentinel: the evaluator writes Pos into it.
//
// Zero is the only divisor Go's integer `/` and `%` panic on: MinInt64 / -1
// wraps to MinInt64 and MinInt64 % -1 is 0 (Go spec, "Integer operators").
func CheckIntDivisor(op string, divisor int64) error {
	if divisor != 0 {
		return nil
	}
	return &DivByZeroError{Op: op}
}
