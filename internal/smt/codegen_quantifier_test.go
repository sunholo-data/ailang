package smt

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

func TestEncodeForall_Basic(t *testing.T) {
	// forall i: 0..n => i >= 0
	// should produce: (forall ((i Int)) (=> (and (>= i 0) (< i n)) (>= i 0)))
	f := &core.Forall{
		Var: "i",
		Lo:  &core.Lit{Kind: core.IntLit, Value: int64(0)},
		Hi:  &core.Var{Name: "n"},
		Body: &core.BinOp{
			Op:    ">=",
			Left:  &core.Var{Name: "i"},
			Right: &core.Lit{Kind: core.IntLit, Value: int64(0)},
		},
	}

	result, err := EncodeExpr(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "(forall ((i Int)) (=> (and (>= i 0) (< i n)) (>= i 0)))"
	if result != expected {
		t.Errorf("expected:\n  %s\ngot:\n  %s", expected, result)
	}
}

func TestEncodeForall_WithLiterals(t *testing.T) {
	// forall j: 0..10 => j >= 0
	f := &core.Forall{
		Var: "j",
		Lo:  &core.Lit{Kind: core.IntLit, Value: int64(0)},
		Hi:  &core.Lit{Kind: core.IntLit, Value: int64(10)},
		Body: &core.BinOp{
			Op:    ">=",
			Left:  &core.Var{Name: "j"},
			Right: &core.Lit{Kind: core.IntLit, Value: int64(0)},
		},
	}

	result, err := EncodeExpr(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "(forall ((j Int)) (=> (and (>= j 0) (< j 10)) (>= j 0)))"
	if result != expected {
		t.Errorf("expected:\n  %s\ngot:\n  %s", expected, result)
	}
}

func TestEncodeForall_NestedQuantifiers(t *testing.T) {
	// Nested forall: forall i: 0..n => forall j: 0..m => i + j >= 0
	// Z3 handles nested quantifiers, so this should encode correctly.
	inner := &core.Forall{
		Var: "j",
		Lo:  &core.Lit{Kind: core.IntLit, Value: int64(0)},
		Hi:  &core.Var{Name: "m"},
		Body: &core.BinOp{
			Op: ">=",
			Left: &core.BinOp{
				Op:    "+",
				Left:  &core.Var{Name: "i"},
				Right: &core.Var{Name: "j"},
			},
			Right: &core.Lit{Kind: core.IntLit, Value: int64(0)},
		},
	}

	outer := &core.Forall{
		Var:  "i",
		Lo:   &core.Lit{Kind: core.IntLit, Value: int64(0)},
		Hi:   &core.Var{Name: "n"},
		Body: inner,
	}

	result, err := EncodeExpr(outer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain two nested forall declarations
	if !strings.Contains(result, "(forall ((i Int))") {
		t.Error("expected outer forall with i")
	}
	if !strings.Contains(result, "(forall ((j Int))") {
		t.Error("expected inner forall with j")
	}
}

func TestEncodeForall_InLet(t *testing.T) {
	// let n = 5 in forall i: 0..n => i >= 0
	expr := &core.Let{
		Name:  "n",
		Value: &core.Lit{Kind: core.IntLit, Value: int64(5)},
		Body: &core.Forall{
			Var: "i",
			Lo:  &core.Lit{Kind: core.IntLit, Value: int64(0)},
			Hi:  &core.Var{Name: "n"},
			Body: &core.BinOp{
				Op:    ">=",
				Left:  &core.Var{Name: "i"},
				Right: &core.Lit{Kind: core.IntLit, Value: int64(0)},
			},
		},
	}

	result, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "(let ((n 5))") {
		t.Error("expected let binding")
	}
	if !strings.Contains(result, "(forall ((i Int))") {
		t.Error("expected forall in let body")
	}
}

func TestEncodeForall_ContainsRefShadowing(t *testing.T) {
	// Test that containsRef respects forall variable shadowing.
	// forall x: 0..10 => x >= 0
	// If we search for "x", the body should NOT be searched because x is bound.
	forallExpr := &core.Forall{
		Var: "x",
		Lo:  &core.Lit{Kind: core.IntLit, Value: int64(0)},
		Hi:  &core.Lit{Kind: core.IntLit, Value: int64(10)},
		Body: &core.BinOp{
			Op:    ">=",
			Left:  &core.Var{Name: "x"},
			Right: &core.Lit{Kind: core.IntLit, Value: int64(0)},
		},
	}

	// x is bound by forall, so containsRef should return false
	if containsRef(forallExpr, "x") {
		t.Error("containsRef should return false for forall-bound variable")
	}

	// y is not bound, so if it appears in bounds, it should be found
	forallWithY := &core.Forall{
		Var: "x",
		Lo:  &core.Var{Name: "y"},
		Hi:  &core.Lit{Kind: core.IntLit, Value: int64(10)},
		Body: &core.BinOp{
			Op:    ">=",
			Left:  &core.Var{Name: "x"},
			Right: &core.Lit{Kind: core.IntLit, Value: int64(0)},
		},
	}
	if !containsRef(forallWithY, "y") {
		t.Error("containsRef should find 'y' in forall bounds")
	}
}

func TestReplaceSelfCalls_Forall(t *testing.T) {
	// Test that ReplaceSelfCalls handles Forall nodes correctly.
	f := &core.Forall{
		Var: "i",
		Lo:  &core.Lit{Kind: core.IntLit, Value: int64(0)},
		Hi:  &core.Var{Name: "foo"},
		Body: &core.BinOp{
			Op:    ">=",
			Left:  &core.Var{Name: "i"},
			Right: &core.Lit{Kind: core.IntLit, Value: int64(0)},
		},
	}

	result := ReplaceSelfCalls(f, "foo", "foo_1")

	replaced, ok := result.(*core.Forall)
	if !ok {
		t.Fatalf("expected Forall, got %T", result)
	}

	// Hi should have been replaced: foo -> foo_1
	hiVar, ok := replaced.Hi.(*core.Var)
	if !ok {
		t.Fatalf("expected Var for Hi, got %T", replaced.Hi)
	}
	if hiVar.Name != "foo_1" {
		t.Errorf("expected Hi to be 'foo_1', got %q", hiVar.Name)
	}
}

func TestReplaceSelfCalls_ForallShadowing(t *testing.T) {
	// If forall variable shadows the function name, body should NOT be replaced
	f := &core.Forall{
		Var: "foo",
		Lo:  &core.Lit{Kind: core.IntLit, Value: int64(0)},
		Hi:  &core.Var{Name: "n"},
		Body: &core.BinOp{
			Op:    ">=",
			Left:  &core.Var{Name: "foo"},
			Right: &core.Lit{Kind: core.IntLit, Value: int64(0)},
		},
	}

	result := ReplaceSelfCalls(f, "foo", "foo_1")

	replaced, ok := result.(*core.Forall)
	if !ok {
		t.Fatalf("expected Forall, got %T", result)
	}

	// Body should NOT have been replaced because foo is shadowed by forall variable
	bodyBinOp := replaced.Body.(*core.BinOp)
	leftVar := bodyBinOp.Left.(*core.Var)
	if leftVar.Name != "foo" {
		t.Errorf("expected body variable to remain 'foo' (shadowed), got %q", leftVar.Name)
	}
}

func TestForall_SMTEncodableFragment(t *testing.T) {
	// A function with forall in ensures should still be SMT-encodable
	// (assuming other fragment criteria are met)
	body := &core.BinOp{
		Op:    "+",
		Left:  &core.Var{Name: "x"},
		Right: &core.Lit{Kind: core.IntLit, Value: int64(1)},
	}

	forallContract := &core.Forall{
		Var: "i",
		Lo:  &core.Lit{Kind: core.IntLit, Value: int64(0)},
		Hi:  &core.Var{Name: "x"},
		Body: &core.BinOp{
			Op:    ">=",
			Left:  &core.Var{Name: "i"},
			Right: &core.Lit{Kind: core.IntLit, Value: int64(0)},
		},
	}

	meta := &core.DeclMeta{
		Name:   "test",
		IsPure: true,
		Contracts: []*core.Contract{
			{
				Kind: core.EnsuresKind,
				Expr: forallContract,
			},
		},
	}

	encodable, reasons := IsSMTEncodable("test", meta, body)
	if !encodable {
		t.Errorf("expected function with forall contract to be SMT-encodable, got reasons: %v", reasons)
	}
}
