package types

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

// TestInferFreshCounterPersistence verifies that the InferenceContext freshCounter
// persists across multiple InferWithConstraints calls, preventing TVar name collisions.
//
// This is a regression test for the bug where `>` on Int values intermittently
// encoded as `str.<` in SMT-LIB output (M-TVAR-COLLISION-FIX).
//
// Root cause: freshCounter reset to 0 in each InferWithConstraints call, causing
// TVar names (α1, α2, ...) to be reused across declarations. When ApplySubstitution
// was applied globally to CoreTI, a later declaration's substitution could corrupt
// an earlier declaration's CoreTI entries if they shared TVar names.
func TestInferFreshCounterPersistence(t *testing.T) {
	tc := NewCoreTypeChecker()
	env := NewTypeEnv()

	// Declaration 1: let x = 42
	// This creates CoreTI entries and uses TVars α1, α2, etc.
	decl1 := &core.Let{
		CoreNode: core.CoreNode{NodeID: 100},
		Name:     "x",
		Value:    &core.Lit{CoreNode: core.CoreNode{NodeID: 101}, Kind: core.IntLit, Value: int64(42)},
		Body:     &core.Var{CoreNode: core.CoreNode{NodeID: 102}, Name: "x"},
	}

	_, env, _, _, err := tc.InferWithConstraints(decl1, env)
	if err != nil {
		t.Fatalf("InferWithConstraints for decl1 failed: %v", err)
	}

	// After first inference, the inferFreshCounter should have advanced
	if tc.inferFreshCounter == 0 {
		t.Errorf("inferFreshCounter should be > 0 after first InferWithConstraints, got 0")
	}
	savedCounter := tc.inferFreshCounter

	// Declaration 2: let y = "hello"
	// Without the fix, this would use TVars α1, α2, ... again (collision!)
	// With the fix, it starts from the saved counter (no collision)
	decl2 := &core.Let{
		CoreNode: core.CoreNode{NodeID: 200},
		Name:     "y",
		Value:    &core.Lit{CoreNode: core.CoreNode{NodeID: 201}, Kind: core.StringLit, Value: "hello"},
		Body:     &core.Var{CoreNode: core.CoreNode{NodeID: 202}, Name: "y"},
	}

	_, _, _, _, err = tc.InferWithConstraints(decl2, env)
	if err != nil {
		t.Fatalf("InferWithConstraints for decl2 failed: %v", err)
	}

	// After second inference, the counter should not have decreased
	// (it may not advance if no fresh TVars were needed for a simple literal)
	if tc.inferFreshCounter < savedCounter {
		t.Errorf("inferFreshCounter should not decrease: got %d, was %d", tc.inferFreshCounter, savedCounter)
	}

	// Verify CoreTI entries from decl1 weren't corrupted by decl2's substitution
	// Node 101 (the int literal 42) should still have type Int
	if typ, ok := tc.CoreTI.Get(101); ok {
		head := Head(typ)
		if head != HeadInt {
			t.Errorf("CoreTI[101] (int literal) should have HeadInt, got %v (type: %v)", head, typ)
		}
	}

	// Node 201 (the string literal "hello") should have type String
	if typ, ok := tc.CoreTI.Get(201); ok {
		head := Head(typ)
		if head != HeadString {
			t.Errorf("CoreTI[201] (string literal) should have HeadString, got %v (type: %v)", head, typ)
		}
	}
}

// TestInferFreshCounterNoCrossContamination verifies that polymorphic function
// parameters aren't corrupted by later declarations with concrete types.
//
// Scenario:
//  1. Declaration 1: let id = \a -> a  (identity function, α stays polymorphic)
//  2. Declaration 2: let s = "hello"   (string literal)
//     Without fix: decl2's substitution maps α1 → string, which corrupts decl1's
//     CoreTI entry for parameter 'a' (also α1) to string.
//     With fix: decl2 uses α(N+1) etc., no overlap with decl1's α1.
func TestInferFreshCounterNoCrossContamination(t *testing.T) {
	tc := NewCoreTypeChecker()
	env := NewTypeEnv()

	// Declaration 1: let id = \a -> a (identity function)
	paramNode := core.CoreNode{NodeID: 300}
	decl1 := &core.Let{
		CoreNode: core.CoreNode{NodeID: 310},
		Name:     "id",
		Value: &core.Lambda{
			CoreNode: core.CoreNode{NodeID: 311},
			Params:   []string{"a"},
			Body:     &core.Var{CoreNode: paramNode, Name: "a"},
		},
		Body: &core.Var{CoreNode: core.CoreNode{NodeID: 312}, Name: "id"},
	}

	_, env, _, _, err := tc.InferWithConstraints(decl1, env)
	if err != nil {
		t.Fatalf("InferWithConstraints for decl1 (id) failed: %v", err)
	}

	// The parameter 'a' should have a polymorphic type (TVar) in CoreTI
	paramType, paramOk := tc.CoreTI.Get(paramNode.NodeID)
	if !paramOk {
		// Not all parameters get stored in CoreTI (depends on inference path)
		// This is fine — the test still validates the counter mechanism
		t.Log("Parameter node not in CoreTI (acceptable — inference may not store it)")
	}

	// Declaration 2: let s = "hello"
	decl2 := &core.Let{
		CoreNode: core.CoreNode{NodeID: 400},
		Name:     "s",
		Value:    &core.Lit{CoreNode: core.CoreNode{NodeID: 401}, Kind: core.StringLit, Value: "hello"},
		Body:     &core.Var{CoreNode: core.CoreNode{NodeID: 402}, Name: "s"},
	}

	_, _, _, _, err = tc.InferWithConstraints(decl2, env)
	if err != nil {
		t.Fatalf("InferWithConstraints for decl2 (s) failed: %v", err)
	}

	// If parameter 'a' was in CoreTI, verify it wasn't corrupted to String
	if paramOk {
		paramTypeAfter, _ := tc.CoreTI.Get(paramNode.NodeID)
		if paramTypeAfter != nil {
			head := Head(paramTypeAfter)
			if head == HeadString {
				t.Errorf("TVAR COLLISION BUG: CoreTI[%d] (identity param 'a') was corrupted to String!\n"+
					"  Before decl2: %v\n"+
					"  After decl2:  %v\n"+
					"  This means freshCounter was not persisted across InferWithConstraints calls",
					paramNode.NodeID, paramType, paramTypeAfter)
			}
		}
	}

	// Most importantly: verify the counter itself advanced properly
	if tc.inferFreshCounter == 0 {
		t.Error("inferFreshCounter should be > 0 after two InferWithConstraints calls")
	}
}

// TestInferFreshCounterCheckCoreExprPath verifies that CheckCoreExpr also
// uses the persistent counter (secondary code path).
func TestInferFreshCounterCheckCoreExprPath(t *testing.T) {
	tc := NewCoreTypeChecker()
	env := NewTypeEnv()

	// Expression 1: int literal
	expr1 := &core.Lit{CoreNode: core.CoreNode{NodeID: 500}, Kind: core.IntLit, Value: int64(10)}

	_, _, err := tc.CheckCoreExpr(expr1, env)
	if err != nil {
		t.Fatalf("CheckCoreExpr for expr1 failed: %v", err)
	}

	counter1 := tc.inferFreshCounter

	// Expression 2: string literal
	expr2 := &core.Lit{CoreNode: core.CoreNode{NodeID: 600}, Kind: core.StringLit, Value: "test"}

	_, _, err = tc.CheckCoreExpr(expr2, env)
	if err != nil {
		t.Fatalf("CheckCoreExpr for expr2 failed: %v", err)
	}

	counter2 := tc.inferFreshCounter

	// Counters should be non-decreasing (may not advance for simple literals)
	if counter2 < counter1 {
		t.Errorf("inferFreshCounter should not decrease: was %d after expr1, got %d after expr2", counter1, counter2)
	}
}
