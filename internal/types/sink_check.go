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
	argLabel := LabelOf(argType)
	forbidden := LabelConst(refinement.NotLabel)
	// EvalNot(L, ℓ) = true means L does NOT subsume ℓ → safe to pass the sink
	if EvalNot(argLabel, forbidden) {
		return nil
	}
	return &SinkError{ArgLabel: argLabel, SinkLabel: refinement.NotLabel}
}

// DeclassError is returned when a function changes a value's label without
// the Declassify capability in its effect row.
type DeclassError struct {
	InputLabel      Label
	OutputLabel     Label
	NeedsDeclarrify bool
}

func (e *DeclassError) Error() string {
	return fmt.Sprintf(
		"DECLASS violation: function changes label from %s to %s without Declassify in effect row — add ! {Declassify}",
		e.InputLabel, e.OutputLabel,
	)
}

// CheckDeclassify verifies that a function is not silently changing the label of
// a value between its input and output without the Declassify capability.
//
// Rule (DECLASS): if inputLabel ≠ outputLabel (and they are not equal label variables),
// the effect row MUST contain "Declassify".
//
// Identity functions (same label variable or same constant) do not require Declassify.
// ⊥-to-⊥ functions (plain unlabelled code) also do not require Declassify.
func CheckDeclassify(inputLabel, outputLabel Label, effectRow []string) *DeclassError {
	// If the labels are equal (same constant, same var, or both ⊥), no label change.
	if LabelEqual(inputLabel, outputLabel) {
		return nil
	}
	// Check if Declassify is present in the effect row.
	for _, eff := range effectRow {
		if eff == "Declassify" {
			return nil
		}
	}
	return &DeclassError{
		InputLabel:      inputLabel,
		OutputLabel:     outputLabel,
		NeedsDeclarrify: true,
	}
}
