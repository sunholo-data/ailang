package smt

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// TestIsSMTEncodable_NamesBlockingBuiltin verifies M4: when a function uses
// an unencodable builtin (e.g., `_str_trim`), the rejection message names
// the specific builtin so users can either narrow contracts or refactor.
//
// Before M4 the message was a generic "list, unsupported string operations".
// After M4 the message names the actual blocker.
func TestIsSMTEncodable_NamesBlockingBuiltin(t *testing.T) {
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_str_trim"}},
		Args: []core.CoreExpr{&core.Lit{Kind: core.StringLit, Value: " hello "}},
	}
	meta := &core.DeclMeta{
		Name:   "trimFoo",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.EnsuresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}

	encodable, reasons := IsSMTEncodable("trimFoo", meta, body)
	if encodable {
		t.Fatalf("expected unencodable, got encodable")
	}

	found := false
	for _, r := range reasons {
		if r.Code == RejectUnencodable && strings.Contains(r.Message, "_str_trim") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected rejection message naming _str_trim, got reasons: %+v", reasons)
	}
}

// TestIsSMTEncodable_GenericMessageWhenNoSpecificBuiltin keeps the old
// generic message when the unencodable shape isn't a named builtin (e.g., a
// `core.Array` literal which has no obvious named blocker).
func TestIsSMTEncodable_GenericMessageWhenNoSpecificBuiltin(t *testing.T) {
	body := &core.Array{Elements: []core.CoreExpr{}}
	meta := &core.DeclMeta{
		Name:   "arrFoo",
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.EnsuresKind, Expr: &core.Lit{Kind: core.BoolLit, Value: true}},
		},
	}
	encodable, reasons := IsSMTEncodable("arrFoo", meta, body)
	if encodable {
		t.Fatalf("expected unencodable, got encodable")
	}
	if len(reasons) == 0 {
		t.Fatalf("expected at least one rejection reason")
	}
}
