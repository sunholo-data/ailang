package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/types"
)

// TestREPLSmoke_TypeCommand is a CRITICAL regression guard for REPL env initialization.
//
// The v0.3.10 bug also affected the REPL - builtin effects were missing from the
// type environment, causing :type queries to show incorrect types.
//
// This test ensures:
// - REPL initializes with correct builtin types
// - :type command shows effect rows for IO/Net builtins
// - Type environment matches the spec registry
//
// If this test fails, it means the REPL's type environment initialization is broken.
func TestREPLSmoke_TypeCommand(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		mustContain    []string // All of these must appear in output
		mustNotContain []string // None of these must appear
	}{
		{
			name:    ":type _io_print shows ! {IO}",
			command: ":type _io_print",
			mustContain: []string{
				"! {IO}", // CRITICAL: Effect row must be present
				"String", // Parameter type
			},
			mustNotContain: []string{
				"error",
				"unbound",
			},
		},
		{
			name:    ":type _io_println shows ! {IO}",
			command: ":type _io_println",
			mustContain: []string{
				"! {IO}",
				"String",
			},
		},
		{
			name:    ":type _io_readLine shows ! {IO}",
			command: ":type _io_readLine",
			mustContain: []string{
				"! {IO}",
				"String", // Return type
			},
		},
		{
			name:    ":type _net_httpRequest shows ! {Net}",
			command: ":type _net_httpRequest",
			mustContain: []string{
				"! {Net}", // CRITICAL: Net effect must be present
				"String",
			},
		},
		{
			name:    ":type _str_len is pure (no effects)",
			command: ":type _str_len",
			mustContain: []string{
				"String",
				"Int",
			},
			mustNotContain: []string{
				"! {", // No effect row for pure functions
			},
		},
		{
			name:    ":type concat_String is pure",
			command: ":type concat_String",
			mustContain: []string{
				"String",
			},
			mustNotContain: []string{
				"! {",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh REPL instance
			repl := New()

			// Capture output
			var buf bytes.Buffer

			// Execute command
			repl.HandleCommand(tt.command, &buf)
			output := buf.String()

			// Check required strings
			for _, required := range tt.mustContain {
				assert.Contains(t, output, required,
					"REPL output missing required text: %q\nCommand: %s\nOutput:\n%s",
					required, tt.command, output)
			}

			// Check forbidden strings
			for _, forbidden := range tt.mustNotContain {
				assert.NotContains(t, output, forbidden,
					"REPL output contains forbidden text: %q\nCommand: %s\nOutput:\n%s",
					forbidden, tt.command, output)
			}

			// Special check: If testing IO/Net builtin, ensure effect is NOT missing
			if strings.Contains(tt.command, "_io_") && !strings.Contains(output, "! {IO}") {
				t.Fatalf("REGRESSION: _io_* builtin lost {IO} effect!\n"+
					"This is the v0.3.10 bug recurring in REPL.\n"+
					"Command: %s\nOutput:\n%s",
					tt.command, output)
			}

			if strings.Contains(tt.command, "_net_") && !strings.Contains(output, "! {Net}") {
				t.Fatalf("REGRESSION: _net_* builtin lost {Net} effect!\n"+
					"This is the v0.3.10 bug recurring in REPL.\n"+
					"Command: %s\nOutput:\n%s",
					tt.command, output)
			}
		})
	}

	t.Logf("✅ All %d REPL :type smoke tests passed", len(tests))
}

// TestREPLSmoke_EnvInitialization verifies REPL initializes with all builtins.
//
// This is a sanity check that NewTypeEnvWithBuiltins() is called during REPL creation
// and that the type environment contains the expected builtins.
func TestREPLSmoke_EnvInitialization(t *testing.T) {
	repl := New()

	// Check that type env is not nil
	require.NotNil(t, repl.typeEnv, "REPL type environment is nil")

	// Try to lookup critical builtins
	criticalBuiltins := []string{
		"_io_print",
		"_io_println",
		"_io_readLine",
		"_net_httpRequest",
		"_str_len",
		"concat_String",
	}

	for _, name := range criticalBuiltins {
		t.Run(name, func(t *testing.T) {
			// Lookup in type env
			binding, err := repl.typeEnv.Lookup(name)
			require.NoError(t, err, "Builtin %s missing from REPL type env", name)
			require.NotNil(t, binding, "Builtin %s has nil binding", name)

			t.Logf("✅ %s found in REPL type env", name)
		})
	}

	t.Logf("✅ All %d critical builtins present in REPL env", len(criticalBuiltins))
}

// TestREPLSmoke_EffectRowPreservation is the most focused regression test.
//
// This directly checks that effect rows survive the REPL initialization path:
//
//	builtins.AllSpecs() → link.RegisterBuiltinModule() → types.NewTypeEnvWithBuiltins()
//
// If this fails, effect rows are being lost somewhere in the chain.
func TestREPLSmoke_EffectRowPreservation(t *testing.T) {
	repl := New()

	// Check _io_print specifically (the canary for v0.3.10 bug)
	binding, err := repl.typeEnv.Lookup("_io_print")
	require.NoError(t, err, "_io_print missing from REPL env")

	// Extract type from scheme
	scheme, ok := binding.(*types.Scheme)
	require.True(t, ok, "_io_print should be a Scheme")

	// Check it's a function type
	fn, ok := scheme.Type.(*types.TFunc2)
	require.True(t, ok, "_io_print should have TFunc2 type")

	// CRITICAL: Check effect row exists and contains IO
	require.NotNil(t, fn.EffectRow, "_io_print effect row is nil (v0.3.10 regression!)")
	require.NotEmpty(t, fn.EffectRow.Labels, "_io_print effect row has no labels (v0.3.10 regression!)")

	_, hasIO := fn.EffectRow.Labels["IO"]
	assert.True(t, hasIO, "_io_print missing IO effect (v0.3.10 regression!)")

	t.Log("✅ _io_print effect row preserved: ! {IO}")
}
