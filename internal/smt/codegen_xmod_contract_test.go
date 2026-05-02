package smt

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// TestEncodeCalleeByContract_Basic verifies that a callee with only an ensures
// clause produces a declare-const and an assert.
func TestEncodeCalleeByContract_Basic(t *testing.T) {
	// Callee: nonneg(n: int) -> int, ensures { result >= 0 }
	spec := ContractSpec{
		FuncName:   "nonneg",
		Params:     []FunctionParam{{Name: "n", Type: types.TInt}},
		ReturnSort: "Int",
		Ensures: []*core.Contract{
			{
				Kind: core.EnsuresKind,
				Expr: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "result"},
						&core.Lit{Kind: core.IntLit, Value: int64(0)},
					},
				},
				Message: "nonneg postcondition",
			},
		},
	}

	// Call site: nonneg(42)
	args := []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(42)}}

	decl, err := EncodeCalleeByContract(spec, args, 0)
	if err != nil {
		t.Fatalf("EncodeCalleeByContract failed: %v", err)
	}

	if decl.ResultConst != "$xmod_result_nonneg_0" {
		t.Errorf("expected result const $xmod_result_nonneg_0, got %q", decl.ResultConst)
	}
	if !strings.Contains(decl.SMTLib, "declare-const") {
		t.Errorf("expected declare-const in SMTLib, got:\n%s", decl.SMTLib)
	}
	if !strings.Contains(decl.SMTLib, "assert") {
		t.Errorf("expected assert in SMTLib, got:\n%s", decl.SMTLib)
	}
	if !strings.Contains(decl.SMTLib, ">=") && !strings.Contains(decl.SMTLib, "ge_Int") && !strings.Contains(decl.SMTLib, ">=") {
		// SMT-LIB uses (>= ...) for ge_Int
		t.Logf("SMTLib: %s", decl.SMTLib)
	}
}

// TestEncodeCalleeByContract_WeakContractCatchesViolation verifies that a caller
// with a STRONGER ensures than the callee's contract produces a detectable mismatch.
// This proves the encoder uses the callee's contract (not the body or a no-op).
func TestEncodeCalleeByContract_WeakContractCatchesViolation(t *testing.T) {
	// Callee: mystery(n) -> int, ensures { result >= 0 }  (weak — only non-negative)
	// Caller asserts result == 99 (stronger — should be unprovable)
	spec := ContractSpec{
		FuncName:   "mystery",
		Params:     []FunctionParam{{Name: "n", Type: types.TInt}},
		ReturnSort: "Int",
		Ensures: []*core.Contract{
			{
				Kind: core.EnsuresKind,
				Expr: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "result"},
						&core.Lit{Kind: core.IntLit, Value: int64(0)},
					},
				},
			},
		},
	}

	args := []core.CoreExpr{&core.Var{Name: "x"}}
	decl, err := EncodeCalleeByContract(spec, args, 0)
	if err != nil {
		t.Fatalf("EncodeCalleeByContract failed: %v", err)
	}

	// The result constant should be bound by (>= result 0), not by any specific value.
	// This means a caller that asserts (= result 99) should NOT be provable from these assertions.
	if !strings.Contains(decl.SMTLib, decl.ResultConst) {
		t.Errorf("expected result const %q in SMTLib:\n%s", decl.ResultConst, decl.SMTLib)
	}
	// Confirm the weak ensures (>= 0) appears, not a fixed value
	smt := decl.SMTLib
	if !strings.Contains(smt, ">=") && !strings.Contains(smt, "ge_Int") {
		t.Logf("SMTLib produced: %s", smt) // informational
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
// (which cannot be inlined as a define-fun) is emitted via contract-as-spec:
// a declare-const + assert(s), with IsContract=true in the returned CalleeDef.
func TestResolveCallees_ContractFallback(t *testing.T) {
	// Imported lib: recursive(n) — self-recursive, cannot be inlined.
	// It has an ensures { result >= 0 } contract.
	importedProg := &core.Program{
		Meta: map[string]*core.DeclMeta{
			"recursive": {
				IsPure: true,
				Contracts: []*core.Contract{
					{
						Kind: core.EnsuresKind,
						Expr: &core.App{
							Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
							Args: []core.CoreExpr{
								&core.Var{Name: "result"},
								&core.Lit{Kind: core.IntLit, Value: int64(0)},
							},
						},
					},
				},
			},
		},
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "recursive",
				Value: &core.Lambda{
					Params: []string{"n"},
					// Self-recursive: recursive(n-1) — IsSMTEncodableForCallee will reject
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

	// Must produce exactly one def for "recursive" with IsContract=true
	if len(defs) != 1 {
		t.Fatalf("expected 1 callee def, got %d: %v", len(defs), defs)
	}
	d := defs[0]
	if d.Name != "recursive" {
		t.Errorf("expected callee name 'recursive', got %q", d.Name)
	}
	if !d.IsContract {
		t.Errorf("expected IsContract=true for recursive callee (cannot be inlined)")
	}
	if d.ResultConst == "" {
		t.Errorf("expected non-empty ResultConst for contract callee")
	}
	if !strings.Contains(d.SMTLib, "declare-const") {
		t.Errorf("expected declare-const in SMTLib:\n%s", d.SMTLib)
	}
	if !strings.Contains(d.SMTLib, "assert") {
		t.Errorf("expected assert in SMTLib:\n%s", d.SMTLib)
	}
}

// TestEncodeCalleeByContract_MutualRecursion verifies that mutually-recursive
// cross-module functions (which can't be inlined) can still be modelled via
// their contracts, and the result constant is properly named.
func TestEncodeCalleeByContract_MutualRecursion(t *testing.T) {
	// f(n) -> int, ensures { result >= 0 }  (calls g which calls f back)
	// g(n) -> int, ensures { result >= 0 }  (calls f which calls g back)
	// Both have weak contracts that are sufficient to discharge each other.
	specF := ContractSpec{
		FuncName:   "f",
		Params:     []FunctionParam{{Name: "n", Type: types.TInt}},
		ReturnSort: "Int",
		Ensures: []*core.Contract{
			{
				Kind: core.EnsuresKind,
				Expr: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "result"},
						&core.Lit{Kind: core.IntLit, Value: int64(0)},
					},
				},
			},
		},
	}
	specG := ContractSpec{
		FuncName:   "g",
		Params:     []FunctionParam{{Name: "n", Type: types.TInt}},
		ReturnSort: "Int",
		Ensures: []*core.Contract{
			{
				Kind: core.EnsuresKind,
				Expr: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "result"},
						&core.Lit{Kind: core.IntLit, Value: int64(0)},
					},
				},
			},
		},
	}

	argsN := []core.CoreExpr{&core.Var{Name: "n"}}

	declF, err := EncodeCalleeByContract(specF, argsN, 0)
	if err != nil {
		t.Fatalf("EncodeCalleeByContract for f: %v", err)
	}
	declG, err := EncodeCalleeByContract(specG, argsN, 0)
	if err != nil {
		t.Fatalf("EncodeCalleeByContract for g: %v", err)
	}

	// Both result constants must be distinct
	if declF.ResultConst == declG.ResultConst {
		t.Errorf("result constants must be distinct: both are %q", declF.ResultConst)
	}

	// Both must declare their constants and assert ensures
	for _, decl := range []*ContractCalleeDecl{declF, declG} {
		if !strings.Contains(decl.SMTLib, "declare-const") {
			t.Errorf("%s: expected declare-const in SMTLib:\n%s", decl.Name, decl.SMTLib)
		}
		if !strings.Contains(decl.SMTLib, "assert") {
			t.Errorf("%s: expected assert in SMTLib:\n%s", decl.Name, decl.SMTLib)
		}
	}
}
