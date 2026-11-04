package types

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"strings"
	"testing"
)

func TestApplicationEffectDiagnosticParity(t *testing.T) {
	for _, lambda := range []bool{false, true} {
		tc := NewCoreTypeChecker()
		ctx := NewInferenceContext()
		pure := &TFunc2{Params: []Type{TString}, Return: TUnit, EffectRow: EmptyEffectRow()}
		effectful := &TFunc2{Params: []Type{TString}, Return: TUnit, EffectRow: &Row{Kind: EffectRow, Labels: map[string]Type{"IO": TUnit}}}
		ctx.env.BindType("map", &TFunc2{Params: []Type{pure}, Return: TUnit, EffectRow: EmptyEffectRow()})
		ctx.env.BindType("print", effectful)
		var arg core.CoreExpr = &core.Var{Name: "print"}
		if lambda {
			arg = &core.Lambda{Params: []string{"x"}, Body: &core.App{Func: &core.Var{Name: "print"}, Args: []core.CoreExpr{&core.Var{Name: "x"}}}}
		}
		_, _, err := tc.inferApp(ctx, &core.App{Func: &core.Var{Name: "map"}, Args: []core.CoreExpr{arg}})
		if err == nil {
			t.Fatal("accepted effectful callback")
		}
		word := "argument"
		if lambda {
			word = "lambda"
		}
		for _, want := range []string{"`map` requires parameter 0", "(pure)", "this " + word, "IO", "(solver: type unification failed", "failed to unify parameter 0"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%v missing %q", err, want)
			}
		}
		_, _, deferred := ctx.SolveConstraints()
		if deferred == nil || deferred.Error() != err.Error() {
			t.Fatalf("local %v deferred %v", err, deferred)
		}
	}
}

func TestEffectDiagnosticDecorationCases(t *testing.T) {
	row := func(labels ...string) *Row {
		r := EmptyEffectRow()
		for _, label := range labels {
			r.Labels[label] = TUnit
		}
		return r
	}
	raw := fmt.Errorf("original solver detail")
	for _, tc := range []struct {
		name         string
		app          *AppConstraintInfo
		mismatch     *EffectRowMismatchError
		want, absent string
	}{
		{"anonymous", &AppConstraintInfo{}, &EffectRowMismatchError{DeclaredRow: row(), ActualRow: row("IO"), ParamIdx: 0, cause: raw}, "this call requires", "std/list"},
		{"both", &AppConstraintInfo{}, &EffectRowMismatchError{DeclaredRow: row("Net"), ActualRow: row("IO", "FS"), ParamIdx: 0, cause: raw}, "performs {FS, IO}, missing {Net}", "(pure)"},
		{"own", &AppConstraintInfo{Callee: "map"}, &EffectRowMismatchError{DeclaredRow: row(), ActualRow: row("IO"), ParamIdx: -1, cause: raw}, "type unification failed at [function application of map", "requires parameter"},
		{"nested", &AppConstraintInfo{}, &EffectRowMismatchError{DeclaredRow: row(), ActualRow: row("IO"), ParamIdx: 0, nested: true, cause: raw}, "type unification failed", "requires parameter"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old := EffectfulVariantSuggester
			EffectfulVariantSuggester = nil
			defer func() { EffectfulVariantSuggester = old }()
			err := wrapTypeEqFailure(TypeEq{App: tc.app, Path: []string{"old path", "also exported by"}}, tc.mismatch)
			for _, want := range []string{tc.want, "original solver detail"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("%v missing %s", err, want)
				}
			}
			if strings.Contains(err.Error(), tc.absent) || strings.Contains(err.Error(), "also exported by") {
				t.Fatal(err)
			}
		})
	}
	constraint := TypeEq{Path: []string{"untouched", "also exported by"}}
	if got, want := wrapTypeEqFailure(constraint, raw).Error(), fmt.Sprintf("type unification failed at %v: %v", constraint.Path, raw); got != want {
		t.Fatalf("%s != %s", got, want)
	}
}

func TestApplicationConstraintContext(t *testing.T) {
	pos := ast.Pos{File: "context.ail", Line: 3, Column: 4}
	for _, tc := range []struct {
		head         core.CoreExpr
		name, module string
	}{
		{&core.Var{Name: "alias"}, "alias", ""},
		{&core.VarGlobal{Ref: core.GlobalRef{Name: "map", Module: "std/list"}}, "map", "std/list"},
		{&core.Lambda{}, "", ""},
	} {
		info := applicationConstraintInfo(&core.App{CoreNode: core.CoreNode{CoreSpan: pos}, Func: tc.head, Args: []core.CoreExpr{&core.Lambda{CoreNode: core.CoreNode{CoreSpan: pos}}, &core.Var{Name: "f"}}})
		if info.Callee != tc.name || info.FromModule != tc.module || info.Span != pos || len(info.ArgSpans) != 2 || info.ArgSpans[0] != pos || !info.ArgIsLambda[0] || info.ArgIsLambda[1] {
			t.Fatalf("bad context: %+v", info)
		}
	}
}
