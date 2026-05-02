package smt

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// TestImportedProgramsFieldRoundTrip verifies that ImportedPrograms survives
// the EncodeFunctionOpts plumbing: a 2-module fixture can pass the imported
// program in via opts and the field is accessible to the encoder.
func TestImportedProgramsFieldRoundTrip(t *testing.T) {
	// Build a minimal "lib" core.Program: double(n) = n + n
	libProg := &core.Program{
		Meta: map[string]*core.DeclMeta{
			"double": {
				IsExport: true,
				IsPure:   true,
				Contracts: []*core.Contract{
					{
						Kind:    core.EnsuresKind,
						Expr:    &core.Var{Name: "result"},
						Message: "double result",
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

	// Build a trivial caller program: identity(x) = x, ensures { result == x }
	callerProg := &core.Program{
		Meta: map[string]*core.DeclMeta{
			"identity": {
				IsExport: true,
				IsPure:   true,
				Contracts: []*core.Contract{
					{
						Kind: core.EnsuresKind,
						Expr: &core.App{
							Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "eq_Int"}},
							Args: []core.CoreExpr{
								&core.Var{Name: "result"},
								&core.Var{Name: "x"},
							},
						},
						Message: "identity postcondition",
					},
				},
			},
		},
		Decls: []core.CoreExpr{
			&core.Let{
				Name:  "identity",
				Value: &core.Lambda{Params: []string{"x"}, Body: &core.Var{Name: "x"}},
				Body:  &core.Var{Name: "identity"},
			},
		},
	}

	importedPrograms := map[string]*core.Program{
		"myapp/math_lib": libProg,
	}

	opts := EncodeFunctionOpts{
		Program:          callerProg,
		ImportedPrograms: importedPrograms,
	}

	// Verify ImportedPrograms is accessible in opts
	if opts.ImportedPrograms == nil {
		t.Fatal("ImportedPrograms should not be nil")
	}
	if prog, ok := opts.ImportedPrograms["myapp/math_lib"]; !ok {
		t.Fatal("expected myapp/math_lib in ImportedPrograms")
	} else if prog == nil {
		t.Fatal("imported program should not be nil")
	} else if len(prog.Decls) == 0 {
		t.Fatal("imported program should have decls")
	}

	// EncodeFunction with ImportedPrograms should not crash
	params := []FunctionParam{{Name: "x", Type: types.TInt}}
	body := &core.Var{Name: "x"}
	meta := callerProg.Meta["identity"]

	result, err := EncodeFunction("identity", params, body, "Int", meta, nil, opts)
	if err != nil {
		t.Fatalf("EncodeFunction with ImportedPrograms should not error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// TestImportedProgramsNilSafe verifies nil ImportedPrograms doesn't crash (backwards compat).
func TestImportedProgramsNilSafe(t *testing.T) {
	prog := &core.Program{
		Meta: map[string]*core.DeclMeta{
			"add1": {
				IsPure: true,
				Contracts: []*core.Contract{
					{
						Kind: core.EnsuresKind,
						Expr: &core.App{
							Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "eq_Int"}},
							Args: []core.CoreExpr{
								&core.Var{Name: "result"},
								&core.App{
									Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
									Args: []core.CoreExpr{&core.Var{Name: "n"}, &core.Lit{Kind: core.IntLit, Value: int64(1)}},
								},
							},
						},
						Message: "add1 postcondition",
					},
				},
			},
		},
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "add1",
				Value: &core.Lambda{
					Params: []string{"n"},
					Body: &core.App{
						Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
						Args: []core.CoreExpr{&core.Var{Name: "n"}, &core.Lit{Kind: core.IntLit, Value: int64(1)}},
					},
				},
				Body: &core.Var{Name: "add1"},
			},
		},
	}

	// No ImportedPrograms — nil map must be safe
	opts := EncodeFunctionOpts{Program: prog}
	params := []FunctionParam{{Name: "n", Type: types.TInt}}
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
		Args: []core.CoreExpr{&core.Var{Name: "n"}, &core.Lit{Kind: core.IntLit, Value: int64(1)}},
	}
	meta := prog.Meta["add1"]

	result, err := EncodeFunction("add1", params, body, "Int", meta, nil, opts)
	if err != nil {
		t.Fatalf("nil ImportedPrograms should not error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}
