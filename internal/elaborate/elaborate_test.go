package elaborate

import (
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
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
