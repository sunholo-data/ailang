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
		{"missing", nil, false, true}, // rowless, no function parameter: same guarantees as ! {}
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

// A rowless function (no keyword, no `!`) is admitted unless a parameter type
// mentions a function type anywhere: a callback could carry its effect row
// into the function's own.
func TestSMTAdmissionRowlessCallbackParams(t *testing.T) {
	fn := &ast.FuncType{Params: []ast.Type{&ast.SimpleType{Name: "int"}}, Return: &ast.SimpleType{Name: "int"}}
	for _, tc := range []struct {
		name  string
		param ast.Type
		want  bool
	}{
		{"int", &ast.SimpleType{Name: "int"}, true},
		{"named", &ast.SimpleType{Name: "Json"}, true},
		{"Option[int]", &ast.TypeApp{Constructor: "Option", Args: []ast.Type{&ast.SimpleType{Name: "int"}}}, true},
		{"callback", fn, false},
		{"list of callbacks", &ast.ListType{Element: fn}, false},
		{"array of callbacks", &ast.ArrayType{Element: fn}, false},
		{"Option[callback]", &ast.TypeApp{Constructor: "Option", Args: []ast.Type{fn}}, false},
		{"tuple with callback", &ast.TupleType{Elements: []ast.Type{&ast.SimpleType{Name: "int"}, fn}}, false},
		{"record field callback", &ast.RecordType{Fields: []*ast.RecordField{{Name: "act", Type: fn}}}, false},
		{"labelled callback", &ast.LabelledType{Base: fn}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			meta := &core.DeclMeta{Contracts: []*core.Contract{{Kind: core.EnsuresKind}}}
			fd := &ast.FuncDecl{Params: []*ast.Param{{Name: "p", Type: tc.param}}}
			if got := smtAdmissible(meta, fd); got != tc.want {
				t.Fatalf("rowless f(p: %s) admitted=%v, want %v", tc.name, got, tc.want)
			}
			fd.Effects = []ast.EffectAnnotation{}
			if !smtAdmissible(meta, fd) {
				t.Fatalf("f(p: %s) ! {} refused: an explicit row is admitted whatever the params", tc.name)
			}
		})
	}
}
