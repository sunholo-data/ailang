package parser

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/lexer"
)

// TestPatternMatchingInFunctions tests that pattern matching works inside function bodies
// This was previously broken - patterns would parse at top-level but fail inside functions
func TestPatternMatchingInFunctions(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "empty list pattern",
			input: `module test
				export func isEmpty(xs: [int]) -> bool {
					match xs { [] => true, _ => false }
				}`,
		},
		{
			name: "list with spread",
			input: `module test
				export func length(xs: [int]) -> int {
					match xs {
						[] => 0,
						[_, ...rest] => 1 + length(rest)
					}
				}`,
		},
		{
			name: "tuple pattern",
			input: `module test
				export func swap(p: (int, string)) -> (string, int) {
					match p { (x, y) => (y, x) }
				}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(lexer.New(tt.input, "<test>"))
			prog := p.Parse()

			if len(p.Errors()) > 0 {
				t.Errorf("Expected to parse but got errors:")
				for _, err := range p.Errors() {
					t.Errorf("  %v", err)
				}
			}

			if prog == nil || prog.File == nil {
				t.Errorf("Expected valid program")
			}
		})
	}
}

// TestListPatternSpreadError tests error handling for spread without identifier
func TestListPatternSpreadError(t *testing.T) {
	input := `module test
		export func f(xs: [int]) -> int {
			match xs { [x, ...] => x }
		}`

	p := New(lexer.New(input, "<test>"))
	_ = p.Parse()
	errs := p.Errors()

	if len(errs) == 0 {
		t.Errorf("Expected parse errors but got none")
		return
	}

	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "PAT_SPREAD_NEEDS_IDENT") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected error containing PAT_SPREAD_NEEDS_IDENT but got: %v", errs)
	}
}
