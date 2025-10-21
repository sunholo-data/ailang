package link

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/builtins"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/types"
)

// mockModuleLoader is a minimal ModuleLoader for testing
type mockModuleLoader struct{}

func (m *mockModuleLoader) LoadInterface(modulePath string) (*iface.Iface, error) {
	return nil, fmt.Errorf("mock loader: module %s not found", modulePath)
}

func (m *mockModuleLoader) EvaluateExport(ref core.GlobalRef) (eval.Value, error) {
	return nil, fmt.Errorf("mock loader: cannot evaluate %v", ref)
}

// TestBuiltinIface_EffectRowsPreserved verifies that effect rows survive export
// for all effect types (IO, Net, FS, Clock), not just IO
func TestBuiltinIface_EffectRowsPreserved(t *testing.T) {
	ml := NewModuleLinker(&mockModuleLoader{})
	RegisterBuiltinModule(ml)
	iface := ml.GetIface("$builtin")
	require.NotNil(t, iface, "$builtin interface not registered")

	// Test multiple effect types to ensure not IO-only
	// Note: Only testing builtins that are actually registered in the spec registry.
	// FS and Clock builtins are not yet migrated to the spec-based system (v0.3.10).
	tests := []struct {
		name   string
		effect string
	}{
		{"_io_print", "IO"},
		{"_io_println", "IO"},
		{"_net_httpRequest", "Net"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, ok := iface.Exports[tt.name]
			require.True(t, ok, "%s not exported", tt.name)

			fn, ok := item.Type.Type.(*types.TFunc2)
			require.True(t, ok, "%s not a TFunc2, got %T", tt.name, item.Type.Type)
			require.NotNil(t, fn.EffectRow, "missing effect row on %s", tt.name)
			require.NotEmpty(t, fn.EffectRow.Labels, "effect row has no labels on %s", tt.name)

			_, hasEffect := fn.EffectRow.Labels[tt.effect]
			require.True(t, hasEffect, "%s effect row missing %s label. Labels: %v",
				tt.name, tt.effect, fn.EffectRow.Labels)

			// Ensure not marked as pure if has effects
			require.False(t, item.Purity, "%s should not be pure (has effect %s)", tt.name, tt.effect)
		})
	}
}

// TestBuiltinIface_NoSilentRewrap ensures exported types preserve effect rows
// and don't silently reconstruct types without effect information
func TestBuiltinIface_NoSilentRewrap(t *testing.T) {
	specs := builtins.AllSpecs()
	ml := NewModuleLinker(&mockModuleLoader{})
	RegisterBuiltinModule(ml)
	iface := ml.GetIface("$builtin")

	// Check _io_print specifically
	spec := specs["_io_print"]
	originalType := spec.Type()
	exportedItem := iface.Exports["_io_print"]
	exportedType := exportedItem.Type.Type

	origFn, ok1 := originalType.(*types.TFunc2)
	expFn, ok2 := exportedType.(*types.TFunc2)
	require.True(t, ok1 && ok2, "both should be TFunc2")

	// If types are different pointers, effect rows MUST be preserved
	if origFn != expFn {
		require.NotNil(t, expFn.EffectRow, "exported type lost effect row")
		require.Equal(t, len(origFn.EffectRow.Labels), len(expFn.EffectRow.Labels),
			"effect row labels count changed: orig=%d, exported=%d",
			len(origFn.EffectRow.Labels), len(expFn.EffectRow.Labels))

		// Same labels present
		for label := range origFn.EffectRow.Labels {
			_, ok := expFn.EffectRow.Labels[label]
			require.True(t, ok, "exported type lost label: %s", label)
		}
	}
}

// TestBuiltinIface_PureBuiltinsHaveNoEffects verifies pure builtins don't have effect rows
func TestBuiltinIface_PureBuiltinsHaveNoEffects(t *testing.T) {
	ml := NewModuleLinker(&mockModuleLoader{})
	RegisterBuiltinModule(ml)
	iface := ml.GetIface("$builtin")

	// Test some pure builtins
	pureBuiltins := []string{
		"add_Int",
		"eq_Int",
		"show_Int",
		"intToFloat",
	}

	for _, name := range pureBuiltins {
		t.Run(name, func(t *testing.T) {
			item, ok := iface.Exports[name]
			if !ok {
				t.Skipf("%s not found (may not be registered yet)", name)
				return
			}

			// Should be marked as pure
			require.True(t, item.Purity, "%s should be marked as pure", name)

			// If it's a function, effect row should be empty or nil
			if fn, ok := item.Type.Type.(*types.TFunc2); ok {
				if fn.EffectRow != nil {
					require.Empty(t, fn.EffectRow.Labels,
						"%s is pure but has effect labels: %v", name, fn.EffectRow.Labels)
				}
			}
		})
	}
}

// TestBuiltinIface_AllRegisteredBuiltinsExported ensures all builtins from spec are exported
func TestBuiltinIface_AllRegisteredBuiltinsExported(t *testing.T) {
	specs := builtins.AllSpecs()
	ml := NewModuleLinker(&mockModuleLoader{})
	RegisterBuiltinModule(ml)
	iface := ml.GetIface("$builtin")

	missing := []string{}
	for name := range specs {
		if _, ok := iface.Exports[name]; !ok {
			missing = append(missing, name)
		}
	}

	require.Empty(t, missing, "Some builtins not exported: %v", missing)
}
