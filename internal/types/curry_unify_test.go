package types

import (
	"testing"
)

// TestCurriedLambdaUnifiesWithMultiParam verifies that a curried function type
// (a -> (b -> c)) unifies with a multi-param function type ((a, b) -> c).
// This is the core bug in M-DOCPARSE-DX M1: foldl(\acc. \cell. expr) fails
// because the typechecker creates a 1-arity function returning a function,
// but foldl expects a 2-arity function.
func TestCurriedLambdaUnifiesWithMultiParam(t *testing.T) {
	u := NewUnifier()

	// Curried form: a -> (b -> c) — what \acc. \cell. acc + cell produces
	curried := &TFunc2{
		Params: []Type{&TVar2{Name: "a", Kind: Star}},
		Return: &TFunc2{
			Params: []Type{&TVar2{Name: "b", Kind: Star}},
			Return: &TVar2{Name: "c", Kind: Star},
		},
	}

	// Multi-param form: (a, b) -> c — what foldl expects
	multiParam := &TFunc2{
		Params: []Type{&TVar2{Name: "x", Kind: Star}, &TVar2{Name: "y", Kind: Star}},
		Return: &TVar2{Name: "z", Kind: Star},
	}

	sub, err := u.Unify(curried, multiParam, Substitution{})
	if err != nil {
		t.Fatalf("curried (a -> (b -> c)) should unify with multi-param ((x, y) -> z), got error: %v", err)
	}

	// Verify substitution maps correctly
	// a should unify with x, b with y, c with z
	if sub == nil {
		t.Fatal("expected non-nil substitution")
	}
}

// TestCurriedLambdaUnifiesWithConcreteTypes tests curried unification
// with concrete types (int -> (int -> int)) vs ((int, int) -> int).
func TestCurriedLambdaUnifiesWithConcreteTypes(t *testing.T) {
	u := NewUnifier()

	// Curried: int -> (int -> int)
	curried := &TFunc2{
		Params: []Type{TInt},
		Return: &TFunc2{
			Params: []Type{TInt},
			Return: TInt,
		},
	}

	// Multi-param: (int, int) -> int
	multiParam := &TFunc2{
		Params: []Type{TInt, TInt},
		Return: TInt,
	}

	_, err := u.Unify(curried, multiParam, Substitution{})
	if err != nil {
		t.Fatalf("curried (int -> (int -> int)) should unify with ((int, int) -> int), got error: %v", err)
	}
}

// TestTripleCurriedUnifies tests 3-level currying: a -> (b -> (c -> d))
// should unify with (a, b, c) -> d.
func TestTripleCurriedUnifies(t *testing.T) {
	u := NewUnifier()

	// Triple curried: a -> (b -> (c -> d))
	curried := &TFunc2{
		Params: []Type{&TVar2{Name: "a", Kind: Star}},
		Return: &TFunc2{
			Params: []Type{&TVar2{Name: "b", Kind: Star}},
			Return: &TFunc2{
				Params: []Type{&TVar2{Name: "c", Kind: Star}},
				Return: &TVar2{Name: "d", Kind: Star},
			},
		},
	}

	// Multi-param: (x, y, z) -> w
	multiParam := &TFunc2{
		Params: []Type{&TVar2{Name: "x", Kind: Star}, &TVar2{Name: "y", Kind: Star}, &TVar2{Name: "z", Kind: Star}},
		Return: &TVar2{Name: "w", Kind: Star},
	}

	_, err := u.Unify(curried, multiParam, Substitution{})
	if err != nil {
		t.Fatalf("triple curried should unify with 3-param, got error: %v", err)
	}
}

// TestMultiParamStillWorks ensures existing multi-param lambdas are unaffected.
func TestMultiParamStillWorks(t *testing.T) {
	u := NewUnifier()

	// Both multi-param: (a, b) -> c with (x, y) -> z
	f1 := &TFunc2{
		Params: []Type{&TVar2{Name: "a", Kind: Star}, &TVar2{Name: "b", Kind: Star}},
		Return: &TVar2{Name: "c", Kind: Star},
	}
	f2 := &TFunc2{
		Params: []Type{&TVar2{Name: "x", Kind: Star}, &TVar2{Name: "y", Kind: Star}},
		Return: &TVar2{Name: "z", Kind: Star},
	}

	_, err := u.Unify(f1, f2, Substitution{})
	if err != nil {
		t.Fatalf("multi-param should unify with multi-param, got error: %v", err)
	}
}

// TestCurriedMismatchStillFails ensures that a genuine arity mismatch still fails.
// (a -> (b -> c)) should NOT unify with (x, y, z) -> w (3 params vs 2 flattened).
func TestCurriedMismatchStillFails(t *testing.T) {
	u := NewUnifier()

	// Curried 2-param: a -> (b -> c)
	curried := &TFunc2{
		Params: []Type{&TVar2{Name: "a", Kind: Star}},
		Return: &TFunc2{
			Params: []Type{&TVar2{Name: "b", Kind: Star}},
			Return: &TVar2{Name: "c", Kind: Star},
		},
	}

	// 3-param: (x, y, z) -> w
	threeParam := &TFunc2{
		Params: []Type{&TVar2{Name: "x", Kind: Star}, &TVar2{Name: "y", Kind: Star}, &TVar2{Name: "z", Kind: Star}},
		Return: &TVar2{Name: "w", Kind: Star},
	}

	_, err := u.Unify(curried, threeParam, Substitution{})
	if err == nil {
		t.Fatal("curried 2-param should NOT unify with 3-param")
	}
}

// TestSymmetricCurriedUnification tests that flattening works in both directions.
// If t1 is multi-param and t2 is curried, it should still unify.
func TestSymmetricCurriedUnification(t *testing.T) {
	u := NewUnifier()

	// Multi-param first: (a, b) -> c
	multiParam := &TFunc2{
		Params: []Type{&TVar2{Name: "a", Kind: Star}, &TVar2{Name: "b", Kind: Star}},
		Return: &TVar2{Name: "c", Kind: Star},
	}

	// Curried second: x -> (y -> z)
	curried := &TFunc2{
		Params: []Type{&TVar2{Name: "x", Kind: Star}},
		Return: &TFunc2{
			Params: []Type{&TVar2{Name: "y", Kind: Star}},
			Return: &TVar2{Name: "z", Kind: Star},
		},
	}

	_, err := u.Unify(multiParam, curried, Substitution{})
	if err != nil {
		t.Fatalf("symmetric: multi-param should unify with curried, got error: %v", err)
	}
}

// TestPartialCurriedUnification tests mixed currying: (a, b) -> (c -> d)
// should unify with (a, b, c) -> d.
func TestPartialCurriedUnification(t *testing.T) {
	u := NewUnifier()

	// Partially curried: (a, b) -> (c -> d) — 2 params, return is function
	partial := &TFunc2{
		Params: []Type{&TVar2{Name: "a", Kind: Star}, &TVar2{Name: "b", Kind: Star}},
		Return: &TFunc2{
			Params: []Type{&TVar2{Name: "c", Kind: Star}},
			Return: &TVar2{Name: "d", Kind: Star},
		},
	}

	// Flat 3-param: (x, y, z) -> w
	flat := &TFunc2{
		Params: []Type{&TVar2{Name: "x", Kind: Star}, &TVar2{Name: "y", Kind: Star}, &TVar2{Name: "z", Kind: Star}},
		Return: &TVar2{Name: "w", Kind: Star},
	}

	_, err := u.Unify(partial, flat, Substitution{})
	if err != nil {
		t.Fatalf("partial curried (a,b) -> (c -> d) should unify with (x,y,z) -> w, got error: %v", err)
	}
}
