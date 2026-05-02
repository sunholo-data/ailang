package smt

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// TestXmodInline_DoubleQuadruple verifies that an imported pure function
// (double) is inlined into the caller (quadruple) so the verifier can
// reason about the composition.
//
// math_lib: double(n) = n + n,  ensures { result == n + n }
// main:     quadruple(n) = double(double(n)),  ensures { result == n * 4 }
func TestXmodInline_DoubleQuadruple(t *testing.T) {
	// Build math_lib core.Program
	mathLibProg := &core.Program{
		Meta: map[string]*core.DeclMeta{
			"double": {
				IsExport: true,
				IsPure:   true,
				Contracts: []*core.Contract{
					{
						Kind: core.EnsuresKind,
						Expr: &core.App{
							Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "eq_Int"}},
							Args: []core.CoreExpr{
								&core.Var{Name: "result"},
								&core.App{
									Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
									Args: []core.CoreExpr{&core.Var{Name: "n"}, &core.Var{Name: "n"}},
								},
							},
						},
						Message: "double postcondition",
					},
				},
			},
		},
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "double",
				Value: &core.Lambda{
					Params: []string{"n"},
					Body: &core.App{
						Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
						Args: []core.CoreExpr{&core.Var{Name: "n"}, &core.Var{Name: "n"}},
					},
				},
				Body: &core.Var{Name: "double"},
			},
		},
	}

	// Build main core.Program: quadruple calls double(double(n))
	mainProg := &core.Program{
		Meta: map[string]*core.DeclMeta{
			"quadruple": {
				IsExport: true,
				IsPure:   true,
				Contracts: []*core.Contract{
					{
						Kind: core.EnsuresKind,
						Expr: &core.App{
							Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "eq_Int"}},
							Args: []core.CoreExpr{
								&core.Var{Name: "result"},
								&core.App{
									Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "mul_Int"}},
									Args: []core.CoreExpr{
										&core.Var{Name: "n"},
										&core.Lit{Kind: core.IntLit, Value: int64(4)},
									},
								},
							},
						},
						Message: "quadruple postcondition",
					},
				},
			},
		},
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "quadruple",
				Value: &core.Lambda{
					Params: []string{"n"},
					// double(double(n))
					Body: &core.App{
						Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "myapp/math_lib", Name: "double"}},
						Args: []core.CoreExpr{
							&core.App{
								Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "myapp/math_lib", Name: "double"}},
								Args: []core.CoreExpr{&core.Var{Name: "n"}},
							},
						},
					},
				},
				Body: &core.Var{Name: "quadruple"},
			},
		},
	}

	importedPrograms := map[string]*core.Program{
		"myapp/math_lib": mathLibProg,
	}

	// Verify that double is collected as a cross-module callee
	quadBody, _ := findFuncBodyInAnyProg("quadruple", mainProg, importedPrograms)
	if quadBody == nil {
		t.Fatal("expected to find quadruple body")
	}
	_, innerBody := unwrapLambda(quadBody)
	callees := collectCalleeCalls(innerBody, "quadruple", mainProg, importedPrograms, 0)
	found := false
	for _, c := range callees {
		if c == "double" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'double' in collected callees, got: %v", callees)
	}

	// Verify ResolveCallees produces a define-fun for double
	surfaceParams := map[string][]FunctionParam{
		"double":    {{Name: "n", Type: types.TInt}},
		"quadruple": {{Name: "n", Type: types.TInt}},
	}
	surfaceReturnSorts := map[string]string{
		"double":    "Int",
		"quadruple": "Int",
	}

	defs, err := ResolveCallees("quadruple", innerBody, mainProg, surfaceParams, surfaceReturnSorts, nil, importedPrograms)
	if err != nil {
		t.Fatalf("ResolveCallees failed: %v", err)
	}
	if len(defs) == 0 {
		t.Fatal("expected at least one callee def (double), got none")
	}
	found = false
	for _, d := range defs {
		if d.Name == "double" {
			found = true
			if !strings.Contains(d.SMTLib, "define-fun") {
				t.Errorf("expected define-fun in SMTLib, got: %s", d.SMTLib)
			}
		}
	}
	if !found {
		t.Errorf("expected callee def for 'double', got: %v", defs)
	}
}

// TestXmodInline_DepthCap verifies that the crossModuleInlineDepth cap
// prevents infinite recursion when imported functions call other imported functions.
func TestXmodInline_DepthCap(t *testing.T) {
	// lib_a: funcA calls funcB from lib_b
	// lib_b: funcB calls funcC from lib_c
	// lib_c: funcC calls funcA from lib_a (mutual recursion across 3 modules)
	// Depth cap should prevent infinite loop.

	libA := &core.Program{
		Meta: map[string]*core.DeclMeta{"funcA": {IsPure: true}},
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "funcA",
				Value: &core.Lambda{
					Params: []string{"n"},
					Body: &core.App{
						Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "lib_b", Name: "funcB"}},
						Args: []core.CoreExpr{&core.Var{Name: "n"}},
					},
				},
				Body: &core.Var{Name: "funcA"},
			},
		},
	}
	libB := &core.Program{
		Meta: map[string]*core.DeclMeta{"funcB": {IsPure: true}},
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "funcB",
				Value: &core.Lambda{
					Params: []string{"n"},
					Body: &core.App{
						Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "lib_c", Name: "funcC"}},
						Args: []core.CoreExpr{&core.Var{Name: "n"}},
					},
				},
				Body: &core.Var{Name: "funcB"},
			},
		},
	}
	libC := &core.Program{
		Meta: map[string]*core.DeclMeta{"funcC": {IsPure: true}},
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "funcC",
				Value: &core.Lambda{
					Params: []string{"n"},
					Body: &core.App{
						Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "lib_a", Name: "funcA"}},
						Args: []core.CoreExpr{&core.Var{Name: "n"}},
					},
				},
				Body: &core.Var{Name: "funcC"},
			},
		},
	}

	// main calls funcA
	mainProg := &core.Program{
		Meta: map[string]*core.DeclMeta{"caller": {IsPure: true}},
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "caller",
				Value: &core.Lambda{
					Params: []string{"n"},
					Body: &core.App{
						Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "lib_a", Name: "funcA"}},
						Args: []core.CoreExpr{&core.Var{Name: "n"}},
					},
				},
				Body: &core.Var{Name: "caller"},
			},
		},
	}

	imported := map[string]*core.Program{
		"lib_a": libA,
		"lib_b": libB,
		"lib_c": libC,
	}

	// This must terminate (not loop forever) — depth cap kicks in
	callerBody, _ := findFuncBodyInAnyProg("caller", mainProg, imported)
	if callerBody == nil {
		t.Fatal("expected caller body")
	}
	_, inner := unwrapLambda(callerBody)

	done := make(chan struct{})
	go func() {
		collectCalleeCalls(inner, "caller", mainProg, imported, 0)
		close(done)
	}()

	<-done // must terminate — depth cap prevents infinite loop
}
