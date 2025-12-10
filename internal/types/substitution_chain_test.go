package types

import (
	"testing"
)

// TestSubstitutionChainResolution verifies that ApplySubstitution follows chains.
// This test was added after M-FIX-FLOAT-OP discovered that chains weren't being followed:
// When α -> β and β -> float, applying to α should return float, not β.
func TestSubstitutionChainResolution(t *testing.T) {
	// Create chain: α -> β -> float
	sub := Substitution{
		"α": &TVar2{Name: "β"},
		"β": TFloat,
	}

	// Apply to α - should get float, not β
	result := ApplySubstitution(sub, &TVar2{Name: "α"})

	// The result should be TFloat, not TVar2{β}
	if tvar, ok := result.(*TVar2); ok {
		t.Errorf("Chain not resolved: got TVar2{%s}, want TFloat", tvar.Name)
	}

	if result != TFloat {
		t.Errorf("ApplySubstitution(α) = %v, want TFloat", result)
	}
}

// TestSubstitutionIdempotent verifies that applying substitution twice gives same result.
// This is a fundamental property: S(S(t)) = S(t) for a well-formed substitution.
func TestSubstitutionIdempotent(t *testing.T) {
	sub := Substitution{
		"α": &TVar2{Name: "β"},
		"β": TFloat,
		"γ": TInt,
	}

	testCases := []struct {
		name string
		typ  Type
	}{
		{"direct var", &TVar2{Name: "α"}},
		{"intermediate var", &TVar2{Name: "β"}},
		{"concrete", TFloat},
		{"function", &TFunc{Params: []Type{&TVar2{Name: "α"}}, Return: &TVar2{Name: "β"}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			once := ApplySubstitution(sub, tc.typ)
			twice := ApplySubstitution(sub, once)

			// Applying twice should give same result as once
			if once.String() != twice.String() {
				t.Errorf("Not idempotent: S(t)=%v, S(S(t))=%v", once, twice)
			}
		})
	}
}

// TestSubstitutionLongChain tests longer substitution chains.
func TestSubstitutionLongChain(t *testing.T) {
	// Create longer chain: α -> β -> γ -> δ -> float
	sub := Substitution{
		"α": &TVar2{Name: "β"},
		"β": &TVar2{Name: "γ"},
		"γ": &TVar2{Name: "δ"},
		"δ": TFloat,
	}

	result := ApplySubstitution(sub, &TVar2{Name: "α"})

	if result != TFloat {
		t.Errorf("Long chain not resolved: got %v, want TFloat", result)
	}
}

// TestSubstitutionNoChain verifies direct mappings still work.
func TestSubstitutionNoChain(t *testing.T) {
	sub := Substitution{
		"α": TFloat,
		"β": TInt,
	}

	result := ApplySubstitution(sub, &TVar2{Name: "α"})
	if result != TFloat {
		t.Errorf("Direct mapping failed: got %v, want TFloat", result)
	}

	result = ApplySubstitution(sub, &TVar2{Name: "β"})
	if result != TInt {
		t.Errorf("Direct mapping failed: got %v, want TInt", result)
	}
}

// TestSubstitutionInFunction tests that chains are resolved inside function types.
func TestSubstitutionInFunction(t *testing.T) {
	sub := Substitution{
		"α": &TVar2{Name: "β"},
		"β": TFloat,
	}

	funcType := &TFunc{
		Params: []Type{&TVar2{Name: "α"}, &TVar2{Name: "α"}},
		Return: &TVar2{Name: "α"},
	}

	result := ApplySubstitution(sub, funcType)

	fn, ok := result.(*TFunc)
	if !ok {
		t.Fatalf("Expected TFunc, got %T", result)
	}

	// All αs should be resolved to float
	for i, param := range fn.Params {
		if param != TFloat {
			t.Errorf("Param %d not resolved: got %v, want TFloat", i, param)
		}
	}
	if fn.Return != TFloat {
		t.Errorf("Return not resolved: got %v, want TFloat", fn.Return)
	}
}
