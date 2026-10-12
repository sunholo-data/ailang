package smt

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"testing"
)

func TestSMTAdmissionRows(t *testing.T) {
	for _, tc := range []struct {
		name       string
		effects    []ast.EffectAnnotation
		pure, want bool
	}{
		{"keyword", nil, true, true},
		{"missing", nil, false, false},
		{"empty", []ast.EffectAnnotation{}, false, true},
		{"declassify", []ast.EffectAnnotation{{Name: "Declassify"}}, false, true},
		{"scoped", []ast.EffectAnnotation{{Name: "Declassify", Params: []ast.EffectParam{{Key: "label", Value: "x"}}}}, false, true},
		{"io", []ast.EffectAnnotation{{Name: "IO"}}, false, false},
		{"debug", []ast.EffectAnnotation{{Name: "Debug"}}, false, false},
		{"mixed", []ast.EffectAnnotation{{Name: "Declassify"}, {Name: "IO"}}, false, false},
		{"scopedMixed", []ast.EffectAnnotation{{Name: "Declassify", Params: []ast.EffectParam{{Key: "label", Value: "x"}}}, {Name: "IO"}}, false, false},
		{"open", []ast.EffectAnnotation{{Name: "e"}}, false, false},
		{"scopedOpen", []ast.EffectAnnotation{{Name: "Declassify", Params: []ast.EffectParam{{Key: "label", Value: "x"}}}, {Name: "e"}}, false, false},
		{"mixedOpen", []ast.EffectAnnotation{{Name: "Declassify"}, {Name: "e"}}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			meta := &core.DeclMeta{IsPure: tc.pure, Contracts: []*core.Contract{{Kind: core.EnsuresKind}}}
			fd := &ast.FuncDecl{IsPure: tc.pure, Effects: tc.effects}
			body := &core.Lit{Kind: core.IntLit, Value: int64(1)}
			entry, _ := IsSMTEncodable("f", meta, body, fd)
			callee, _ := IsSMTEncodableForCallee("f", meta, body, fd)
			if entry != tc.want || callee != tc.want {
				t.Fatalf("entry=%v callee=%v want=%v", entry, callee, tc.want)
			}
			if meta.IsPure != tc.pure {
				t.Fatal("admission mutated purity")
			}
		})
	}
}

func TestSMTAdmissionMissingProvenance(t *testing.T) {
	if smtAdmissible(nil) {
		t.Fatal("nil metadata admitted")
	}
	if smtAdmissible(&core.DeclMeta{}) {
		t.Fatal("missing declaration admitted")
	}
	if !smtAdmissible(&core.DeclMeta{IsPure: true}) {
		t.Fatal("keyword-only metadata refused")
	}
}

func TestSMTCalleeAdmissionCannotUseContractFallback(t *testing.T) {
	meta := pureMetaWithContracts()
	meta.IsPure = false
	helper := &core.Lambda{Params: []string{"x"}, Body: &core.Var{Name: "x"}}
	body := &core.App{Func: &core.VarGlobal{Ref: core.GlobalRef{Name: "helper"}}, Args: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(1)}}}
	prog := makeTestProgram(map[string]core.CoreExpr{"helper": helper}, map[string]*core.DeclMeta{"helper": meta})
	declarations := map[*core.DeclMeta]*ast.FuncDecl{meta: {Effects: []ast.EffectAnnotation{{Name: "IO"}}}}
	defs, err := resolveCalleesFromRoots("caller", []core.CoreExpr{body}, prog, nil, nil, nil, declarations)
	if err == nil || len(defs) != 0 {
		t.Fatalf("inadmissible callee fell back to contract: defs=%+v err=%v", defs, err)
	}
}
