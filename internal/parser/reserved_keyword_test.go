package parser

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
)

// TestReservedKeywordErrorMessage tests that reserved keywords produce helpful error messages
// This addresses M-DX24 feedback about cryptic "expected IDENT, got exists" errors
func TestReservedKeywordErrorMessage(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		expectedErr string
		shouldFail  bool
	}{
		{
			name: "exists as variable name",
			code: `module test
func check() -> bool {
  let exists = true;
  exists
}`,
			expectedErr: "PAR_RESERVED_KEYWORD",
			shouldFail:  true,
		},
		{
			name: "forall as variable name",
			code: `module test
func check() -> bool {
  let forall = true;
  forall
}`,
			expectedErr: "PAR_RESERVED_KEYWORD",
			shouldFail:  true,
		},
		{
			name: "if as variable name",
			code: `module test
func check() -> bool {
  let if = true;
  if
}`,
			expectedErr: "PAR_RESERVED_KEYWORD",
			shouldFail:  true,
		},
		{
			name: "match as variable name",
			code: `module test
func check() -> bool {
  let match = true;
  match
}`,
			expectedErr: "PAR_RESERVED_KEYWORD",
			shouldFail:  true,
		},
		{
			name: "Valid alternative to exists",
			code: `module test
func check() -> bool {
  let found = true;
  found
}`,
			expectedErr: "",
			shouldFail:  false,
		},
		{
			name: "Valid alternative to forall",
			code: `module test
func checkAll() -> bool {
  let allChecked = true;
  allChecked
}`,
			expectedErr: "",
			shouldFail:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.code, "test.ail")
			p := New(l)
			program := p.Parse()

			hasErrors := len(p.Errors()) > 0
			if tt.shouldFail != hasErrors {
				t.Errorf("shouldFail=%v, but hasErrors=%v", tt.shouldFail, hasErrors)
			}

			if tt.shouldFail && tt.expectedErr != "" {
				found := false
				for _, err := range p.Errors() {
					errStr := err.Error()
					if strings.Contains(errStr, tt.expectedErr) {
						found = true
						// Verify helpful message is present
						if tt.expectedErr == "PAR_RESERVED_KEYWORD" {
							if !strings.Contains(errStr, "reserved keyword") {
								t.Errorf("Error message missing 'reserved keyword' hint: %s", errStr)
							}
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected error code %s not found in:\n", tt.expectedErr)
					for _, err := range p.Errors() {
						t.Logf("  %s", err)
					}
				}
			}

			// Valid code should parse without errors
			if !tt.shouldFail && len(p.Errors()) > 0 {
				t.Logf("Unexpected parse errors:")
				for _, err := range p.Errors() {
					t.Logf("  %s", err)
				}
				t.Fatal("Valid code failed to parse")
			}

			// Both cases should produce a program (or nil on error)
			if program == nil && !tt.shouldFail {
				t.Error("Expected program to be non-nil for valid code")
			}
		})
	}
}

// TestReservedKeywordSuggestions verifies that error suggestions are helpful
// This tests the M-DX24 improvement: "Did you mean one of these?"
func TestReservedKeywordSuggestions(t *testing.T) {
	code := `module test
func check() -> bool {
  let exists = fileExists("test");
  exists
}`

	l := lexer.New(code, "test.ail")
	p := New(l)
	_ = p.Parse()

	// Should have exactly one error for the reserved keyword
	if len(p.Errors()) == 0 {
		t.Fatal("Expected parse error for 'exists' keyword")
	}

	// Look for a ParserError with suggestions
	found := false
	for _, errIface := range p.Errors() {
		if perr, ok := errIface.(*ParserError); ok {
			if perr.Code == "PAR_RESERVED_KEYWORD" {
				found = true
				// Verify suggestions exist
				if len(perr.Suggestions) == 0 {
					t.Error("Expected suggestions for reserved keyword error")
				}

				// Print suggestions for visibility
				t.Logf("Error message: %s", perr.Message)
				t.Logf("Suggestions (%d):", len(perr.Suggestions))
				for i, sugg := range perr.Suggestions {
					t.Logf("  %d. %s", i+1, sugg)
				}

				// Verify suggestion content
				suggestions := strings.Join(perr.Suggestions, " ")
				if !strings.Contains(suggestions, "found") && !strings.Contains(suggestions, "doesExist") {
					t.Error("Suggestions should include alternative names")
				}
				break
			}
		}
	}

	if !found {
		t.Fatal("Expected PAR_RESERVED_KEYWORD error with suggestions")
	}
}

// BenchmarkReservedKeywordDetection measures performance of reserved keyword checking
func BenchmarkReservedKeywordDetection(b *testing.B) {
	code := `module test
func checkA() -> bool { let found = true; found }
func checkB() -> bool { let notExists = true; notExists }
func checkC() -> bool { let value = true; value }
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := lexer.New(code, "test.ail")
		p := New(l)
		_ = p.Parse()
	}
}
