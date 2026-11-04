package types

import "fmt"

// EffectRowMismatchError classifies function effect failures without changing
// the raw solver text. ParamIdx is zero based; -1 denotes the function's own row.
type EffectRowMismatchError struct {
	DeclaredRow, ActualRow *Row
	ParamIdx               int
	nested                 bool // an inner callback's effects are not the outer argument's effects
	cause                  error
}

func (e *EffectRowMismatchError) Error() string {
	return fmt.Sprintf("failed to unify effect rows: %v", e.cause)
}
func (e *EffectRowMismatchError) Unwrap() error { return e.cause }
