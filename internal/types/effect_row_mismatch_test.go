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

func TestEffectRowMismatchOpenAndNested(t *testing.T) {
	pure := &TFunc2{Return: TInt, EffectRow: EmptyEffectRow()}
	io := &TFunc2{Return: TInt, EffectRow: &Row{Kind: EffectRow, Labels: map[string]Type{"IO": TUnit}, Tail: &RowVar{Name: "e"}}}
	_, err := NewUnifier().Unify(pure, io, Substitution{})
	var mismatch *EffectRowMismatchError
	if !errors.As(err, &mismatch) || mismatch.ParamIdx != -1 {
		t.Fatalf("open row: %v", err)
	}
	if got, want := err.Error(), "failed to unify effect rows: "+errors.Unwrap(err).Error(); got != want {
		t.Fatalf("changed text: %s != %s", got, want)
	}
	left := &TFunc2{Params: []Type{&TFunc2{Params: []Type{pure}, Return: TInt}}, Return: TInt}
	right := &TFunc2{Params: []Type{&TFunc2{Params: []Type{io}, Return: TInt}}, Return: TInt}
	_, err = NewUnifier().Unify(left, right, Substitution{})
	if !errors.As(err, &mismatch) || !mismatch.nested {
		t.Fatalf("nested callback: %v", err)
	}
	_, err = NewUnifier().Unify(&Row{Kind: RecordRow, Labels: map[string]Type{"a": TInt}}, &Row{Kind: RecordRow, Labels: map[string]Type{"b": TInt}}, Substitution{})
	if err == nil || errors.As(err, &mismatch) {
		t.Fatalf("record row classified as effect: %v", err)
	}
}
