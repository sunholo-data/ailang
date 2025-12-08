package builtins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/effects/testctx"
	"github.com/sunholo/ailang/internal/eval"
)

// ============================================================================
// _env_getArgs Tests
// ============================================================================

func TestEnvGetArgs_EmptyArgs(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	ctx.GrantAll("Env")
	ctx.EffContext.Args = []string{} // No CLI args

	result, err := envGetArgsImpl(ctx.EffContext, []eval.Value{&eval.UnitValue{}})

	assert.NoError(t, err)
	listVal, ok := result.(*eval.ListValue)
	assert.True(t, ok, "Result should be a ListValue")
	assert.Empty(t, listVal.Elements, "Should return empty list when no args")
}

func TestEnvGetArgs_SingleArg(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	ctx.GrantAll("Env")
	ctx.EffContext.Args = []string{"arg1"}

	result, err := envGetArgsImpl(ctx.EffContext, []eval.Value{&eval.UnitValue{}})

	assert.NoError(t, err)
	listVal, ok := result.(*eval.ListValue)
	assert.True(t, ok, "Result should be a ListValue")
	assert.Len(t, listVal.Elements, 1, "Should have 1 element")

	elem0, ok := listVal.Elements[0].(*eval.StringValue)
	assert.True(t, ok, "Element should be StringValue")
	assert.Equal(t, "arg1", elem0.Value)
}

func TestEnvGetArgs_MultipleArgs(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	ctx.GrantAll("Env")
	ctx.EffContext.Args = []string{"arg1", "arg2", "arg3"}

	result, err := envGetArgsImpl(ctx.EffContext, []eval.Value{&eval.UnitValue{}})

	assert.NoError(t, err)
	listVal, ok := result.(*eval.ListValue)
	assert.True(t, ok, "Result should be a ListValue")
	assert.Len(t, listVal.Elements, 3, "Should have 3 elements")

	elem0, ok := listVal.Elements[0].(*eval.StringValue)
	assert.True(t, ok, "Element 0 should be StringValue")
	assert.Equal(t, "arg1", elem0.Value)

	elem1, ok := listVal.Elements[1].(*eval.StringValue)
	assert.True(t, ok, "Element 1 should be StringValue")
	assert.Equal(t, "arg2", elem1.Value)

	elem2, ok := listVal.Elements[2].(*eval.StringValue)
	assert.True(t, ok, "Element 2 should be StringValue")
	assert.Equal(t, "arg3", elem2.Value)
}

func TestEnvGetArgs_ArgsWithSpaces(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	ctx.GrantAll("Env")
	ctx.EffContext.Args = []string{"hello world", "foo bar"}

	result, err := envGetArgsImpl(ctx.EffContext, []eval.Value{&eval.UnitValue{}})

	assert.NoError(t, err)
	listVal, ok := result.(*eval.ListValue)
	assert.True(t, ok, "Result should be a ListValue")
	assert.Len(t, listVal.Elements, 2, "Should have 2 elements")

	elem0, ok := listVal.Elements[0].(*eval.StringValue)
	assert.True(t, ok, "Element 0 should be StringValue")
	assert.Equal(t, "hello world", elem0.Value)

	elem1, ok := listVal.Elements[1].(*eval.StringValue)
	assert.True(t, ok, "Element 1 should be StringValue")
	assert.Equal(t, "foo bar", elem1.Value)
}

func TestEnvGetArgs_ArgsWithSpecialChars(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	ctx.GrantAll("Env")
	ctx.EffContext.Args = []string{
		"--flag=value",
		"-x",
		"path/to/file.txt",
		"key=value",
		"@special!chars#",
	}

	result, err := envGetArgsImpl(ctx.EffContext, []eval.Value{&eval.UnitValue{}})

	assert.NoError(t, err)
	listVal, ok := result.(*eval.ListValue)
	assert.True(t, ok, "Result should be a ListValue")
	assert.Len(t, listVal.Elements, 5, "Should have 5 elements")

	expected := []string{
		"--flag=value",
		"-x",
		"path/to/file.txt",
		"key=value",
		"@special!chars#",
	}

	for i, exp := range expected {
		elem, ok := listVal.Elements[i].(*eval.StringValue)
		assert.True(t, ok, "Element %d should be StringValue", i)
		assert.Equal(t, exp, elem.Value, "Element %d value mismatch", i)
	}
}

func TestEnvGetArgs_RequiresCapability(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	// Don't grant Env capability
	ctx.EffContext.Args = []string{"arg1", "arg2"}

	_, err := envGetArgsImpl(ctx.EffContext, []eval.Value{&eval.UnitValue{}})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "effect 'Env' requires capability")
	assert.Contains(t, err.Error(), "--caps Env")
}

func TestEnvGetArgs_UnitArgument(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	ctx.GrantAll("Env")
	ctx.EffContext.Args = []string{"arg1"}

	// Unit-argument model: should accept exactly one unit argument
	result, err := envGetArgsImpl(ctx.EffContext, []eval.Value{&eval.UnitValue{}})

	assert.NoError(t, err)
	listVal, ok := result.(*eval.ListValue)
	assert.True(t, ok, "Result should be a ListValue")
	assert.Len(t, listVal.Elements, 1, "Should have 1 element")
}

// ============================================================================
// Helper function to get envGetArgsImpl
// ============================================================================

// envGetArgsImpl wraps the builtin implementation for testing
func envGetArgsImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	return effects.Call(ctx, "Env", "getArgs", args)
}
