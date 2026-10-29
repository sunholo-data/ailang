package smt

import (
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
	"strings"
	"testing"
)

func helperProgram() (*core.Program, map[string][]FunctionParam, map[string]string) {
	spec := finiteSpec()
	prog := &core.Program{Meta: map[string]*core.DeclMeta{}}
	params := map[string][]FunctionParam{"bounded": spec.Params}
	sorts := map[string]string{"bounded": "Int"}
	add := func(name string, ps []FunctionParam, body core.CoreExpr) {
		var names []string
		for _, p := range ps {
			names = append(names, p.Name)
		}
		prog.Decls = append(prog.Decls, &core.Let{Name: name, Value: &core.Lambda{Params: names, Body: body}, Body: &core.Var{Name: name}})
		prog.Meta[name] = &core.DeclMeta{IsPure: true}
		params[name] = ps
		sorts[name] = "Int"
	}
	add("bounded", spec.Params, specCall(&core.Var{Name: "n"}))
	prog.Meta["bounded"].Contracts = append(spec.Requires, spec.Ensures...)
	ps := []FunctionParam{{Name: "environment", Type: types.TInt}, {Name: "n", Type: types.TInt}}
	add("helper", ps, specCall(builtinApp("add_Int", &core.Var{Name: "environment"}, &core.Var{Name: "n"})))
	for _, name := range []string{"left", "right"} {
		add(name, ps, &core.App{Func: &core.Var{Name: "helper"}, Args: []core.CoreExpr{&core.Var{Name: "environment"}, &core.Var{Name: "n"}}})
	}
	return prog, params, sorts
}
func TestFiniteFacts_HelperEnvironmentAndDiamond(t *testing.T) {
	prog, params, sorts := helperProgram()
	body := builtinApp("add_Int", &core.App{Func: &core.Var{Name: "left"}, Args: []core.CoreExpr{intLit(1), intLit(2)}}, &core.App{Func: &core.Var{Name: "right"}, Args: []core.CoreExpr{intLit(7), intLit(2)}})
	defs, err := ResolveCallees("root", body, prog, params, sorts, nil)
	if err != nil {
		t.Fatal(err)
	}
	var all string
	for _, d := range defs {
		all += d.SMTLib + "\n"
	}
	if strings.Count(all, "(define-fun $facts_helper ") != 1 {
		t.Fatal(all)
	}
	signature := "((environment Int) (n Int))"
	if !strings.Contains(all, "(define-fun helper "+signature+" Int") || !strings.Contains(all, "(define-fun $facts_helper "+signature+" Bool") {
		t.Fatal(all)
	}
	if !strings.Contains(all, "($facts_helper environment n)") {
		t.Fatal(all)
	}
}
func TestParseModel_ContractFunctionsAreNotBindings(t *testing.T) {
	rows := parseModel("sat\n(\n(define-fun $spec_bounded ((x Int)) Int\n  (+ x 1))\n(define-fun input () Int\n  2)\n)")
	if len(rows) != 1 || rows[0].Name != "input" || rows[0].Value != "2" {
		t.Fatalf("%+v", rows)
	}
}

func TestFiniteFacts_HelperDistinctEnvironments(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}
	prog, params, sorts := helperProgram()
	body := builtinApp("add_Int", &core.App{Func: &core.Var{Name: "left"}, Args: []core.CoreExpr{intLit(1), intLit(2)}}, &core.App{Func: &core.Var{Name: "right"}, Args: []core.CoreExpr{intLit(7), intLit(2)}})
	for _, goal := range []struct {
		op    string
		bound int64
		want  SolverStatus
	}{{"ge_Int", 12, StatusVerified}, {"eq_Int", 0, StatusCounterexample}} {
		meta := &core.DeclMeta{Contracts: []*core.Contract{{Kind: core.EnsuresKind, Expr: builtinApp(goal.op, &core.Var{Name: "result"}, intLit(goal.bound))}}}
		encoded, err := EncodeFunction("root", nil, body, "Int", meta, nil, EncodeFunctionOpts{Program: prog, SurfaceParams: params, SurfaceReturnSorts: sorts})
		if err != nil {
			t.Fatal(err)
		}
		r, err := Solve(encoded.SMTLib, DefaultSolverConfig())
		if err != nil {
			t.Fatal(err)
		}
		if r.Status != goal.want {
			t.Fatalf("got %s want %s\n%s\n%s", r.Status, goal.want, encoded.SMTLib, r.RawOutput)
		}
	}
}
