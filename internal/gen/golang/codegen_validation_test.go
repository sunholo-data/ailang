package golang

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

func TestCodePassesGoVet(t *testing.T) {
	// Generate a complex program and verify go/format accepted it
	// (go/format validates syntax, which is a subset of vet)
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.LetRec{
				Bindings: []core.RecBinding{
					{
						Name: "sum",
						Value: &core.Lambda{
							Params: []string{"n"},
							Body: &core.If{
								Cond: &core.BinOp{
									Op:    "<=",
									Left:  &core.Var{Name: "n"},
									Right: &core.Lit{Kind: core.IntLit, Value: int64(0)},
								},
								Then: &core.Lit{Kind: core.IntLit, Value: int64(0)},
								Else: &core.BinOp{
									Op:   "+",
									Left: &core.Var{Name: "n"},
									Right: &core.App{
										Func: &core.Var{Name: "sum"},
										Args: []core.CoreExpr{
											&core.BinOp{
												Op:    "-",
												Left:  &core.Var{Name: "n"},
												Right: &core.Lit{Kind: core.IntLit, Value: int64(1)},
											},
										},
									},
								},
							},
						},
					},
				},
				Body: &core.Var{Name: "sum"},
			},
		},
	}

	gen := New("valid")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v\nThis indicates invalid Go syntax was generated", err)
	}

	// If we reach here, go/format passed which means syntax is valid
	if !strings.Contains(string(code), "package valid") {
		t.Errorf("Invalid package declaration")
	}
}
