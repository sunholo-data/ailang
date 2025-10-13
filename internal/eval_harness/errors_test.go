package eval_harness

import (
	"strings"
	"testing"
)

func TestCategorizeErrorCode(t *testing.T) {
	tests := []struct {
		name      string
		stderr    string
		wantCode  ErrCode
		wantHint  bool
		hintTitle string
	}{
		{
			name:      "PAR_001: Block semicolon missing",
			stderr:    "PAR_NO_PREFIX_PARSE at benchmark/solution.ail:1:14: unexpected token in expression: =",
			wantCode:  PAR_001,
			wantHint:  true,
			hintTitle: "Parse error",
		},
		{
			name:      "TC_REC_001: Record field not found",
			stderr:    "field 'email' not found in record {name: String, age: Int}",
			wantCode:  TC_REC_001,
			wantHint:  true,
			hintTitle: "Record field missing",
		},
		{
			name:      "TC_INT_001: Modulo on Float",
			stderr:    "Float 3.14 is not an instance of Integral",
			wantCode:  TC_INT_001,
			wantHint:  true,
			hintTitle: "Modulo on Float",
		},
		{
			name:      "TC_INT_001: Modulo not defined for Float (alternative)",
			stderr:    "mod not defined for Float",
			wantCode:  TC_INT_001,
			wantHint:  true,
			hintTitle: "Modulo on Float",
		},
		{
			name:      "EQ_001: Wrong Eq dictionary",
			stderr:    "Eq dictionary resolution failed for Float",
			wantCode:  EQ_001,
			wantHint:  true,
			hintTitle: "Float equality dictionary",
		},
		{
			name:      "EQ_001: Using wrong dictionary (alternative)",
			stderr:    "using eq_Int for Float arguments",
			wantCode:  EQ_001,
			wantHint:  true,
			hintTitle: "Float equality dictionary",
		},
		{
			name:      "CAP_001: Missing capability",
			stderr:    "effect 'IO' requires capability but none provided",
			wantCode:  CAP_001,
			wantHint:  true,
			hintTitle: "Missing capability",
		},
		{
			name:      "MOD_001: Entrypoint not found",
			stderr:    "entrypoint 'main' not found in module",
			wantCode:  MOD_001,
			wantHint:  true,
			hintTitle: "Entrypoint/module resolution",
		},
		{
			name:      "MOD_001: Module not found (alternative)",
			stderr:    "module std/nonexistent not found",
			wantCode:  MOD_001,
			wantHint:  true,
			hintTitle: "Entrypoint/module resolution",
		},
		{
			name:     "Unknown error",
			stderr:   "some completely unrecognized error message",
			wantCode: "",
			wantHint: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCode, gotHint := CategorizeErrorCode(tt.stderr)

			if gotCode != tt.wantCode {
				t.Errorf("CategorizeErrorCode() code = %v, want %v", gotCode, tt.wantCode)
			}

			if tt.wantHint {
				if gotHint == nil {
					t.Errorf("CategorizeErrorCode() hint = nil, want non-nil")
				} else {
					if gotHint.Title != tt.hintTitle {
						t.Errorf("CategorizeErrorCode() hint.Title = %v, want %v", gotHint.Title, tt.hintTitle)
					}
					if gotHint.Why == "" {
						t.Errorf("CategorizeErrorCode() hint.Why is empty")
					}
					if gotHint.How == "" {
						t.Errorf("CategorizeErrorCode() hint.How is empty")
					}
				}
			} else {
				if gotHint != nil {
					t.Errorf("CategorizeErrorCode() hint = %v, want nil", gotHint)
				}
			}
		})
	}
}

func TestFormatRepairPrompt(t *testing.T) {
	tests := []struct {
		name         string
		code         ErrCode
		stderr       string
		benchmarkID  string
		lang         string
		wantContains []string
	}{
		{
			name:        "PAR_001 repair prompt",
			code:        PAR_001,
			stderr:      "PAR_NO_PREFIX_PARSE at benchmark/solution.ail:1:14: unexpected token",
			benchmarkID: "test_blocks",
			lang:        "ailang",
			wantContains: []string{
				"<PAR_001>",
				"Parse error",
				"AILANG syntax error",
				"semicolons",
				"ailang program",
				"test_blocks",
			},
		},
		{
			name:        "EQ_001 repair prompt",
			code:        EQ_001,
			stderr:      "Eq dictionary resolution failed for Float",
			benchmarkID: "float_comparison",
			lang:        "ailang",
			wantContains: []string{
				"<EQ_001>",
				"Float equality",
				"Eq dictionary must match",
				"Annotate as",
				"float_comparison",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the actual hint from the Rules
			_, hint := CategorizeErrorCode(tt.stderr)
			if hint == nil {
				t.Fatalf("CategorizeErrorCode() returned nil hint for %s", tt.stderr)
			}

			// Provide example failed code
			failedCode := "let x = 1\nprint(x)"

			got := FormatRepairPrompt(tt.code, hint, tt.benchmarkID, tt.lang, failedCode, tt.stderr)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("FormatRepairPrompt() missing %q in output:\n%s", want, got)
				}
			}

			// Verify structure includes error context
			if !strings.Contains(got, "ERROR:") {
				t.Errorf("FormatRepairPrompt() missing ERROR section")
			}
			if !strings.Contains(got, "YOUR PREVIOUS CODE:") {
				t.Errorf("FormatRepairPrompt() missing code section")
			}
			if !strings.Contains(got, "DIAGNOSIS:") {
				t.Errorf("FormatRepairPrompt() missing diagnosis")
			}
			if !strings.Contains(got, "Please produce a corrected") {
				t.Errorf("FormatRepairPrompt() missing instructions")
			}
		})
	}
}

func TestAllErrorCodesHaveRules(t *testing.T) {
	// Ensure every error code constant has a corresponding rule
	allCodes := []ErrCode{PAR_001, TC_REC_001, TC_INT_001, EQ_001, CAP_001, MOD_001}

	foundCodes := make(map[ErrCode]bool)
	for _, rule := range Rules {
		foundCodes[rule.Code] = true
	}

	for _, code := range allCodes {
		if !foundCodes[code] {
			t.Errorf("Error code %s has no rule defined", code)
		}
	}
}

func TestAllRulesHaveCompleteHints(t *testing.T) {
	for _, rule := range Rules {
		if rule.Hint.Title == "" {
			t.Errorf("Rule for %s has empty Title", rule.Code)
		}
		if rule.Hint.Why == "" {
			t.Errorf("Rule for %s has empty Why", rule.Code)
		}
		if rule.Hint.How == "" {
			t.Errorf("Rule for %s has empty How", rule.Code)
		}
		if rule.Re == nil {
			t.Errorf("Rule for %s has nil regex", rule.Code)
		}
	}
}

func TestRegexPatternsCompile(t *testing.T) {
	// All patterns should already be compiled, but verify they match something
	testCases := map[ErrCode]string{
		PAR_001:    "parse error: unexpected token near",
		TC_REC_001: "field 'x' not found in record {}",
		TC_INT_001: "Float 1.5 is not an instance of Integral",
		EQ_001:     "Eq dictionary resolution failed",
		CAP_001:    "effect 'IO' requires capability",
		MOD_001:    "entrypoint 'main' not found",
	}

	for code, testInput := range testCases {
		found := false
		for _, rule := range Rules {
			if rule.Code == code && rule.Re.MatchString(testInput) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("No rule for %s matches test input: %s", code, testInput)
		}
	}
}
