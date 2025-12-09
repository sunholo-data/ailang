package golang

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

func TestGenerateMatch_ConstructorPattern(t *testing.T) {
	// match x with
	//   | Some(v) -> v
	//   | None -> 0
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "getValue",
				Value: &core.Lambda{
					Params: []string{"x"},
					Body: &core.Match{
						Scrutinee:  &core.Var{Name: "x"},
						Exhaustive: true,
						Arms: []core.MatchArm{
							{
								Pattern: &core.ConstructorPattern{
									Name: "Some",
									Args: []core.CorePattern{&core.VarPattern{Name: "v"}},
								},
								Body: &core.Var{Name: "v"},
							},
							{
								Pattern: &core.ConstructorPattern{
									Name: "None",
									Args: []core.CorePattern{},
								},
								Body: &core.Lit{Kind: core.IntLit, Value: int64(0)},
							},
						},
					},
				},
				Body: &core.Var{Name: "getValue"},
			},
		},
	}

	gen := New("test")
	// Register ADT constructors so the generator can determine the parent type
	gen.RegisterADTConstructor("Option", "Some", 1)
	gen.RegisterADTConstructor("Option", "None", 0)

	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should have switch statement
	if !strings.Contains(codeStr, "switch") {
		t.Errorf("Missing switch statement, got:\n%s", codeStr)
	}

	// Should have case clauses
	if !strings.Contains(codeStr, "case") {
		t.Errorf("Missing case clause, got:\n%s", codeStr)
	}

	// Should have proper Kind-based case (not type-based)
	if !strings.Contains(codeStr, "OptionKind") {
		t.Errorf("Missing OptionKind in case clause, got:\n%s", codeStr)
	}
}

func TestGenerateMatch_WildcardPattern(t *testing.T) {
	// match x with
	//   | _ -> 42
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "always42",
				Value: &core.Lambda{
					Params: []string{"x"},
					Body: &core.Match{
						Scrutinee:  &core.Var{Name: "x"},
						Exhaustive: true,
						Arms: []core.MatchArm{
							{
								Pattern: &core.WildcardPattern{},
								Body:    &core.Lit{Kind: core.IntLit, Value: int64(42)},
							},
						},
					},
				},
				Body: &core.Var{Name: "always42"},
			},
		},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should contain 42 as the result
	if !strings.Contains(codeStr, "42") {
		t.Errorf("Missing result value, got:\n%s", codeStr)
	}
}
