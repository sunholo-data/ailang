package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

// TestResultMatchArmUnification is a regression test for the bug where importing
// Result from std/result and using Ok/Err in different match arms would fail
// with type unification errors like "cannot unify type constructors: () vs string"
// or "No instance for Num[string]".
//
// Root cause: astTypeToInternalType in iface/builder.go was missing the
// *ast.TypeVar case, causing constructor field type variables (e.g., 'a' in Ok(a))
// to become TVar2{Name: "unknown"} instead of TVar2{Name: "a"}.
func TestResultMatchArmUnification(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-result-match-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create std/ directory with result.ail
	stdDir := filepath.Join(tempDir, "std")
	if err := os.MkdirAll(stdDir, 0755); err != nil {
		t.Fatalf("failed to create std dir: %v", err)
	}

	resultContent := `module std/result
export type Result[a, e] = Ok(a) | Err(e)
`
	if err := os.WriteFile(filepath.Join(stdDir, "result.ail"), []byte(resultContent), 0644); err != nil {
		t.Fatalf("failed to write result.ail: %v", err)
	}

	tests := []struct {
		name    string
		content string
	}{
		{
			name: "Result[unit, string] match with Ok and Err",
			content: `module test
import std/result (Result, Ok, Err)
export func t1(x: bool) -> Result[unit, string] {
  match x {
    true => Ok(()),
    false => Err("bad")
  }
}
`,
		},
		{
			name: "Result[int, string] match with Ok and Err",
			content: `module test
import std/result (Result, Ok, Err)
export func t2(x: bool) -> Result[int, string] {
  match x {
    true => Ok(42),
    false => Err("bad")
  }
}
`,
		},
		{
			name: "Result[int, string] match Err first then Ok",
			content: `module test
import std/result (Result, Ok, Err)
export func t3(x: bool) -> Result[int, string] {
  match x {
    true => Err("bad"),
    false => Ok(42)
  }
}
`,
		},
		{
			name: "Result without return type annotation",
			content: `module test
import std/result (Ok, Err)
export func t4(x: bool) {
  match x {
    true => Ok(42),
    false => Err("bad")
  }
}
`,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write test module
			testFile := filepath.Join(tempDir, "test.ail")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test.ail: %v", err)
			}

			// Change to temp dir so std/ import resolves
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get working directory: %v", err)
			}
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("failed to change to temp dir: %v", err)
			}
			defer os.Chdir(originalDir)

			// Compile - should succeed without type errors
			src := Source{Filename: "test.ail"}
			cfg := Config{Mode: ModeCheck}

			_, err = Run(cfg, src)
			if err != nil {
				t.Fatalf("test case %d (%s): Run failed: %v", i, tt.name, err)
			}
		})
	}
}
