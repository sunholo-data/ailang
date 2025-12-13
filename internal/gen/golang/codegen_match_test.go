package golang

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
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

// TestGenerateMatch_NestedBoolFromADT tests M-DX27: nested bool match on typed ADT field
// should NOT add .(bool) type assertion since the field is already bool, not interface{}.
func TestGenerateMatch_NestedBoolFromADT(t *testing.T) {
	// Simulates:
	// match content {
	//   ContentStarfield(d, s) => match s { true => 1.0, false => 0.0 }
	// }
	// Where s is a bool extracted from ADT field
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "render",
				Value: &core.Lambda{
					Params: []string{"content"},
					Body: &core.Match{
						Scrutinee:  &core.Var{Name: "content"},
						Exhaustive: true,
						Arms: []core.MatchArm{
							{
								Pattern: &core.ConstructorPattern{
									Name: "ContentStarfield",
									Args: []core.CorePattern{
										&core.VarPattern{Name: "d"},
										&core.VarPattern{Name: "s"}, // s is bool from ADT field
									},
								},
								Body: &core.Match{
									Scrutinee:  &core.Var{Name: "s"},
									Exhaustive: true,
									Arms: []core.MatchArm{
										{
											Pattern: &core.LitPattern{Value: true},
											Body:    &core.Lit{Kind: core.FloatLit, Value: 1.0},
										},
										{
											Pattern: &core.LitPattern{Value: false},
											Body:    &core.Lit{Kind: core.FloatLit, Value: 0.0},
										},
									},
								},
							},
						},
					},
				},
				Body: &core.Var{Name: "render"},
			},
		},
	}

	gen := New("test")
	// Register ADT with bool field (fieldTypes are the Go types for the fields)
	gen.RegisterADTConstructorWithTypes("Content", "ContentStarfield", []string{"float64", "bool"})

	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// M-DX27: The nested bool match should NOT have .(bool) type assertion
	// because s is already a bool from the ADT field, not interface{}
	if strings.Contains(codeStr, "s.(bool)") {
		t.Errorf("M-DX27 FAIL: Generated code should NOT contain s.(bool) for typed ADT field.\n"+
			"The variable 's' is already bool from ADT extraction, not interface{}.\n"+
			"Got:\n%s", codeStr)
	}

	// Should still contain the bool match logic (just without type assertion)
	if !strings.Contains(codeStr, "if") || !strings.Contains(codeStr, "return") {
		t.Errorf("Missing if/return in bool match, got:\n%s", codeStr)
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

// TestGenerateMatch_OptionNestedADT tests M-DX29: extracting ADT from Option[ADT] should add type assertion
// When matching Option[InteractableID], the extracted value should be typed as *InteractableID
func TestGenerateMatch_OptionNestedADT(t *testing.T) {
	// Simulates:
	// match optInteractable {     -- Option[InteractableID]
	//   Some(interactable) =>
	//     match interactable {    -- should be *InteractableID, not interface{}
	//       InteractConsole(station) => station
	//       _ => 0
	//     }
	//   None => 0
	// }

	// Create NodeIDs for type info tracking
	outerMatchNodeID := uint64(100)
	outerScrutineeNodeID := uint64(101)
	innerMatchNodeID := uint64(102)
	innerScrutineeNodeID := uint64(103)

	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "getStation",
				Value: &core.Lambda{
					Params: []string{"optInteractable"},
					Body: &core.Match{
						CoreNode:   core.CoreNode{NodeID: outerMatchNodeID},
						Scrutinee:  &core.Var{CoreNode: core.CoreNode{NodeID: outerScrutineeNodeID}, Name: "optInteractable"},
						Exhaustive: true,
						Arms: []core.MatchArm{
							{
								Pattern: &core.ConstructorPattern{
									Name: "Some",
									Args: []core.CorePattern{
										&core.VarPattern{Name: "interactable"},
									},
								},
								Body: &core.Match{
									CoreNode:   core.CoreNode{NodeID: innerMatchNodeID},
									Scrutinee:  &core.Var{CoreNode: core.CoreNode{NodeID: innerScrutineeNodeID}, Name: "interactable"},
									Exhaustive: true,
									Arms: []core.MatchArm{
										{
											Pattern: &core.ConstructorPattern{
												Name: "InteractConsole",
												Args: []core.CorePattern{
													&core.VarPattern{Name: "station"},
												},
											},
											Body: &core.Var{Name: "station"},
										},
										{
											Pattern: &core.WildcardPattern{},
											Body:    &core.Lit{Kind: core.IntLit, Value: int64(0)},
										},
									},
								},
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
				Body: &core.Var{Name: "getStation"},
			},
		},
	}

	gen := New("test")

	// Register Option ADT (generic container - field type is interface{})
	gen.RegisterADTConstructor("Option", "Some", 1)
	gen.RegisterADTConstructor("Option", "None", 0)

	// Register InteractableID ADT
	gen.RegisterADTConstructorWithTypes("InteractableID", "InteractConsole", []string{"int64"})
	gen.RegisterADTConstructorWithTypes("InteractableID", "InteractCrew", []string{"int64"})

	// M-DX29: Set up CoreTypeInfo with TApp type for the outer scrutinee
	// Option[InteractableID] = TApp(Option, [InteractableID])
	coreTypeInfo := make(types.CoreTypeInfo)
	optionType := &types.TApp{
		Constructor: &types.TCon{Name: "Option"},
		Args:        []types.Type{&types.TCon{Name: "InteractableID"}},
	}
	coreTypeInfo[outerScrutineeNodeID] = optionType
	coreTypeInfo[outerMatchNodeID] = &types.TCon{Name: "int"}

	// The inner scrutinee (interactable) should be InteractableID
	coreTypeInfo[innerScrutineeNodeID] = &types.TCon{Name: "InteractableID"}
	coreTypeInfo[innerMatchNodeID] = &types.TCon{Name: "int"}

	gen.SetCoreTypeInfo(coreTypeInfo)

	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// M-DX29: The extraction from Option should add type assertion for the ADT
	// Should contain: interactable := _adt.Some.Value0.(*InteractableID)
	if !strings.Contains(codeStr, ".(*InteractableID)") {
		t.Errorf("M-DX29 FAIL: Generated code should contain .(*InteractableID) type assertion.\n"+
			"When extracting from Option[InteractableID], the value should be type-asserted.\n"+
			"Got:\n%s", codeStr)
	}

	// Should NOT have interface{} access on the inner match
	// The inner match should work because interactable is now typed
	if strings.Contains(codeStr, "interactable.(") && !strings.Contains(codeStr, "interactable.(*InteractableID)") {
		t.Errorf("M-DX29 FAIL: interactable should already be *InteractableID, not need additional assertion.\n"+
			"Got:\n%s", codeStr)
	}
}
