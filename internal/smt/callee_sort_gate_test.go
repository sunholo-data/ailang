package smt

import (
	"errors"
	"reflect"
	"testing"
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
