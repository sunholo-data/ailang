package iface

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/messaging"
)

func gtZero(name string) ast.Expr {
	return &ast.BinaryOp{Left: &ast.Identifier{Name: name}, Op: ">", Right: &ast.Literal{Kind: ast.IntLit, Value: int64(0)}}
}

// FuncContracts keeps requires/ensures in source order, renders them without
// positions, and ignores property tests and invariants.
func TestFuncContracts_NormalisedAndFiltered(t *testing.T) {
	fd := &ast.FuncDecl{Properties: []*ast.Property{
		{Kind: ast.EnsuresKind, Expr: gtZero("result"), Pos: ast.Pos{Line: 9}},
		{Kind: ast.PropertyKind, Name: "p", Expr: gtZero("x")},
		{Kind: ast.RequiresKind, Expr: gtZero("n"), Pos: ast.Pos{Line: 3}},
		{Kind: ast.InvariantKind, Expr: gtZero("y")},
		nil,
		{Kind: ast.RequiresKind},
	}}
	got := FuncContracts(fd)
	want := []string{"ensures (result > 0)", "requires (n > 0)"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FuncContracts = %#v, want %#v", got, want)
	}
	moved := &ast.FuncDecl{Properties: []*ast.Property{{Kind: ast.EnsuresKind, Expr: gtZero("result"), Pos: ast.Pos{Line: 40, Column: 7}}}}
	if FuncContracts(moved)[0] != want[0] {
		t.Error("a contract's string must not depend on its source position")
	}
	if FuncContracts(nil) != nil || FuncContracts(&ast.FuncDecl{}) != nil {
		t.Error("no contracts must yield nil")
	}
}

// Contract lines enter the signature set; the hash projection (and so
// interface_hash_v2) is byte-identical with or without them (D7).
func TestContracts_InSignaturesNotInHash(t *testing.T) {
	mk := func(contracts []string) *InterfaceJSON {
		return &InterfaceJSON{Module: "q/lib", Schema: "ailang.iface/v1",
			Funcs: []FuncJSON{{Name: "f", Type: "(int)->int", Effects: []string{}, Pure: true, Contracts: contracts}}}
	}
	plain, withC := mk(nil), mk([]string{"requires (n > 0)"})

	hp, err := HashProjection(plain)
	if err != nil {
		t.Fatal(err)
	}
	hc, err := HashProjection(withC)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(hp, hc) || strings.Contains(string(hc), "contracts") {
		t.Fatalf("contracts must not reach the hash projection:\n%s\n%s", hp, hc)
	}

	sp, sc := SignatureSet(plain), SignatureSet(withC)
	if !reflect.DeepEqual(sp, []string{ContractsMarker, "q/lib:func:f:(int)->int:"}) {
		t.Errorf("plain set = %#v", sp)
	}
	wantLine := "q/lib:contract:f:requires (n > 0)"
	found := false
	for _, s := range sc {
		found = found || s == wantLine
	}
	if !found {
		t.Errorf("contract line %q missing from %#v", wantLine, sc)
	}
	// A comma in a clause is escaped like any other field.
	commas := withoutContractsMarker(SignatureSet(mk([]string{"requires f(a, b)"})))
	if commas[0] != `q/lib:contract:f:requires f(a\, b)` {
		t.Errorf("clause not escaped: %#v", commas)
	}
}

// The marker is defined twice so internal/messaging need not import iface.
func TestContractsMarker_MatchesMessaging(t *testing.T) {
	if ContractsMarker != messaging.SignatureContractsMarker {
		t.Fatalf("iface %q != messaging %q", ContractsMarker, messaging.SignatureContractsMarker)
	}
}

func withoutContractsMarker(set []string) []string {
	out := make([]string, 0, len(set))
	for _, s := range set {
		if s != ContractsMarker {
			out = append(out, s)
		}
	}
	return out
}
