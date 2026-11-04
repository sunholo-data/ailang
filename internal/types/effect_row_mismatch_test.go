package types

import (
	"errors"
	"strings"
	"testing"
)

func TestEffectRowMismatchClassification(t *testing.T) {
	pure := &TFunc2{Return: TInt, EffectRow: EmptyEffectRow()}
	io := &TFunc2{Return: TInt, EffectRow: &Row{Kind: EffectRow, Labels: map[string]Type{"IO": TUnit}}}
	for _, nested := range []bool{false, true} {
		var left, right Type = pure, io
		index := -1
		if nested {
			left = &TFunc2{Params: []Type{pure}, Return: TInt}
			right = &TFunc2{Params: []Type{io}, Return: TInt}
			index = 0
		}
		_, err := NewUnifier().Unify(left, right, Substitution{})
		var mismatch *EffectRowMismatchError
		if !errors.As(err, &mismatch) || mismatch.ParamIdx != index {
			t.Fatalf("classification: %v (%+v)", err, mismatch)
		}
		if !strings.Contains(err.Error(), "failed to unify effect rows: incompatible closed rows") {
			t.Fatalf("changed solver text: %v", err)
		}
	}
}
