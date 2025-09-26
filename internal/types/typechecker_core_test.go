package types

import (
	"testing"

	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
)

func TestCoreTypeChecker(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectError    bool
		expectWarnings bool // For Num constraints
	}{
		{
			name:           "simple arithmetic with Num warnings",
			input:          "2 + 3",
			expectError:    false,
			expectWarnings: true, // Num constraints
		},
		{
			name:           "pure lambda - no warnings",
			input:          `\x. x`,
			expectError:    false,
			expectWarnings: false,
		},
		{
			name:           "polymorphic identity",
			input:          `let id = \x. x in id`,
			expectError:    false,
			expectWarnings: false,
		},
		{
			name:           "let polymorphism",
			input:          `let id = \x. x in let a = id(5) in let b = id("hello") in a`,
			expectError:    false,
			expectWarnings: false, // Actually succeeds without Num constraints!
		},
		{
			name:           "function composition",
			input:          `let compose = \f g x. f(g(x)) in compose`,
			expectError:    false,
			expectWarnings: false,
		},
		{
			name:           "if expression",
			input:          `if true then 1 else 2`,
			expectError:    false,
			expectWarnings: false,
		},
		{
			name:           "list literal",
			input:          `[1, 2, 3]`,
			expectError:    false,
			expectWarnings: false,
		},
		{
			name:           "curried function",
			input:          `\x y. x`,
			expectError:    false,
			expectWarnings: false,
		},
		{
			name:           "Church booleans",
			input:          `let tru = \t f. t in let fls = \t f. f in tru`,
			expectError:    false,
			expectWarnings: false,
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

			// Elaborate to Core
			elab := elaborate.NewElaborator()
			coreProg, err := elab.Elaborate(prog)
			if err != nil {
				t.Fatalf("elaboration error: %v", err)
			}

			// Type check
			tc := NewCoreTypeChecker()
			typedProg, err := tc.CheckCoreProgram(coreProg)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else if tt.expectWarnings {
				// Should have unsolved Num constraints
				if err == nil {
					t.Errorf("expected Num constraint warnings but got success")
				} else if err.Error() == "" || err.Error() == "unsolved constraints" {
					// This is fine - we expect unsolved constraints
				}
			} else {
				// Should succeed completely
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if typedProg == nil {
					t.Errorf("expected typed program but got nil")
				}
			}
		})
	}
}

func TestCoreTypeInference(t *testing.T) {
	// Test that the type inference works correctly for pure expressions
	tests := []struct {
		name  string
		input string
		// We can't easily check the exact type without exposing internals,
		// so we just check it succeeds
	}{
		{
			name:  "identity function",
			input: `\x. x`,
		},
		{
			name:  "const function",
			input: `\x y. x`,
		},
		{
			name:  "flip function",
			input: `\f x y. f(y)(x)`,
		},
		{
			name:  "self application",
			input: `let apply = \f x. f(x) in apply`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Full pipeline
			l := lexer.New(tt.input, "test.ail")
			p := parser.New(l)
			prog := p.Parse()

			if len(p.Errors()) > 0 {
				t.Fatalf("parse errors: %v", p.Errors())
			}

			elab := elaborate.NewElaborator()
			coreProg, err := elab.Elaborate(prog)
			if err != nil {
				t.Fatalf("elaboration error: %v", err)
			}

			tc := NewCoreTypeChecker()
			typedProg, err := tc.CheckCoreProgram(coreProg)
			
			// These pure lambda expressions should type check successfully
			if err != nil {
				t.Errorf("type checking failed: %v", err)
			}
			if typedProg == nil {
				t.Errorf("expected typed program")
			}
		})
	}
}

func TestLetPolymorphism(t *testing.T) {
	// Test that let-polymorphism works correctly
	input := `
		let id = \x. x in
		let five = id(5) in
		let hello = id("hello") in
		id
	`

	l := lexer.New(input, "test.ail")
	p := parser.New(l)
	prog := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}

	elab := elaborate.NewElaborator()
	coreProg, err := elab.Elaborate(prog)
	if err != nil {
		t.Fatalf("elaboration error: %v", err)
	}

	tc := NewCoreTypeChecker()
	_, err = tc.CheckCoreProgram(coreProg)
	
	// This will have Num constraints from id(5), but should otherwise work
	if err != nil && err.Error() != "unsolved constraints" {
		t.Errorf("unexpected error: %v", err)
	}
}