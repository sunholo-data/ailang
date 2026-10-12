package types

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
)

// SinkError is returned when a labelled value reaches a sink that forbids its label.
// The sink is a parameter annotated with T{not LABEL}.
type SinkError struct {
	ArgLabel  Label  // the label carried by the argument
	SinkLabel string // the label name forbidden by the sink ({not SinkLabel})
}

func (e *SinkError) Error() string {
	return fmt.Sprintf(
		"sink violation: value with label %s reaches sink expecting {not %s}",
		e.ArgLabel, e.SinkLabel,
	)
}

// CheckSinkRefinement verifies that argType's label satisfies refinement.
// Returns a *SinkError if the argument's label is subsumed by the forbidden label,
// or nil if the check passes (including when refinement is nil).
func CheckSinkRefinement(argType Type, refinement *ast.RefinementExpr) *SinkError {
	if refinement == nil {
		return nil
	}
	return CheckSinkLabel(LabelOf(argType), refinement.NotLabel)
}

// CheckSinkLabel is the label-level core of the sink check: it reports a
// *SinkError when a value carrying argLabel reaches a {not notLabel} sink, or
// nil when the flow is permitted. CheckModuleIFC (ifc_check.go) calls this
// directly during its surface-AST walk; CheckSinkRefinement wraps it for the
// type-level entry point.
func CheckSinkLabel(argLabel Label, notLabel string) *SinkError {
	forbidden := LabelConst(notLabel)
	// EvalNot(L, ℓ) = true means L does NOT subsume ℓ → safe to pass the sink.
	if EvalNot(argLabel, forbidden) {
		return nil
	}
	return &SinkError{ArgLabel: argLabel, SinkLabel: notLabel}
}
