package smt

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/core"
	"strings"
	"testing"
)

func withFiniteSpec(t *testing.T, spec ContractSpec) {
	t.Helper()
	oldSpecs, oldCallees := activeContractSpecs, activeContractCallees
	activeContractSpecs = map[string]*ContractSpec{spec.FuncName: &spec}
	activeContractCallees = map[string]string{spec.FuncName: contractSymbol(spec)}
	t.Cleanup(func() { activeContractSpecs = oldSpecs; activeContractCallees = oldCallees })
}
func specCall(arg core.CoreExpr) core.CoreExpr {
	return &core.App{Func: &core.Var{Name: "bounded"}, Args: []core.CoreExpr{arg}}
}
func checkFiniteGoal(t *testing.T, body core.CoreExpr, want SolverStatus) {
	t.Helper()
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}
	value, facts, err := captureCallFacts(func() (string, error) { return EncodeExpr(body) })
	if err != nil {
		t.Fatal(err)
	}
	decl, err := declareContractFunction(finiteSpec())
	if err != nil {
		t.Fatal(err)
	}
	script := fmt.Sprintf("(set-logic ALL)\n%s\n(assert %s)\n(assert (not %s))\n(check-sat)\n", decl, facts, value)
	r, err := Solve(script, DefaultSolverConfig())
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != want {
		t.Fatalf("got %s want %s\n%s\n%s", r.Status, want, script, r.RawOutput)
	}
}
func TestFiniteFacts_LexicalAndSelected(t *testing.T) {
	withFiniteSpec(t, finiteSpec())
	yes := builtinApp("ge_Int", specCall(&core.Var{Name: "local"}), &core.Var{Name: "local"})
	body := &core.Let{Name: "local", Value: intLit(2), Body: yes}
	checkFiniteGoal(t, body, StatusVerified)
	_, facts, err := captureCallFacts(func() (string, error) { return EncodeExpr(body) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(facts, "(let ((local 2))") {
		t.Fatal(facts)
	}
}
func TestFiniteFacts_UnselectedContradiction(t *testing.T) {
	spec := finiteSpec()
	spec.Requires = nil
	spec.Ensures[0].Expr = &core.Lit{Kind: core.BoolLit, Value: false}
	withFiniteSpec(t, spec)
	// A contradictory contract in an unselected RHS must not prove a false goal.
	body := &core.If{Cond: &core.Lit{Kind: core.BoolLit, Value: false}, Then: builtinApp("eq_Int", specCall(intLit(1)), intLit(0)), Else: &core.Lit{Kind: core.BoolLit, Value: false}}
	checkFiniteGoal(t, body, StatusCounterexample)
}
func TestFiniteFacts_ShadowedOuterParameter(t *testing.T) {
	withFiniteSpec(t, finiteSpec())
	old := activeParamRenames
	activeParamRenames = map[string]string{"local": "$p_local"}
	defer func() { activeParamRenames = old }()
	checkFiniteGoal(t, &core.Let{Name: "local", Value: intLit(2), Body: builtinApp("ge_Int", specCall(&core.Var{Name: "local"}), intLit(2))}, StatusVerified)
}
