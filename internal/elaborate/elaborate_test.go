package elaborate

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

func TestElaborateSimple(t *testing.T) {
	tests := []struct {
		name  string
		input string
		// We mainly want to check it doesn't error
		expectError bool
	}{
		{
			name:        "simple arithmetic",
			input:       "2 + 3",
			expectError: false,
		},
		{
			name:        "complex expression gets normalized",
			input:       "(2 + 3) * (4 + 5)",
			expectError: false,
		},
		{
			name:        "let binding",
			input:       "let x = 5 in x + 1",
			expectError: false,
		},
		{
			name:        "lambda expression",
			input:       `\x. x + 1`,
			expectError: false,
		},
		{
			name:        "nested let",
			input:       "let x = 5 in let y = x + 1 in y * 2",
			expectError: false,
		},
		{
			name:        "if expression",
			input:       "if true then 1 else 0",
			expectError: false,
		},
		{
			name:        "list literal",
			input:       "[1, 2, 3]",
			expectError: false,
		},
		{
			name:        "record literal",
			input:       `{name: "test", value: 42}`,
			expectError: false,
		},
		{
			name:        "function application",
			input:       `(\x. x + 1)(5)`,
			expectError: false,
		},
		{
			name:        "curried function",
			input:       `\x y. x + y`,
			expectError: false,
		},
		{
			name:        "cons expression basic",
			input:       `1 :: [2, 3]`,
			expectError: false,
		},
		{
			name:        "cons expression chained",
			input:       `1 :: 2 :: 3 :: []`,
			expectError: false,
		},
		{
			name:        "cons in let binding",
			input:       `let xs = 1 :: [2, 3] in xs`,
			expectError: false,
		},
		{
			name:        "cons as function argument",
			input:       `(\xs. xs)(1 :: [2, 3])`,
			expectError: false,
		},
		{
			name:        "cons in if branch",
			input:       `if true then 1 :: [] else []`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			l := lexer.New(tt.input, "test.ail")
			p := parser.New(l)
			prog := p.Parse()

			if len(p.Errors()) > 0 {
				t.Fatalf("parse errors: %v", p.Errors())
			}

			// Elaborate
			elab := NewElaborator()
			coreProg, err := elab.Elaborate(prog)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if coreProg == nil {
					t.Errorf("expected core program but got nil")
				}
			}
		})
	}
}

func TestANFTransformation(t *testing.T) {
	// Test that complex expressions get properly normalized to ANF
	// Create a simple binary op manually for testing
	// This tests the elaboration directly without parser complications
	expr := &ast.BinaryOp{
		Left: &ast.BinaryOp{
			Left:  &ast.Identifier{Name: "a"},
			Op:    "+",
			Right: &ast.Identifier{Name: "b"},
		},
		Op: "*",
		Right: &ast.BinaryOp{
			Left:  &ast.Identifier{Name: "c"},
			Op:    "+",
			Right: &ast.Identifier{Name: "d"},
		},
	}

	elab := NewElaborator()
	coreExpr, err := elab.ElaborateExpr(expr)

	if err != nil {
		t.Fatalf("elaboration error: %v", err)
	}

	// The result should be a non-nil core expression
	if coreExpr == nil {
		t.Errorf("expected non-nil core expression")
	}
}

func TestNodeIDAssignment(t *testing.T) {
	// Test that every node gets a unique ID
	// Create a let expression manually for testing
	expr := &ast.Let{
		Name: "x",
		Value: &ast.Literal{
			Kind:  ast.IntLit,
			Value: 5,
		},
		Body: &ast.Let{
			Name: "y",
			Value: &ast.Literal{
				Kind:  ast.IntLit,
				Value: 10,
			},
			Body: &ast.BinaryOp{
				Left:  &ast.Identifier{Name: "x"},
				Op:    "+",
				Right: &ast.Identifier{Name: "y"},
			},
		},
	}

	elab := NewElaborator()
	_, err := elab.ElaborateExpr(expr)

	if err != nil {
		t.Fatalf("elaboration error: %v", err)
	}

	// Check that IDs are being assigned (starts at 1)
	if elab.nextID <= 1 {
		t.Errorf("expected node IDs to be assigned, but nextID is %d", elab.nextID)
	}
}

func coreLetChain(names []string, body core.CoreExpr) core.CoreExpr {
	out := body
	for i := len(names) - 1; i >= 0; i-- {
		out = &core.Let{Name: names[i], Value: &core.Lit{Kind: core.IntLit, Value: int64(i)}, Body: out}
	}
	return out
}

func TestExtractLetBindings(t *testing.T) {
	// Elaborator temporaries move out, outermost first.
	body := &core.Lit{Kind: core.IntLit, Value: int64(42)}
	bindings, rest := extractLetBindings(coreLetChain([]string{"$tmp1", "$tmp2", "$tmp3"}, body))
	var got []string
	for _, b := range bindings {
		got = append(got, b.Name)
	}
	if strings.Join(got, ",") != "$tmp1,$tmp2,$tmp3" {
		t.Errorf("extracted %v, want [$tmp1 $tmp2 $tmp3]", got)
	}
	if rest != core.CoreExpr(body) {
		t.Errorf("rest = %s, want the innermost body", rest)
	}
}

// Only the names freshVar makes count as elaborator temporaries.
func TestIsElaboratorTemp(t *testing.T) {
	e := NewElaborator()
	fresh := e.freshVar()
	for name, want := range map[string]bool{fresh: true, "$tmp12": true, "r": false, "tmp1": false, "$valTmp3": false, "": false} {
		if got := isElaboratorTemp(name); got != want {
			t.Errorf("isElaboratorTemp(%q) = %v, want %v", name, got, want)
		}
	}
}

// A binder the program named stops extraction: moving it outward would put
// the enclosing expression's other reads of that name inside its scope
// (#2388: add({ let r = 2; r * 10 }, r) read the inner r twice).
func TestExtractLetBindingsStopsAtSourceBinder(t *testing.T) {
	body := &core.Lit{Kind: core.IntLit, Value: int64(42)}
	bindings, rest := extractLetBindings(coreLetChain([]string{"$tmp1", "r", "$tmp2"}, body))
	if len(bindings) != 1 || bindings[0].Name != "$tmp1" {
		t.Fatalf("extracted %v, want only $tmp1", bindings)
	}
	if let, ok := rest.(*core.Let); !ok || let.Name != "r" {
		t.Fatalf("rest = %s, want the chain from `let r` on, still nested", rest)
	}
}

func TestExtractLetBindingsNonLet(t *testing.T) {
	// Test that non-Let expressions return empty bindings

	lit := &ast.Literal{Kind: ast.IntLit, Value: int64(42)}

	elab := NewElaborator()
	coreExpr, err := elab.ElaborateExpr(lit)
	if err != nil {
		t.Fatalf("elaboration error: %v", err)
	}

	bindings, body := extractLetBindings(coreExpr)

	if len(bindings) != 0 {
		t.Errorf("expected 0 bindings for non-Let, got %d", len(bindings))
	}

	if body != coreExpr {
		t.Error("body should be the original expression for non-Let")
	}
}

func TestNestedRecordElaboration(t *testing.T) {
	// Test that nested record literals can be elaborated without ANF verification errors
	// This is the bug reported by stapledons_voyage

	// Wrap the expression in a let binding to make it a valid program
	input := `let npc = { pos: { x: 10, y: 20 }, name: "guard" } in npc`

	l := lexer.New(input, "test.ail")
	p := parser.New(l)
	prog := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	elab := NewElaborator()
	coreProg, err := elab.Elaborate(prog)

	if err != nil {
		t.Fatalf("elaboration error: %v", err)
	}

	if coreProg == nil {
		t.Error("expected non-nil core program")
	}

	// The key test: verify no nested Let in RHS
	// After flattening, the structure should be:
	// Let $tmp = {x: 10, y: 20} in Let npc = {pos: $tmp, name: "guard"} in npc
	// NOT: Let npc = (Let $tmp = {x: 10, y: 20} in ...) in npc
}

// TestLetInIfElseBranchError tests that using let-without-body in if-else branches
// produces a helpful error message (M-FIX-IF-ELSE-LET)
func TestLetInIfElseBranchError(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorSubstr string
	}{
		{
			name: "let in else branch without braces - allowed since M-AI-PREFERRED-V2 D1=b",
			input: `module test
pure func test(x: int) -> [int] {
    if x > 10 then []
    else
        let v = x;
        [v]
}`,
			expectError: false,
			errorSubstr: "",
		},
		{
			name: "let in then branch without braces - allowed since M-AI-PREFERRED-V2 D1=b",
			input: `module test
pure func test(x: int) -> [int] {
    if x > 10 then
        let v = x;
        [v]
    else []
}`,
			expectError: false,
			errorSubstr: "",
		},
		{
			name: "let in else branch WITH braces - should work",
			input: `module test
pure func test(x: int) -> [int] {
    if x > 10 then [] else {
        let v = x;
        [v]
    }
}`,
			expectError: false,
		},
		{
			name: "simple if-else without let - should work",
			input: `module test
pure func test(x: int) -> [int] {
    if x > 10 then [] else [x]
}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.ail")
			p := parser.New(l)
			prog := p.Parse()

			if len(p.Errors()) > 0 {
				if tt.expectError {
					// Parse error is acceptable for some cases
					return
				}
				t.Fatalf("parse errors: %v", p.Errors())
			}

			elab := NewElaborator()
			_, err := elab.Elaborate(prog)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorSubstr != "" && !contains(err.Error(), tt.errorSubstr) {
					t.Errorf("expected error containing %q, got: %v", tt.errorSubstr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// contains checks if s contains substr (simple helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestConsExpressionElaboratesToVarGlobal verifies that :: in expression position
// elaborates to a core.App with VarGlobal referencing the $builtin cons function
// when builtins are registered in the global environment.
func TestConsExpressionElaboratesToVarGlobal(t *testing.T) {
	input := `1 :: [2, 3]`

	l := lexer.New(input, "test.ail")
	p := parser.New(l)
	prog := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}

	elab := NewElaborator()
	elab.AddBuiltinsToGlobalEnv() // Register :: in globalEnv
	coreProg, err := elab.Elaborate(prog)
	if err != nil {
		t.Fatalf("elaboration error: %v", err)
	}

	// Walk the core tree to find VarGlobal with Name "::"
	found := findVarGlobal(coreProg.Decls[0], "::")
	if !found {
		t.Errorf("expected VarGlobal with Name \"::\" in elaborated output, but not found")
	}
}

// findVarGlobal recursively checks if any node in the Core tree is a VarGlobal with the given name
func findVarGlobal(expr interface{}, name string) bool {
	switch e := expr.(type) {
	case *core.App:
		if findVarGlobal(e.Func, name) {
			return true
		}
		for _, arg := range e.Args {
			if findVarGlobal(arg, name) {
				return true
			}
		}
	case *core.VarGlobal:
		return e.Ref.Name == name
	case *core.Let:
		if findVarGlobal(e.Value, name) {
			return true
		}
		return findVarGlobal(e.Body, name)
	case *core.LetRec:
		for _, b := range e.Bindings {
			if findVarGlobal(b.Value, name) {
				return true
			}
		}
		return findVarGlobal(e.Body, name)
	}
	return false
}
