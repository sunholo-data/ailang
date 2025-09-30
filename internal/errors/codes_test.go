package errors

import (
	"testing"
)

func TestErrorCodeTaxonomy(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		phase    string
		category string
	}{
		// Parser errors
		{"PAR001", PAR001, "parser", "syntax"},
		{"PAR003", PAR003, "parser", "syntax"},
		{"PAR010", PAR010, "parser", "syntax"},

		// Module errors
		{"MOD001", MOD001, "module", "structure"},
		{"MOD003", MOD003, "module", "feature"},
		{"MOD004", MOD004, "module", "namespace"},

		// Loader errors
		{"LDR001", LDR001, "loader", "resolution"},
		{"LDR002", LDR002, "loader", "dependency"},

		// Type checking errors
		{"TC001", TC001, "typecheck", "type"},
		{"TC007", TC007, "typecheck", "defaulting"},
		{"TC009", TC009, "typecheck", "effect"},

		// Runtime errors
		{"RT001", RT001, "runtime", "arithmetic"},
		{"RT005", RT005, "runtime", "stack"},

		// Evaluation errors
		{"EVA001", EVA001, "eval", "scope"},
		{"EVA004", EVA004, "eval", "effect"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, exists := GetErrorInfo(tt.code)
			if !exists {
				t.Errorf("Error code %s not found in registry", tt.code)
				return
			}

			if info.Code != tt.code {
				t.Errorf("Code mismatch: got %s, want %s", info.Code, tt.code)
			}

			if info.Phase != tt.phase {
				t.Errorf("Phase mismatch for %s: got %s, want %s", tt.code, info.Phase, tt.phase)
			}

			if info.Category != tt.category {
				t.Errorf("Category mismatch for %s: got %s, want %s", tt.code, info.Category, tt.category)
			}
		})
	}
}

func TestErrorTypeCheckers(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		isParser  bool
		isModule  bool
		isLoader  bool
		isType    bool
		isRuntime bool
	}{
		{"Parser error", PAR001, true, false, false, false, false},
		{"Module error", MOD001, false, true, false, false, false},
		{"Loader error", LDR001, false, false, true, false, false},
		{"Type error", TC001, false, false, false, true, false},
		{"Runtime error", RT001, false, false, false, false, true},
		{"Eval error", EVA001, false, false, false, false, true}, // Eval counts as runtime
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsParserError(tt.code); got != tt.isParser {
				t.Errorf("IsParserError(%s) = %v, want %v", tt.code, got, tt.isParser)
			}

			if got := IsModuleError(tt.code); got != tt.isModule {
				t.Errorf("IsModuleError(%s) = %v, want %v", tt.code, got, tt.isModule)
			}

			if got := IsLoaderError(tt.code); got != tt.isLoader {
				t.Errorf("IsLoaderError(%s) = %v, want %v", tt.code, got, tt.isLoader)
			}

			if got := IsTypeError(tt.code); got != tt.isType {
				t.Errorf("IsTypeError(%s) = %v, want %v", tt.code, got, tt.isType)
			}

			if got := IsRuntimeError(tt.code); got != tt.isRuntime {
				t.Errorf("IsRuntimeError(%s) = %v, want %v", tt.code, got, tt.isRuntime)
			}
		})
	}
}

func TestAllErrorCodesInRegistry(t *testing.T) {
	// List of all error codes that should be in the registry
	allCodes := []string{
		// Parser
		PAR001, PAR002, PAR003, PAR004, PAR005, PAR006, PAR007, PAR008, PAR009, PAR010,
		// Module
		MOD001, MOD002, MOD003, MOD004, MOD005,
		// Loader
		LDR001, LDR002, LDR003, LDR004, LDR005,
		// Desugar
		DSG001, DSG002, DSG003,
		// Type checking
		TC001, TC002, TC003, TC004, TC005, TC006, TC007, TC008, TC009, TC010,
		// Elaboration
		ELB001, ELB002, ELB003, ELB004, ELB005, ELB006,
		// Linking
		LNK001, LNK002, LNK003, LNK004, LNK005,
		// Evaluation
		EVA001, EVA002, EVA003, EVA004, EVA005,
		// Runtime
		RT001, RT002, RT003, RT004, RT005, RT006, RT007, RT008, RT009,
	}

	for _, code := range allCodes {
		t.Run(code, func(t *testing.T) {
			_, exists := GetErrorInfo(code)
			if !exists {
				t.Errorf("Error code %s is defined but not in registry", code)
			}
		})
	}

	// Check that we have at least the expected number of codes
	// (registry may have more as new codes are added)
	if len(ErrorRegistry) < len(allCodes) {
		t.Errorf("Registry has %d codes, expected at least %d", len(ErrorRegistry), len(allCodes))
	}
}

func TestErrorInfoConsistency(t *testing.T) {
	// Check that all error codes follow naming conventions
	for code, info := range ErrorRegistry {
		// Code should match the key
		if info.Code != code {
			t.Errorf("Code mismatch in registry: key=%s, info.Code=%s", code, info.Code)
		}

		// Check code format (PREFIX###)
		if len(code) < 4 || len(code) > 6 {
			t.Errorf("Invalid code format: %s", code)
		}

		// Check phase is valid
		validPhases := map[string]bool{
			"parser": true, "module": true, "loader": true, "desugar": true,
			"typecheck": true, "elaborate": true, "link": true,
			"eval": true, "runtime": true, "import": true,
		}
		if !validPhases[info.Phase] {
			t.Errorf("Invalid phase for %s: %s", code, info.Phase)
		}

		// Check description is not empty
		if info.Description == "" {
			t.Errorf("Empty description for %s", code)
		}
	}
}
