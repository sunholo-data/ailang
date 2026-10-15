package types

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

func TestApplicationDiagnosticNames(t *testing.T) {
	for _, imported := range []bool{false, true} {
		t.Run(map[bool]string{false: "local", true: "imported"}[imported], func(t *testing.T) {
			tc := NewCoreTypeChecker()
			ctx := NewInferenceContext()
			fn := &TFunc2{Params: []Type{&TMap{Key: TString, Value: TInt}, TString}, Return: TInt, EffectRow: EmptyEffectRow()}
			var head core.CoreExpr = &core.Var{Name: "lookup"}
			if imported {
				head = &core.VarGlobal{Ref: core.GlobalRef{Module: "std/map", Name: "lookup"}}
				tc.SetGlobalType("std/map.lookup", &Scheme{Type: fn})
			} else {
				ctx.env.BindType("lookup", fn)
			}
			ctx.env.BindType("m", &TMap{Key: TString, Value: TInt})
			app := &core.App{CoreNode: core.CoreNode{CoreSpan: ast.Pos{File: "swapped.ail", Line: 6, Column: 3}}, Func: head, Args: []core.CoreExpr{&core.Lit{Kind: core.StringLit, Value: "b"}, &core.Var{Name: "m"}}}
			_, _, err := tc.inferApp(ctx, app)
			if err == nil {
				t.Fatal("swapped arguments accepted")
			}
			for _, want := range []string{"application of lookup", "swapped.ail:6:3", "failed to unify parameter 0"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("diagnostic %q missing %q", err, want)
				}
			}
			// The deferred solver must retain exactly the same callee context.
			_, _, deferredErr := ctx.SolveConstraints()
			if deferredErr == nil || deferredErr.Error() != err.Error() {
				t.Errorf("local=%v deferred=%v", err, deferredErr)
			}
		})
	}
}

func TestApplicationDiagnosticAnonymousAndArity(t *testing.T) {
	tc := NewCoreTypeChecker()
	ctx := NewInferenceContext()
	app := &core.App{Func: &core.Lambda{Params: []string{"x"}, Body: &core.Var{Name: "x"}}}
	_, _, err := tc.inferApp(ctx, app)
	if err == nil || !strings.Contains(err.Error(), "function application at") || strings.Contains(err.Error(), "application of") {
		t.Fatalf("anonymous diagnostic: %v", err)
	}
	ctx = NewInferenceContext()
	ctx.env.BindType("f", &TFunc2{Params: []Type{TInt}, Return: TInt, EffectRow: EmptyEffectRow()})
	_, _, err = tc.inferApp(ctx, &core.App{Func: &core.Var{Name: "f"}})
	if err == nil || !strings.Contains(err.Error(), arityMismatchMsg(1, 0)) {
		t.Fatalf("arity diagnostic changed: %v", err)
	}
	_, rawErr := NewUnifier().Unify(TInt, TString, Substitution{})
	if rawErr == nil || strings.Contains(rawErr.Error(), "application") {
		t.Fatalf("raw unification changed: %v", rawErr)
	}
}

func TestApplicationDiagnosticCollisionHint(t *testing.T) {
	orig := ImportSuggester
	defer func() { ImportSuggester = orig }()
	ImportSuggester = func(string) []string { return []string{"std/list", "std/string"} }
	tc := NewCoreTypeChecker()
	tc.SetGlobalType("std/list.length", &Scheme{Type: &TFunc2{Params: []Type{&TList{Element: TInt}}, Return: TInt, EffectRow: EmptyEffectRow()}})
	_, _, err := tc.inferApp(NewInferenceContext(), &core.App{Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/list", Name: "length"}}, Args: []core.CoreExpr{&core.Lit{Kind: core.StringLit, Value: "wrong"}}})
	if err == nil {
		t.Fatal("bad argument accepted")
	}
	for _, want := range []string{"application of length", "length as lengthAlt", "std/string"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("diagnostic %q missing %q", err, want)
		}
	}
}
