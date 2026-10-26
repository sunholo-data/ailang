package smt

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// #1961 (M-SMT-WELL-SORTED-PREFLIGHT M0, S2): a callee seen only through its
// contract is one uninterpreted function, `<f>__spec`, plus one axiom
//
//	forall params. requires(params) => ensures(params, result := <f>__spec(params))
//
// triggered by the call term. Every distinct call therefore gets its own result,
// constrained at its own arguments, and `f(a) - f(a)` is still 0. The old encoding
// shared one constant across all calls, so `f(a) - f(a + 1)` "proved" 0.

func builtinCall(name string, args ...core.CoreExpr) core.CoreExpr {
	return &core.App{Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: name}}, Args: args}
}

func ensuresContract(e core.CoreExpr) *core.Contract {
	return &core.Contract{Kind: core.EnsuresKind, Expr: e}
}

func requiresContract(e core.CoreExpr) *core.Contract {
	return &core.Contract{Kind: core.RequiresKind, Expr: e}
}

// totalSpec: total(n: int) -> int ensures { result >= 0 }
func totalSpec() ContractSpec {
	return ContractSpec{
		FuncName:   "total",
		Params:     []FunctionParam{{Name: "n", Type: types.TInt}},
		ReturnSort: "Int",
		Ensures:    []*core.Contract{ensuresContract(builtinCall("ge_Int", &core.Var{Name: "result"}, intLit(0)))},
	}
}

// lowBoundSpec: lowBound(n: int) -> int requires { n >= 0 } ensures { result >= n }
func lowBoundSpec() ContractSpec {
	return ContractSpec{
		FuncName:   "lowBound",
		Params:     []FunctionParam{{Name: "n", Type: types.TInt}},
		ReturnSort: "Int",
		Requires:   []*core.Contract{requiresContract(builtinCall("ge_Int", &core.Var{Name: "n"}, intLit(0)))},
		Ensures:    []*core.Contract{ensuresContract(builtinCall("ge_Int", &core.Var{Name: "result"}, &core.Var{Name: "n"}))},
	}
}

func mustEncodeContract(t *testing.T, spec ContractSpec) *ContractCalleeDecl {
	t.Helper()
	decl, err := EncodeCalleeContract(spec)
	if err != nil {
		t.Fatalf("EncodeCalleeContract(%s): %v", spec.FuncName, err)
	}
	return decl
}

func TestEncodeCalleeContract_Shape(t *testing.T) {
	decl := mustEncodeContract(t, lowBoundSpec())
	if decl.SpecFun != "lowBound__spec" || decl.Arity != 1 {
		t.Fatalf("SpecFun/Arity = %q/%d, want lowBound__spec/1", decl.SpecFun, decl.Arity)
	}
	for _, want := range []string{
		"(declare-fun lowBound__spec (Int) Int)",
		"(forall (($q_lowBound_n Int))",
		":pattern ((lowBound__spec $q_lowBound_n))",
		"(=> (>= $q_lowBound_n 0) (>= (lowBound__spec $q_lowBound_n) $q_lowBound_n))",
	} {
		if !strings.Contains(decl.SMTLib, want) {
			t.Errorf("SMTLib is missing %q:\n%s", want, decl.SMTLib)
		}
	}
	// No callee parameter is left free (bucket B3), and no shared result constant.
	if strings.Contains(decl.SMTLib, " n)") || strings.Contains(decl.SMTLib, "xmod_result") {
		t.Errorf("SMTLib leaks a free parameter or a shared constant:\n%s", decl.SMTLib)
	}
}

// TestEncodeCalleeContract_CallerRenameDoesNotCapture: the caller's own `$p_`
// parameter renames are active while the caller encodes; a callee parameter
// with the same name must still bind to the quantifier, not the caller's constant.
func TestEncodeCalleeContract_CallerRenameDoesNotCapture(t *testing.T) {
	prev := activeParamRenames
	activeParamRenames = map[string]string{"n": "$p_n"}
	defer func() { activeParamRenames = prev }()
	decl := mustEncodeContract(t, lowBoundSpec())
	if strings.Contains(decl.SMTLib, "$p_n") {
		t.Fatalf("callee parameter captured by the caller's rename:\n%s", decl.SMTLib)
	}
	if activeParamRenames["n"] != "$p_n" {
		t.Fatal("EncodeCalleeContract did not restore the caller's parameter renames")
	}
}

func TestEncodeCalleeContract_NullaryCallee(t *testing.T) {
	spec := ContractSpec{
		FuncName:   "limit",
		ReturnSort: "Int",
		Ensures:    []*core.Contract{ensuresContract(builtinCall("gt_Int", &core.Var{Name: "result"}, intLit(3)))},
	}
	decl := mustEncodeContract(t, spec)
	if decl.Arity != 0 {
		t.Fatalf("Arity = %d, want 0", decl.Arity)
	}
	if !strings.Contains(decl.SMTLib, "(declare-fun limit__spec () Int)") ||
		!strings.Contains(decl.SMTLib, "(assert (> limit__spec 3))") || strings.Contains(decl.SMTLib, "forall") {
		t.Fatalf("nullary callee should be a plain constant with a plain axiom:\n%s", decl.SMTLib)
	}
}

// TestEncodeCalleeContract_UnencodableRequiresDropsEnsures: the ensures holds only
// under the requires. If the requires cannot be encoded, the guard cannot be
// formed, so the ensures must not be assumed at all (weaker, sound).
func TestEncodeCalleeContract_UnencodableRequiresDropsEnsures(t *testing.T) {
	spec := lowBoundSpec()
	spec.Requires = []*core.Contract{requiresContract(builtinCall("no_such_builtin", &core.Var{Name: "n"}))}
	decl := mustEncodeContract(t, spec)
	if !strings.Contains(decl.SMTLib, "(declare-fun lowBound__spec (Int) Int)") {
		t.Fatalf("callee should still be declared:\n%s", decl.SMTLib)
	}
	if strings.Contains(decl.SMTLib, "assert") {
		t.Fatalf("ensures asserted without its requires guard:\n%s", decl.SMTLib)
	}
}

// TestEncodeCalleeContract_Z3Semantics runs the axioms against the #1961 shapes.
func TestEncodeCalleeContract_Z3Semantics(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}
	total := mustEncodeContract(t, totalSpec()).SMTLib
	low := mustEncodeContract(t, lowBoundSpec()).SMTLib
	cases := []struct {
		name, decls, goal string
		want              SolverStatus
	}{
		{"distinct calls are distinct (gap)", total, "(assert (>= a 0)) (assert (not (= (- (total__spec a) (total__spec (+ a 1))) 0)))", StatusCounterexample},
		{"same call is the same value (selfGap)", total, "(assert (not (= (- (total__spec a) (total__spec a)) 0)))", StatusVerified},
		{"ensures holds per call (sumBoth)", total, "(assert (not (>= (+ (total__spec a) (total__spec (+ a 1))) 0)))", StatusVerified},
		{"parameter instantiated with the argument (atLeast)", low, "(assert (>= a 0)) (assert (not (>= (lowBound__spec a) a)))", StatusVerified},
		{"requires guards the ensures (atLeastUnguarded)", low, "(assert (not (>= (lowBound__spec a) a)))", StatusCounterexample},
	}
	for _, c := range cases {
		script := "(set-logic ALL)\n" + c.decls + "\n(declare-const a Int)\n" + c.goal + "\n(check-sat)\n"
		res, err := Solve(script, DefaultSolverConfig())
		if err != nil {
			t.Fatalf("%s: solve: %v", c.name, err)
		}
		if res.Status != c.want {
			t.Errorf("%s: Z3 %v, want %v\n%s", c.name, res.Status, c.want, script)
		}
	}
}

// TestBuildContractSpec verifies contract classification (requires vs ensures).
func TestBuildContractSpec(t *testing.T) {
	meta := &core.DeclMeta{
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: &core.Var{Name: "n"}, Message: "pre"},
			{Kind: core.EnsuresKind, Expr: &core.Var{Name: "result"}, Message: "post"},
		},
	}
	params := []FunctionParam{{Name: "n", Type: types.TInt}}
	spec := BuildContractSpec("f", meta, params, "Int")

	if len(spec.Requires) != 1 {
		t.Errorf("expected 1 requires, got %d", len(spec.Requires))
	}
	if len(spec.Ensures) != 1 {
		t.Errorf("expected 1 ensures, got %d", len(spec.Ensures))
	}
	if spec.ReturnSort != "Int" {
		t.Errorf("expected return sort Int, got %q", spec.ReturnSort)
	}
}

// TestResolveCallees_ContractFallback verifies that a recursive imported callee
// (which cannot be inlined as a define-fun) is emitted via contract-as-spec: an
// uninterpreted <f>__spec function plus its axiom, with IsContract=true.
func TestResolveCallees_ContractFallback(t *testing.T) {
	// Imported lib: recursive(n) — self-recursive, cannot be inlined.
	// It has an ensures { result >= 0 } contract.
	importedProg := &core.Program{
		Meta: map[string]*core.DeclMeta{
			"recursive": {
				IsPure:    true,
				Contracts: []*core.Contract{ensuresContract(builtinCall("ge_Int", &core.Var{Name: "result"}, intLit(0)))},
			},
		},
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "recursive",
				Value: &core.Lambda{
					Params: []string{"n"},
					// Self-recursive: recursive(n) — IsSMTEncodableForCallee will reject
					Body: &core.App{
						Func: &core.Var{Name: "recursive"},
						Args: []core.CoreExpr{&core.Var{Name: "n"}},
					},
				},
				Body: &core.Var{Name: "recursive"},
			},
		},
	}

	// Main prog: caller(n) calls recursive(n)
	mainProg := &core.Program{
		Meta: map[string]*core.DeclMeta{
			"caller": {IsPure: true},
		},
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "caller",
				Value: &core.Lambda{
					Params: []string{"n"},
					Body: &core.App{
						Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "mylib", Name: "recursive"}},
						Args: []core.CoreExpr{&core.Var{Name: "n"}},
					},
				},
				Body: &core.Var{Name: "caller"},
			},
		},
	}

	importedPrograms := map[string]*core.Program{"mylib": importedProg}
	surfaceParams := map[string][]FunctionParam{
		"recursive": {{Name: "n", Type: types.TInt}},
		"caller":    {{Name: "n", Type: types.TInt}},
	}
	surfaceReturnSorts := map[string]string{
		"recursive": "Int",
		"caller":    "Int",
	}

	callerBody, _ := findFuncBodyInAnyProg("caller", mainProg, importedPrograms)
	_, innerBody := unwrapLambda(callerBody)

	defs, err := ResolveCallees("caller", innerBody, mainProg, surfaceParams, surfaceReturnSorts, nil, importedPrograms)
	if err != nil {
		t.Fatalf("ResolveCallees failed: %v", err)
	}

	if len(defs) != 1 {
		t.Fatalf("expected 1 callee def, got %d: %v", len(defs), defs)
	}
	d := defs[0]
	if d.Name != "recursive" || !d.IsContract {
		t.Fatalf("want contract callee 'recursive', got %+v", d)
	}
	if d.SpecFun != "recursive__spec" || d.Arity != 1 {
		t.Errorf("SpecFun/Arity = %q/%d, want recursive__spec/1", d.SpecFun, d.Arity)
	}
	if !strings.Contains(d.SMTLib, "(declare-fun recursive__spec (Int) Int)") || !strings.Contains(d.SMTLib, "forall") {
		t.Errorf("expected declare-fun + quantified axiom in SMTLib:\n%s", d.SMTLib)
	}
}

// TestEncodeApp_ContractCalleePerCallSite: each call encodes as an application of
// the spec function to that call's arguments, never as a shared constant.
func TestEncodeApp_ContractCalleePerCallSite(t *testing.T) {
	prev := activeContractCallees
	activeContractCallees = map[string]contractCalleeRef{"total": {SpecFun: "total__spec", Arity: 1}}
	defer func() { activeContractCallees = prev }()

	call := func(arg core.CoreExpr) core.CoreExpr {
		return &core.App{Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "m", Name: "total"}}, Args: []core.CoreExpr{arg}}
	}
	gap := builtinCall("sub_Int", call(&core.Var{Name: "a"}), call(builtinCall("add_Int", &core.Var{Name: "a"}, intLit(1))))
	got, err := EncodeExpr(gap)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if got != "(- (total__spec a) (total__spec (+ a 1)))" {
		t.Fatalf("got %s", got)
	}

	// An arity mismatch must skip (ErrUnresolvableTypes), never emit an ill-sorted term.
	bad := &core.App{Func: &core.Var{Name: "total"}, Args: []core.CoreExpr{intLit(1), intLit(2)}}
	if _, err := EncodeExpr(bad); err == nil {
		t.Fatal("arity mismatch encoded without error")
	}
}
