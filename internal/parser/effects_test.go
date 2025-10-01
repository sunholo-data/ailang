package parser

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

func TestEffectAnnotationParsing(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedEffects []string
		shouldError    bool
		errorCode      string
	}{
		{
			name:           "single effect",
			input:          "func f() -> int ! {IO} { 42 }",
			expectedEffects: []string{"IO"},
			shouldError:    false,
		},
		{
			name:           "multiple effects",
			input:          "func f() -> int ! {IO, FS, Net} { 42 }",
			expectedEffects: []string{"IO", "FS", "Net"},
			shouldError:    false,
		},
		{
			name:           "all standard effects",
			input:          "func f() -> int ! {IO, FS, Net, Clock, Rand, DB, Trace, Async} { 42 }",
			expectedEffects: []string{"IO", "FS", "Net", "Clock", "Rand", "DB", "Trace", "Async"},
			shouldError:    false,
		},
		{
			name:           "empty effect set",
			input:          "func f() -> int ! {} { 42 }",
			expectedEffects: []string{},
			shouldError:    false,
		},
		{
			name:        "duplicate effect",
			input:       "func f() -> int ! {IO, IO} { 42 }",
			shouldError: true,
			errorCode:   "PAR_EFF001_DUP",
		},
		{
			name:        "unknown effect lowercase",
			input:       "func f() -> int ! {io} { 42 }",
			shouldError: true,
			errorCode:   "PAR_EFF002_UNKNOWN",
		},
		{
			name:        "unknown effect typo",
			input:       "func f() -> int ! {IOS} { 42 }",
			shouldError: true,
			errorCode:   "PAR_EFF002_UNKNOWN",
		},
		{
			name:        "unknown effect completely wrong",
			input:       "func f() -> int ! {Foo} { 42 }",
			shouldError: true,
			errorCode:   "PAR_EFF002_UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.ail")
			p := New(l)
			program := p.Parse()

			if tt.shouldError {
				if len(p.Errors()) == 0 {
					t.Fatalf("expected error with code %s, but got no errors", tt.errorCode)
				}

				// Check that we got the right error code
				foundError := false
				for _, err := range p.Errors() {
					if perr, ok := err.(*ParserError); ok {
						if perr.Code == tt.errorCode {
							foundError = true
							break
						}
					}
				}

				if !foundError {
					t.Errorf("expected error code %s, but got errors: %v", tt.errorCode, p.Errors())
				}
				return
			}

			if len(p.Errors()) > 0 {
				t.Fatalf("unexpected parser errors: %v", p.Errors())
			}

			if program == nil {
				t.Fatal("Parse() returned nil")
			}

			if len(program.File.Funcs) == 0 {
				t.Fatal("program has no function declarations")
			}

			// First function should have effects
			funcDecl := program.File.Funcs[0]

			// Check effects
			if len(funcDecl.Effects) != len(tt.expectedEffects) {
				t.Errorf("expected %d effects, got %d: %v", len(tt.expectedEffects), len(funcDecl.Effects), funcDecl.Effects)
			}

			for i, expected := range tt.expectedEffects {
				if i >= len(funcDecl.Effects) {
					t.Errorf("missing effect at index %d: expected %s", i, expected)
					continue
				}
				if funcDecl.Effects[i] != expected {
					t.Errorf("effect at index %d: expected %s, got %s", i, expected, funcDecl.Effects[i])
				}
			}
		})
	}
}

func TestLambdaEffectAnnotationParsing(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedEffects []string
		shouldError    bool
	}{
		{
			name:           "lambda with single effect",
			input:          "\\x. print(x) ! {IO}",
			expectedEffects: []string{"IO"},
			shouldError:    false,
		},
		{
			name:           "lambda with multiple effects",
			input:          "\\x. readFile(x) ! {IO, FS}",
			expectedEffects: []string{"IO", "FS"},
			shouldError:    false,
		},
		{
			name:           "lambda without effects",
			input:          "\\x. x + 1",
			expectedEffects: nil,
			shouldError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.ail")
			p := New(l)

			// Parse as a complete program
			prog := p.Parse()

			if tt.shouldError {
				if len(p.Errors()) == 0 {
					t.Fatal("expected error, but got no errors")
				}
				return
			}

			if len(p.Errors()) > 0 {
				t.Fatalf("unexpected parser errors: %v", p.Errors())
			}

			// Extract the expression from the program
			if prog == nil || prog.File == nil || len(prog.File.Statements) == 0 {
				t.Fatal("program has no statements")
			}

			lambda, ok := prog.File.Statements[0].(*ast.Lambda)
			if !ok {
				t.Fatalf("expected Lambda, got %T", prog.File.Statements[0])
			}

			// Check effects
			if tt.expectedEffects == nil {
				if lambda.Effects != nil && len(lambda.Effects) > 0 {
					t.Errorf("expected no effects, got %v", lambda.Effects)
				}
			} else {
				if len(lambda.Effects) != len(tt.expectedEffects) {
					t.Errorf("expected %d effects, got %d: %v", len(tt.expectedEffects), len(lambda.Effects), lambda.Effects)
				}

				for i, expected := range tt.expectedEffects {
					if i >= len(lambda.Effects) {
						t.Errorf("missing effect at index %d: expected %s", i, expected)
						continue
					}
					if lambda.Effects[i] != expected {
						t.Errorf("effect at index %d: expected %s, got %s", i, expected, lambda.Effects[i])
					}
				}
			}
		})
	}
}

func TestFunctionTypeEffectAnnotationParsing(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedEffects []string
		shouldError    bool
	}{
		{
			name:           "function type with single effect",
			input:          "let f: (int) -> string ! {IO} = undefined",
			expectedEffects: []string{"IO"},
			shouldError:    false,
		},
		{
			name:           "function type with multiple effects",
			input:          "let f: (int, string) -> bool ! {IO, FS, Net} = undefined",
			expectedEffects: []string{"IO", "FS", "Net"},
			shouldError:    false,
		},
		{
			name:           "function type without effects",
			input:          "let f: (int) -> int = undefined",
			expectedEffects: nil,
			shouldError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.ail")
			p := New(l)
			program := p.Parse()

			if tt.shouldError {
				if len(p.Errors()) == 0 {
					t.Fatal("expected error, but got no errors")
				}
				return
			}

			if len(p.Errors()) > 0 {
				t.Fatalf("unexpected parser errors: %v", p.Errors())
			}

			if program == nil {
				t.Fatal("ParseProgram() returned nil")
			}

			if len(program.File.Statements) == 0 {
				t.Fatal("program has no statements")
			}

			// First statement should be a let expression
			stmt := program.File.Statements[0]
			letExpr, ok := stmt.(*ast.Let)
			if !ok {
				t.Fatalf("expected Let, got %T", stmt)
			}

			// Type should be a FuncType
			funcType, ok := letExpr.Type.(*ast.FuncType)
			if !ok {
				t.Fatalf("expected FuncType, got %T", letExpr.Type)
			}

			// Check effects
			if tt.expectedEffects == nil {
				if funcType.Effects != nil && len(funcType.Effects) > 0 {
					t.Errorf("expected no effects, got %v", funcType.Effects)
				}
			} else {
				if len(funcType.Effects) != len(tt.expectedEffects) {
					t.Errorf("expected %d effects, got %d: %v", len(tt.expectedEffects), len(funcType.Effects), funcType.Effects)
				}

				for i, expected := range tt.expectedEffects {
					if i >= len(funcType.Effects) {
						t.Errorf("missing effect at index %d: expected %s", i, expected)
						continue
					}
					if funcType.Effects[i] != expected {
						t.Errorf("effect at index %d: expected %s, got %s", i, expected, funcType.Effects[i])
					}
				}
			}
		})
	}
}

func TestEffectAnnotationErrorMessages(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedErrorCode string
		shouldContain    string // expected substring in error message
	}{
		{
			name:             "duplicate effect suggests removal",
			input:            "func f() -> int ! {IO, IO} { 42 }",
			expectedErrorCode: "PAR_EFF001_DUP",
			shouldContain:    "duplicate",
		},
		{
			name:             "lowercase effect suggests uppercase",
			input:            "func f() -> int ! {io} { 42 }",
			expectedErrorCode: "PAR_EFF002_UNKNOWN",
			shouldContain:    "IO", // should suggest uppercase
		},
		{
			name:             "typo in effect name",
			input:            "func f() -> int ! {Nett} { 42 }",
			expectedErrorCode: "PAR_EFF002_UNKNOWN",
			shouldContain:    "Net", // should suggest Net
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.ail")
			p := New(l)
			_ = p.Parse()

			if len(p.Errors()) == 0 {
				t.Fatal("expected error, but got none")
			}

			// Find the expected error
			foundError := false
			for _, err := range p.Errors() {
				if perr, ok := err.(*ParserError); ok {
					if perr.Code == tt.expectedErrorCode {
						foundError = true

						// Check if error message contains expected substring
						if tt.shouldContain != "" {
							errorMsg := strings.ToLower(perr.Error())
							expectedSubstr := strings.ToLower(tt.shouldContain)
							if !strings.Contains(errorMsg, expectedSubstr) && !strings.Contains(perr.Fix, tt.shouldContain) {
								t.Errorf("error message or fix should contain '%s', got: %s (fix: %s)",
									tt.shouldContain, perr.Error(), perr.Fix)
							}
						}
						break
					}
				}
			}

			if !foundError {
				t.Errorf("expected error code %s, got errors: %v", tt.expectedErrorCode, p.Errors())
			}
		})
	}
}
