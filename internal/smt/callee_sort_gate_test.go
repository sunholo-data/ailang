package smt

import (
	"errors"
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// TestExtractDefineFunSigSorts checks that the signature parser extracts exactly the
// parameter sorts and the return sort — and nothing from the body — across the
// machine-generated (define-fun ...) shapes.
func TestExtractDefineFunSigSorts(t *testing.T) {
	tests := []struct {
		name string
		decl string
		want []string
	}{
		{
			name: "primitives",
			decl: "(define-fun f ((p_x Real) (p_target String)) Real (* p_x 2.0))",
			want: []string{"Real", "String", "Real"},
		},
		{
			name: "undeclared return sort",
			decl: "(define-fun convertTo ((p_x Real)) Option (Some p_x))",
			want: []string{"Real", "Option"},
		},
		{
			name: "seq param and return",
			decl: "(define-fun g ((p_xs (Seq Int))) (Seq Int) p_xs)",
			want: []string{"(Seq Int)", "(Seq Int)"},
		},
		{
			name: "no params",
			decl: "(define-fun k () Int 5)",
			want: []string{"Int"},
		},
		{
			name: "not a define-fun",
			decl: "(declare-datatype Foo ((A) (B)))",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDefineFunSigSorts(tt.decl)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("extractDefineFunSigSorts(%q) = %v, want %v", tt.decl, got, tt.want)
			}
		})
	}
}

// TestValidateDeclarationsRejectsUndeclaredDefineFunSort is the M2 defense-in-depth
// check: a callee define-fun that references an undeclared sort in its signature
// must fail with ErrUnresolvableTypes (→ graceful skip), not pass through to Z3.
func TestValidateDeclarationsRejectsUndeclaredDefineFunSort(t *testing.T) {
	ctx := NewSMTContext()
	decls := []string{
		"(define-fun convertTo ((p_x Real)) Option (Some p_x))",
	}
	err := validateDeclarations(decls, ctx)
	if err == nil {
		t.Fatal("expected ErrUnresolvableTypes for define-fun over undeclared sort 'Option', got nil")
	}
	if !errors.Is(err, ErrUnresolvableTypes) {
		t.Fatalf("expected ErrUnresolvableTypes, got %v", err)
	}
}

// TestValidateDeclarationsAcceptsPrimitiveDefineFunSort confirms M2 does not
// false-positive on a callee define-fun whose signature is all primitives.
func TestValidateDeclarationsAcceptsPrimitiveDefineFunSort(t *testing.T) {
	ctx := NewSMTContext()
	decls := []string{
		"(define-fun f ((p_x Real) (p_target String)) Real (* p_x 2.0))",
	}
	if err := validateDeclarations(decls, ctx); err != nil {
		t.Fatalf("unexpected error for all-primitive signature: %v", err)
	}
}

// TestLeakGuard_UnresolvedUserFunctionErrors is the leak-site guard: when encodeApp
// meets a call to a KNOWN user function that was neither resolved as a define-fun nor
// contract-substituted, it must return ErrUnresolvableTypes (→ graceful skip) rather
// than emit a raw uninterpreted symbol that makes Z3 hard-error with "unknown constant".
func TestLeakGuard_UnresolvedUserFunctionErrors(t *testing.T) {
	// Simulate the encodeApp environment: `canon` is a known user function but was
	// NOT resolved (empty resolved/contract maps).
	activeUserFunctions = map[string]bool{"canon": true}
	activeResolvedCallees = map[string]bool{}
	activeContractCallees = map[string]string{}
	defer func() {
		activeUserFunctions = nil
		activeResolvedCallees = nil
		activeContractCallees = nil
	}()

	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "m", Name: "canon"}},
		Args: []core.CoreExpr{&core.Lit{Kind: core.StringLit, Value: "x"}},
	}
	_, err := encodeApp(app)
	if err == nil {
		t.Fatal("expected ErrUnresolvableTypes for unresolved user function 'canon', got nil")
	}
	if !errors.Is(err, ErrUnresolvableTypes) {
		t.Fatalf("expected ErrUnresolvableTypes, got %v", err)
	}
}

// TestLeakGuard_RealConstructorStillEncodes confirms the guard does NOT misfire on a
// real ADT constructor application (a name that is not a known user function).
func TestLeakGuard_RealConstructorStillEncodes(t *testing.T) {
	activeUserFunctions = map[string]bool{"someUserFunc": true}
	activeResolvedCallees = map[string]bool{}
	activeContractCallees = map[string]string{}
	defer func() {
		activeUserFunctions = nil
		activeResolvedCallees = nil
		activeContractCallees = nil
	}()

	// `Some` is a constructor, not in activeUserFunctions → must NOT trip the guard.
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "m", Name: "Some"}},
		Args: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(1)}},
	}
	_, err := encodeApp(app)
	if errors.Is(err, ErrUnresolvableTypes) {
		t.Fatalf("guard misfired on constructor 'Some': %v", err)
	}
}

// TestValidateDeclarationsAcceptsDeclaredDefineFunSort confirms a define-fun over a
// sort that IS declared (in the same batch) passes.
func TestValidateDeclarationsAcceptsDeclaredDefineFunSort(t *testing.T) {
	ctx := NewSMTContext()
	decls := []string{
		"(declare-datatype Region ((DOMESTIC) (INTERNATIONAL)))",
		"(define-fun costOf ((p_r Region)) Int 5)",
	}
	if err := validateDeclarations(decls, ctx); err != nil {
		t.Fatalf("unexpected error for declared ADT signature: %v", err)
	}
}
