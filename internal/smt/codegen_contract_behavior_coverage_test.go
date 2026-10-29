package smt

import (
	"fmt"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// Exercise a real call specification against Z3, rather than the spelling of
// its emitted substitutions. Both entailment and non-entailment are controls.
func checkCoverageContract(t *testing.T, spec ContractSpec, args []core.CoreExpr, op string, bound int64, want SolverStatus) {
	t.Helper()
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}
	call, err := EncodeCalleeByContract(spec, args, 0)
	if err != nil {
		t.Fatal(err)
	}
	script := fmt.Sprintf("(set-logic ALL)\n%s\n(assert (not (%s %s %d)))\n(check-sat)\n", call.SMTLib, op, call.ResultConst, bound)
	result, err := Solve(script, DefaultSolverConfig())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != want {
		t.Fatalf("got %s want %s\n%s\n%s", result.Status, want, script, result.RawOutput)
	}
}

func TestContractBehavior_MatchParameterAndResultShadowing(t *testing.T) {
	spec := finiteSpec()
	spec.Requires = nil
	spec.Ensures[0].Expr = &core.Match{Scrutinee: &core.Var{Name: "n"}, Arms: []core.MatchArm{
		{Pattern: &core.LitPattern{Value: int64(0)}, Body: builtinApp("eq_Int", &core.Var{Name: "result"}, intLit(0))},
		{Pattern: &core.VarPattern{Name: "matched"}, Body: builtinApp("ge_Int", &core.Var{Name: "result"}, &core.Var{Name: "matched"})},
	}}
	checkCoverageContract(t, spec, []core.CoreExpr{intLit(5)}, ">=", 5, StatusVerified)
	checkCoverageContract(t, spec, []core.CoreExpr{intLit(5)}, ">=", 6, StatusCounterexample)
	// Here "result" is a pattern-bound value, not the callee result pseudo-var.
	spec.Ensures[0].Expr = &core.Match{Scrutinee: &core.Var{Name: "n"}, Arms: []core.MatchArm{
		{Pattern: &core.VarPattern{Name: "result"}, Guard: builtinApp("ge_Int", &core.Var{Name: "result"}, intLit(0)), Body: builtinApp("eq_Int", &core.Var{Name: "result"}, &core.Var{Name: "n"})},
		{Pattern: &core.WildcardPattern{}, Body: boolLit(true)},
	}}
	checkCoverageContract(t, spec, []core.CoreExpr{intLit(5)}, "=", 5, StatusCounterexample)
}

func TestContractBehavior_BoundedQuantifierSubstitution(t *testing.T) {
	spec := finiteSpec()
	spec.Requires = nil
	spec.Ensures[0].Expr = &core.Forall{Var: "i", Lo: intLit(0), Hi: intLit(2), Body: builtinApp("ge_Int", &core.Var{Name: "result"}, builtinApp("add_Int", &core.Var{Name: "n"}, &core.Var{Name: "i"}))}
	checkCoverageContract(t, spec, []core.CoreExpr{intLit(5)}, ">=", 6, StatusVerified)
	checkCoverageContract(t, spec, []core.CoreExpr{intLit(5)}, ">=", 7, StatusCounterexample)
	// A bounded binder named result must not constrain the function's UF value.
	spec.Ensures[0].Expr = &core.Forall{Var: "result", Lo: intLit(0), Hi: intLit(1), Body: builtinApp("eq_Int", &core.Var{Name: "result"}, intLit(0))}
	checkCoverageContract(t, spec, []core.CoreExpr{intLit(5)}, "=", 0, StatusCounterexample)
}

func TestContractBehavior_NullaryAndUnknownEnsures(t *testing.T) {
	spec := ContractSpec{FuncName: "nullary", Params: []FunctionParam{{Name: "unit", Type: types.TUnit}}, ReturnSort: "Int", Ensures: []*core.Contract{{Kind: core.EnsuresKind, Expr: builtinApp("eq_Int", &core.Var{Name: "result"}, intLit(3))}}}
	checkCoverageContract(t, spec, []core.CoreExpr{&core.Lit{Kind: core.UnitLit}}, "=", 3, StatusVerified)
	checkCoverageContract(t, spec, []core.CoreExpr{&core.Lit{Kind: core.UnitLit}}, "=", 4, StatusCounterexample)
	// Unsupported postconditions contribute no invented guarantee.
	spec.Ensures[0].Expr = &core.Lambda{Params: []string{"x"}, Body: &core.Var{Name: "x"}}
	checkCoverageContract(t, spec, []core.CoreExpr{&core.Lit{Kind: core.UnitLit}}, "=", 3, StatusCounterexample)
	spec.Ensures[0].Expr = &core.Let{Name: "result", Value: intLit(3), Body: builtinApp("eq_Int", &core.Var{Name: "result"}, intLit(3))}
	checkCoverageContract(t, spec, []core.CoreExpr{&core.Lit{Kind: core.UnitLit}}, "=", 3, StatusCounterexample)
}
