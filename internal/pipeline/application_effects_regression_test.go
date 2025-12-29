package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/types"
)

// TestApplicationEffects_Propagation is a regression test suite for the v0.3.11 bug fix.
// The bug was that function application included getEffectRow(funcNode) which is always
// empty for variable references, causing missing/spurious effects.
//
// This test ensures that application effects = argument effects + function type's effect row.
//
// NOTE: This test uses unit-level construction of AST nodes rather than parsing full programs,
// to isolate the effect propagation logic in inferApp().
// TestApplicationEffects_BuiltinEnvAvailable verifies that the builtin environment
// is properly initialized and contains expected functions.
//
// This is a simplified test - the full effect propagation is tested via stdlib_canary_test
// and the row_unification_regression_test.
func TestApplicationEffects_BuiltinEnvAvailable(t *testing.T) {
	env := types.NewTypeEnvWithBuiltins()

	// Verify _io_print is available
	printScheme, err := env.Lookup("_io_print")
	require.NoError(t, err, "_io_print should be in builtin environment")
	require.NotNil(t, printScheme, "_io_print should have a type")

	// Verify it's a scheme (polymorphic type)
	if scheme, ok := printScheme.(*types.Scheme); ok {
		// Should be a function String -> () ! {IO}
		assert.NotNil(t, scheme.Type, "_io_print should have underlying type")
	}
}
