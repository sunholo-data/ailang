package smt

import (
	"github.com/sunholo-data/ailang/internal/core"
	"strings"
	"testing"
)

func boolLit(v bool) core.CoreExpr { return &core.Lit{Kind: core.BoolLit, Value: v} }
func TestFiniteFacts_MatchGuardReach(t *testing.T) {
	spec := finiteSpec()
	spec.Requires = nil
	spec.Ensures[0].Expr = boolLit(false)
	withFiniteSpec(t, spec)
	for _, arms := range [][]core.MatchArm{
		{{Pattern: &core.LitPattern{Value: int64(1)}, Guard: builtinApp("eq_Int", specCall(intLit(1)), intLit(0)), Body: boolLit(true)}, {Pattern: &core.WildcardPattern{}, Body: boolLit(false)}},
		{{Pattern: &core.WildcardPattern{}, Body: boolLit(false)}, {Pattern: &core.WildcardPattern{}, Guard: builtinApp("eq_Int", specCall(intLit(1)), intLit(0)), Body: boolLit(true)}},
	} {
		checkFiniteGoal(t, &core.Match{Scrutinee: intLit(0), Arms: arms}, StatusCounterexample)
	}
}
func TestFiniteFacts_MatchBoundPositive(t *testing.T) {
	withFiniteSpec(t, finiteSpec())
	checkFiniteGoal(t, &core.Match{Scrutinee: intLit(2), Arms: []core.MatchArm{{Pattern: &core.VarPattern{Name: "n"}, Guard: builtinApp("ge_Int", specCall(&core.Var{Name: "n"}), &core.Var{Name: "n"}), Body: boolLit(true)}, {Pattern: &core.WildcardPattern{}, Body: boolLit(false)}}}, StatusVerified)
}
func TestFiniteFacts_UnknownRequiresCannotLeakNestedFacts(t *testing.T) {
	spec := finiteSpec()
	spec.Requires[0].Expr = &core.Let{Name: "tmp", Value: specCall(intLit(1)), Body: &core.Lambda{Body: boolLit(true)}}
	withFiniteSpec(t, spec)
	_, facts, err := captureCallFacts(func() (string, error) { return EncodeExpr(specCall(intLit(2))) })
	if err != nil {
		t.Fatal(err)
	}
	if facts != "true" {
		t.Fatalf("unencodable requires leaked facts: %s", facts)
	}
}
func TestContractSubstitution_RespectsShadowing(t *testing.T) {
	spec := finiteSpec()
	spec.Requires = nil
	spec.Ensures[0].Expr = &core.Let{Name: "n", Value: intLit(3), Body: builtinApp("ge_Int", &core.Var{Name: "result"}, &core.Var{Name: "n"})}
	d, err := EncodeCalleeByContract(spec, []core.CoreExpr{intLit(1)}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(d.SMTLib, "(>= ($spec_bounded 1) 1)") || !strings.Contains(d.SMTLib, "_result n)") {
		t.Fatal(d.SMTLib)
	}
}

func TestContractSubstitution_ActualArgumentCapture(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}
	spec := finiteSpec()
	spec.Requires = nil
	spec.Ensures[0].Expr = &core.Let{Name: "caller", Value: intLit(0), Body: builtinApp("eq_Int", &core.Var{Name: "result"}, &core.Var{Name: "n"})}
	d, err := EncodeCalleeByContract(spec, []core.CoreExpr{&core.Var{Name: "caller"}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	script := "(set-logic ALL)\n(declare-const caller Int)\n(assert (= caller 1))\n" + d.SMTLib + "\n(assert (not (= " + d.ResultConst + " 1)))\n(check-sat)\n"
	r, err := Solve(script, DefaultSolverConfig())
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != StatusVerified {
		t.Fatalf("captured actual argument/result: %s\n%s", script, r.RawOutput)
	}
}

func TestContractSubstitution_GeneratedAliasCollision(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}
	spec := finiteSpec()
	spec.Requires = nil
	spec.Ensures[0].Expr = &core.Let{Name: "$spec_sub_0_0", Value: intLit(0), Body: builtinApp("eq_Int", &core.Var{Name: "result"}, &core.Var{Name: "n"})}
	d, err := EncodeCalleeByContract(spec, []core.CoreExpr{&core.Var{Name: "$spec_sub_1_result"}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	script := "(set-logic ALL)\n(declare-const $spec_sub_1_result Int)\n(assert (= $spec_sub_1_result 1))\n" + d.SMTLib + "\n(assert (not (= " + d.ResultConst + " 1)))\n(check-sat)\n"
	r, err := Solve(script, DefaultSolverConfig())
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != StatusVerified {
		t.Fatalf("generated alias capture: %s\n%s", script, r.RawOutput)
	}
}
func TestFiniteFacts_QuantifierBinderAndEmptyRange(t *testing.T) {
	withFiniteSpec(t, finiteSpec())
	body := builtinApp("ge_Int", specCall(&core.Var{Name: "n"}), &core.Var{Name: "n"})
	checkFiniteGoal(t, &core.Forall{Var: "n", Lo: intLit(0), Hi: intLit(3), Body: body}, StatusVerified)
	spec := finiteSpec()
	spec.Requires = nil
	spec.Ensures[0].Expr = boolLit(false)
	activeContractSpecs[spec.FuncName] = &spec
	empty := &core.Forall{Var: "n", Lo: intLit(0), Hi: intLit(0), Body: builtinApp("eq_Int", specCall(&core.Var{Name: "n"}), intLit(0))}
	checkFiniteGoal(t, &core.If{Cond: empty, Then: boolLit(false), Else: boolLit(true)}, StatusCounterexample)
}
