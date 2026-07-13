package types

import (
	"strings"
	"testing"
)

// M-ARITY-STYLE golden tests for the TC_ARITY_001 coded, directional,
// style-aware arity diagnostic. See design_docs/planned/v1_0_0/m-arity-style-diagnostic.md.
//
// Two layers:
//   1. Unit tests over arityMismatchMsg (the exact rendered text).
//   2. Integration tests through Unify using the concrete App-path orientation
//      (Left = declared callee = EXPECTED arity, Right = call-site func type =
//      ACTUAL args), matching how inferApp builds the TypeEq constraint at
//      typechecker_functions.go and how it is solved at inference_helpers.go.

// funcType builds a flat (non-curried) function type with n Star-kinded params.
func funcType(n int) *TFunc2 {
	params := make([]Type, n)
	for i := range params {
		params[i] = &TVar2{Name: "p", Kind: Star}
	}
	return &TFunc2{Params: params, Return: &TVar2{Name: "r", Kind: Star}}
}

// unifyAppArity mirrors the App path: Left=declared callee (expected arity),
// Right=call-site func type built from the supplied args (actual arity).
func unifyAppArity(t *testing.T, expected, actual int) error {
	t.Helper()
	u := NewUnifier()
	_, err := u.Unify(funcType(expected), funcType(actual), Substitution{})
	return err
}

func TestArityDiagnostic_TooFew_Partial(t *testing.T) {
	// add(1) where add is arity-2: expected 2, actual 1 (partial application).
	msg := arityMismatchMsg(2, 1)

	for _, want := range []string{
		"TC_ARITY_001",
		"expects 2 argument(s)",
		"but 1 provided",
		"Suggestion:",
		"no partial application",
		"call with all 2 arguments",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("too-few/partial message missing %q\ngot:\n%s", want, msg)
		}
	}

	// End-to-end through Unify with the App orientation.
	err := unifyAppArity(t, 2, 1)
	if err == nil {
		t.Fatal("arity 2 vs 1 must fail to unify")
	}
	if !strings.Contains(err.Error(), "TC_ARITY_001") || !strings.Contains(err.Error(), "but 1 provided") {
		t.Errorf("Unify App-path error missing code/direction\ngot: %v", err)
	}
}

func TestArityDiagnostic_TooMany(t *testing.T) {
	// add(1, 2, 3) where add is arity-2: expected 2, actual 3 (over-supply).
	msg := arityMismatchMsg(2, 3)

	for _, want := range []string{
		"TC_ARITY_001",
		"expects 2 argument(s)",
		"but 3 provided",
		"Suggestion:",
		"Remove the extra 1 argument(s)",
		"this function takes 2",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("too-many message missing %q\ngot:\n%s", want, msg)
		}
	}

	// The too-many hint must NOT mention partial application.
	if strings.Contains(msg, "partial application") {
		t.Errorf("too-many message should not mention partial application\ngot:\n%s", msg)
	}

	// End-to-end through Unify with the App orientation.
	err := unifyAppArity(t, 2, 3)
	if err == nil {
		t.Fatal("arity 2 vs 3 must fail to unify")
	}
	if !strings.Contains(err.Error(), "TC_ARITY_001") || !strings.Contains(err.Error(), "but 3 provided") {
		t.Errorf("Unify App-path error missing code/direction\ngot: %v", err)
	}
}

func TestArityDiagnostic_TooFew(t *testing.T) {
	// Same family as partial: add(1) where add is arity-2.
	msg := arityMismatchMsg(2, 1)

	if !strings.Contains(msg, "TC_ARITY_001") {
		t.Errorf("too-few message missing code\ngot:\n%s", msg)
	}
	if !strings.Contains(msg, "but 1 provided") {
		t.Errorf("too-few message missing direction\ngot:\n%s", msg)
	}
	if !strings.Contains(msg, "no partial application") {
		t.Errorf("too-few (under-supply) hint must name the no-partial-application rule\ngot:\n%s", msg)
	}
}

// TestArityDiagnostic_EqualDefensive: the else branch never emits this in
// practice (it runs only when arities differ), but the helper must stay coded
// and not emit a nonsense directional hint if ever called with expected==actual.
func TestArityDiagnostic_EqualDefensive(t *testing.T) {
	msg := arityMismatchMsg(2, 2)
	if !strings.Contains(msg, "TC_ARITY_001") {
		t.Errorf("defensive equal-arity message must still be coded\ngot:\n%s", msg)
	}
	if strings.Contains(msg, "partial application") || strings.Contains(msg, "Remove the extra") {
		t.Errorf("defensive equal-arity message must not emit a directional hint\ngot:\n%s", msg)
	}
}

// TestArityDiagnostic_CurriedFlattenStillReconciles is a regression guard: the
// new message lives ONLY in the post-flatten else branch, so curried↔tupled
// forms with equal FLATTENED arity must still unify (no diagnostic emitted).
func TestArityDiagnostic_CurriedFlattenStillReconciles(t *testing.T) {
	u := NewUnifier()

	// a -> (b -> c) (curried, flattened arity 2)
	curried := &TFunc2{
		Params: []Type{&TVar2{Name: "a", Kind: Star}},
		Return: &TFunc2{
			Params: []Type{&TVar2{Name: "b", Kind: Star}},
			Return: &TVar2{Name: "c", Kind: Star},
		},
	}
	// (x, y) -> z (tupled, arity 2)
	tupled := &TFunc2{
		Params: []Type{&TVar2{Name: "x", Kind: Star}, &TVar2{Name: "y", Kind: Star}},
		Return: &TVar2{Name: "z", Kind: Star},
	}

	if _, err := u.Unify(curried, tupled, Substitution{}); err != nil {
		t.Fatalf("curried↔tupled (equal flattened arity) must still unify, got: %v", err)
	}
}
