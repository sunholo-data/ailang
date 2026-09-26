package smt

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// M-SMT-INTERP-SHOW M3: a `show` that survives normalization is on a type with
// no SMT encoding. The skip message must name that TYPE — which the normalizer
// recorded in DeclMeta — and must NOT claim the call came from interpolation,
// because the same Core shape comes from an explicit user `show(x)` call and
// nothing in Core tells them apart.

func showRejection(t *testing.T, meta *core.DeclMeta, body core.CoreExpr) SMTRejectionReason {
	t.Helper()
	_, reasons := IsSMTEncodable("f", meta, body)
	for _, r := range reasons {
		if strings.Contains(r.Message, "show") {
			return r
		}
	}
	t.Fatalf("no show-related rejection in %v", reasons)
	return SMTRejectionReason{}
}

// showBody builds a contracted function body that applies $builtin.show.
func showBody() (*core.DeclMeta, core.CoreExpr) {
	arg := &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "x"}
	fn := &core.VarGlobal{
		CoreNode: core.CoreNode{NodeID: 2},
		Ref:      core.GlobalRef{Module: "$builtin", Name: "show"},
	}
	body := &core.App{CoreNode: core.CoreNode{NodeID: 3}, Func: fn, Args: []core.CoreExpr{arg}}
	meta := &core.DeclMeta{
		Name:      "f",
		IsPure:    true,
		Contracts: []*core.Contract{{Kind: core.EnsuresKind}},
	}
	return meta, body
}

// TestShowRejection_NamesTheRecordedType: with a note present, the message
// states the measured argument type.
func TestShowRejection_NamesTheRecordedType(t *testing.T) {
	meta, body := showBody()
	meta.ShowResidue = []core.ShowResidueNote{{ArgType: "float"}}

	r := showRejection(t, meta, body)
	if !strings.Contains(r.Message, "float") {
		t.Errorf("message %q does not name the recorded argument type", r.Message)
	}
	for _, forbidden := range []string{"interpolat", "${"} {
		if strings.Contains(strings.ToLower(r.Message), forbidden) {
			t.Errorf("message %q claims an origin it cannot know; that belongs in the hint as advice", r.Message)
		}
	}
	if !strings.Contains(r.Hint, "${") {
		t.Errorf("hint %q should mention that \"${x}\" desugars to show(x) — that is the part readers need", r.Hint)
	}
	for _, want := range []string{"string", "bool", "int"} {
		if !strings.Contains(r.Message, want) {
			t.Errorf("message %q does not tell the reader which show arguments ARE encodable (missing %q)", r.Message, want)
		}
	}
}

// TestShowRejection_NoNoteFallsBack: with no note — e.g. the pass disabled —
// the message must fall back to the type-free wording rather than guess.
func TestShowRejection_NoNoteFallsBack(t *testing.T) {
	meta, body := showBody()
	r := showRejection(t, meta, body)
	if strings.Contains(r.Message, "float") || strings.Contains(r.Message, "applies show to a") {
		t.Errorf("message %q invented a type with no note recorded", r.Message)
	}
	if !strings.Contains(r.Message, "unencodable builtin: show") {
		t.Errorf("message %q is not the type-free fallback", r.Message)
	}
}
