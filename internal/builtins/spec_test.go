package builtins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// mockImpl is a no-op implementation for testing
func mockImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	return &eval.UnitValue{}, nil
}

// mockType returns a simple function type for testing
func mockType() types.Type {
	return &types.TFunc2{
		Params: []types.Type{
			&types.TCon{Name: "String"},
		},
		Return: &types.TCon{Name: "String"},
		EffectRow: &types.Row{
			Kind:   types.KRow{ElemKind: types.KEffect{}},
			Labels: make(map[string]types.Type),
			Tail:   nil,
		},
	}
}

func TestRegisterEffectBuiltin_Success(t *testing.T) {
	// Reset registry for testing
	specRegistry = make(map[string]*BuiltinSpec)
	frozen = false

	spec := BuiltinSpec{
		Module:  "std/test",
		Name:    "_test_func",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    mockType,
		Impl:    mockImpl,
	}

	err := RegisterEffectBuiltin(spec)
	require.NoError(t, err)

	// Verify registration
	registered, ok := GetSpec("_test_func")
	require.True(t, ok)
	assert.Equal(t, "std/test", registered.Module)
	assert.Equal(t, "_test_func", registered.Name)
	assert.Equal(t, 1, registered.NumArgs)
	assert.True(t, registered.IsPure)
}

func TestRegisterEffectBuiltin_EmptyName(t *testing.T) {
	specRegistry = make(map[string]*BuiltinSpec)
	frozen = false

	spec := BuiltinSpec{
		Module:  "std/test",
		Name:    "", // Empty name
		NumArgs: 1,
		IsPure:  true,
		Type:    mockType,
		Impl:    mockImpl,
	}

	err := RegisterEffectBuiltin(spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestRegisterEffectBuiltin_NilTypeFunc(t *testing.T) {
	specRegistry = make(map[string]*BuiltinSpec)
	frozen = false

	spec := BuiltinSpec{
		Module:  "std/test",
		Name:    "_test_func",
		NumArgs: 1,
		IsPure:  true,
		Type:    nil, // Nil type function
		Impl:    mockImpl,
	}

	err := RegisterEffectBuiltin(spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Type function is nil")
}

func TestRegisterEffectBuiltin_TypeReturnsNil(t *testing.T) {
	specRegistry = make(map[string]*BuiltinSpec)
	frozen = false

	spec := BuiltinSpec{
		Module:  "std/test",
		Name:    "_test_func",
		NumArgs: 1,
		IsPure:  true,
		Type: func() types.Type {
			return nil // Returns nil
		},
		Impl: mockImpl,
	}

	err := RegisterEffectBuiltin(spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Type() returned nil")
}

func TestRegisterEffectBuiltin_ArityMismatch(t *testing.T) {
	specRegistry = make(map[string]*BuiltinSpec)
	frozen = false

	spec := BuiltinSpec{
		Module:  "std/test",
		Name:    "_test_func",
		NumArgs: 2, // Says 2 args
		IsPure:  true,
		Type:    mockType, // But type has 1 arg
		Impl:    mockImpl,
	}

	err := RegisterEffectBuiltin(spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "NumArgs=2 but type signature has 1 arguments")
}

func TestRegisterEffectBuiltin_NilImpl(t *testing.T) {
	specRegistry = make(map[string]*BuiltinSpec)
	frozen = false

	spec := BuiltinSpec{
		Module:  "std/test",
		Name:    "_test_func",
		NumArgs: 1,
		IsPure:  true,
		Type:    mockType,
		Impl:    nil, // Nil implementation
	}

	err := RegisterEffectBuiltin(spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Impl function is nil")
}

func TestRegisterEffectBuiltin_Duplicate(t *testing.T) {
	specRegistry = make(map[string]*BuiltinSpec)
	frozen = false

	spec := BuiltinSpec{
		Module:  "std/test",
		Name:    "_test_func",
		NumArgs: 1,
		IsPure:  true,
		Type:    mockType,
		Impl:    mockImpl,
	}

	// First registration succeeds
	err := RegisterEffectBuiltin(spec)
	require.NoError(t, err)

	// Second registration fails
	err = RegisterEffectBuiltin(spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestRegisterEffectBuiltin_AfterFreeze(t *testing.T) {
	specRegistry = make(map[string]*BuiltinSpec)
	frozen = false

	// Freeze the registry
	err := Init()
	require.NoError(t, err)

	// Try to register after freeze
	spec := BuiltinSpec{
		Module:  "std/test",
		Name:    "_test_func",
		NumArgs: 1,
		IsPure:  true,
		Type:    mockType,
		Impl:    mockImpl,
	}

	err = RegisterEffectBuiltin(spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "registry is frozen")
}

func TestAllSpecs(t *testing.T) {
	specRegistry = make(map[string]*BuiltinSpec)
	frozen = false

	// Register multiple builtins
	specs := []BuiltinSpec{
		{
			Module:  "std/test",
			Name:    "_test_func1",
			NumArgs: 1,
			IsPure:  true,
			Type:    mockType,
			Impl:    mockImpl,
		},
		{
			Module:  "std/test",
			Name:    "_test_func2",
			NumArgs: 1,
			IsPure:  false,
			Effect:  "Test",
			Type:    mockType,
			Impl:    mockImpl,
		},
	}

	for _, spec := range specs {
		err := RegisterEffectBuiltin(spec)
		require.NoError(t, err)
	}

	// Get all specs
	all := AllSpecs()
	assert.Len(t, all, 2)
	assert.Contains(t, all, "_test_func1")
	assert.Contains(t, all, "_test_func2")
}

func TestAllNames(t *testing.T) {
	specRegistry = make(map[string]*BuiltinSpec)
	frozen = false

	// Register multiple builtins
	names := []string{"_test_func1", "_test_func2", "_test_func3"}
	for _, name := range names {
		spec := BuiltinSpec{
			Module:  "std/test",
			Name:    name,
			NumArgs: 1,
			IsPure:  true,
			Type:    mockType,
			Impl:    mockImpl,
		}
		err := RegisterEffectBuiltin(spec)
		require.NoError(t, err)
	}

	// Get all names
	allNames := AllNames()
	assert.Len(t, allNames, 3)
	for _, name := range names {
		assert.Contains(t, allNames, name)
	}
}

func TestInit(t *testing.T) {
	specRegistry = make(map[string]*BuiltinSpec)
	frozen = false

	assert.False(t, IsFrozen())

	err := Init()
	require.NoError(t, err)

	assert.True(t, IsFrozen())

	// Second init should fail
	err = Init()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already initialized")
}
