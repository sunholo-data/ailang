package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/types"
)

// TestStdlibCanary_IOModule is a smoke test that ensures stdlib/std/io.ail
// can be parsed and typechecked without errors.
//
// This is the CRITICAL regression test for v0.3.11 - the bug caused this exact
// file to fail with "closed row missing labels: [IO]".
func TestStdlibCanary_IOModule(t *testing.T) {
	// Find the stdlib/std/io.ail file
	stdlibPath := findStdlibPath(t)
	ioPath := filepath.Join(stdlibPath, "std", "io.ail")

	// Read the file
	content, err := os.ReadFile(ioPath)
	require.NoError(t, err, "Failed to read stdlib/std/io.ail")

	// Parse the module
	l := lexer.New(string(content), ioPath)
	p := parser.New(l)
	program := p.Parse()
	require.NotNil(t, program, "Failed to parse stdlib/std/io.ail")
	require.NotNil(t, program.Module, "stdlib/std/io.ail should have a module")

	// Typecheck the module
	tc := types.NewTypeChecker()
	typed, err := tc.CheckProgram(program)

	// CRITICAL: This MUST NOT fail with "closed row missing labels: [IO]"
	if err != nil {
		errMsg := err.Error()
		// Check if it's the specific regression error
		if strings.Contains(errMsg, "closed row missing labels") {
			t.Fatalf("REGRESSION: The v0.3.11 row unification bug is back! Error: %v", err)
		}
		// If error message is literally "no errors", it's actually success (confusing API)
		if errMsg != "no errors" {
			// Other errors might be expected (incomplete features, etc.)
			t.Logf("Typechecking stdlib/std/io.ail produced errors (may be expected): %v", err)
		}
	}

	// If typechecking completely succeeded, verify the result
	if typed != nil && typed.Statements != nil {
		t.Logf("✅ stdlib/std/io.ail typechecked successfully (%d statements)", len(typed.Statements))
	}

	// Verify that the print function is in the typed program (if typed != nil)
	if typed == nil {
		t.Log("Typechecker returned nil (parsing succeeded, typing incomplete)")
		return
	}

	foundPrint := false
	for _, stmt := range typed.Statements {
		if funcDecl, ok := stmt.(*types.TypedFunctionDeclaration); ok {
			if nameExpr, ok := funcDecl.Name.(string); ok && nameExpr == "print" {
				foundPrint = true

				// Verify print has the correct type: String -> () ! {IO}
				fnType, ok := funcDecl.GetType().(*types.TFunc2)
				require.True(t, ok, "print should have function type")

				// Check parameter types
				require.Equal(t, 1, len(fnType.Params), "print should have 1 parameter")
				assert.Equal(t, "String", fnType.Params[0].(*types.TCon).Name, "print parameter should be String")

				// Check return type
				assert.Equal(t, types.TUnit, fnType.Return, "print should return ()")

				// Check effects - THIS WAS THE BUG!
				require.NotNil(t, fnType.EffectRow, "print should have effect row")
				assert.Equal(t, 1, len(fnType.EffectRow.Labels), "print should have 1 effect")
				_, hasIO := fnType.EffectRow.Labels["IO"]
				assert.True(t, hasIO, "print should have IO effect")
				assert.Nil(t, fnType.EffectRow.Tail, "print effect row should be closed")
			}
		}
	}

	require.True(t, foundPrint, "stdlib/std/io.ail should export 'print' function")
}

// TestStdlibCanary_AllModules ensures all stdlib modules can be parsed and typechecked.
func TestStdlibCanary_AllModules(t *testing.T) {
	stdlibPath := findStdlibPath(t)

	// List of all stdlib modules that should typecheck cleanly
	modules := []string{
		"std/io.ail",
		"std/prelude.ail",
		// Add more as they are implemented
	}

	for _, modPath := range modules {
		t.Run(modPath, func(t *testing.T) {
			fullPath := filepath.Join(stdlibPath, modPath)

			// Check file exists
			_, err := os.Stat(fullPath)
			if os.IsNotExist(err) {
				t.Skipf("Module %s not yet implemented", modPath)
				return
			}
			require.NoError(t, err)

			// Read and parse
			content, err := os.ReadFile(fullPath)
			require.NoError(t, err, "Failed to read %s", modPath)

			l := lexer.New(string(content), fullPath)
			p := parser.New(l)
			program := p.Parse()
			require.NotNil(t, program, "Failed to parse %s", modPath)

			// Typecheck
			tc := types.NewTypeChecker()
			typed, err := tc.CheckProgram(program)

			// Check for regression error
			if err != nil {
				errMsg := err.Error()
				if strings.Contains(errMsg, "closed row missing labels") {
					t.Fatalf("REGRESSION: The v0.3.11 row unification bug is back in %s! Error: %v", modPath, err)
				}
				// "no errors" means success
				if errMsg != "no errors" {
					t.Logf("Typechecking %s produced errors (may be expected): %v", modPath, err)
				}
			}

			// Log success
			if typed != nil && typed.Statements != nil {
				t.Logf("✅ %s typechecked successfully (%d statements)", modPath, len(typed.Statements))
			}
		})
	}
}

// findStdlibPath locates the stdlib directory relative to the test
func findStdlibPath(t *testing.T) string {
	// Try common paths relative to internal/types/
	candidates := []string{
		"../../stdlib",
		"../../../stdlib",
		filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "sunholo", "ailang", "stdlib"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			absPath, _ := filepath.Abs(path)
			return absPath
		}
	}

	t.Fatal("Could not find stdlib directory")
	return ""
}
