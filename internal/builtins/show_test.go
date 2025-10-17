package builtins

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/effects/testctx"
	"github.com/sunholo/ailang/internal/eval"
)

func TestShow_Primitives(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		input    eval.Value
		expected string
	}{
		// Integers
		{"int positive", testctx.MakeInt(42), "42"},
		{"int negative", testctx.MakeInt(-17), "-17"},
		{"int zero", testctx.MakeInt(0), "0"},
		{"int large", testctx.MakeInt(1000000), "1000000"},

		// Floats
		{"float positive", testctx.MakeFloat(3.14), "3.14"},
		{"float negative", testctx.MakeFloat(-2.5), "-2.5"},
		{"float zero", testctx.MakeFloat(0.0), "0"},
		{"float integer-like", testctx.MakeFloat(5.0), "5"},
		{"float small", testctx.MakeFloat(0.001), "0.001"},
		{"float large", testctx.MakeFloat(123456.789), "123456.789"},

		// Booleans
		{"bool true", testctx.MakeBool(true), "true"},
		{"bool false", testctx.MakeBool(false), "false"},

		// Strings
		{"string empty", testctx.MakeString(""), ""},
		{"string simple", testctx.MakeString("hello"), "hello"},
		{"string with spaces", testctx.MakeString("hello world"), "hello world"},
		{"string with quotes", testctx.MakeString("hello \"world\""), "hello \"world\""},
		{"string with newlines", testctx.MakeString("line1\nline2"), "line1\nline2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := showImpl(ctx.EffContext, []eval.Value{tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, testctx.GetString(result))
		})
	}
}

func TestShow_FloatSpecialValues(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		value    float64
		expected string
	}{
		{"NaN", math.NaN(), "NaN"},
		{"positive infinity", math.Inf(1), "Inf"},
		{"negative infinity", math.Inf(-1), "-Inf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := showImpl(ctx.EffContext, []eval.Value{testctx.MakeFloat(tt.value)})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, testctx.GetString(result))
		})
	}
}

func TestShow_Lists(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		input    eval.Value
		expected string
	}{
		{
			"empty list",
			testctx.MakeList([]eval.Value{}),
			"[]",
		},
		{
			"single int",
			testctx.MakeList([]eval.Value{
				testctx.MakeInt(42),
			}),
			"[42]",
		},
		{
			"multiple ints",
			testctx.MakeList([]eval.Value{
				testctx.MakeInt(1),
				testctx.MakeInt(2),
				testctx.MakeInt(3),
			}),
			"[1, 2, 3]",
		},
		{
			"mixed types",
			testctx.MakeList([]eval.Value{
				testctx.MakeInt(42),
				testctx.MakeString("hello"),
				testctx.MakeBool(true),
			}),
			"[42, hello, true]",
		},
		{
			"nested lists",
			testctx.MakeList([]eval.Value{
				testctx.MakeList([]eval.Value{
					testctx.MakeInt(1),
					testctx.MakeInt(2),
				}),
				testctx.MakeList([]eval.Value{
					testctx.MakeInt(3),
					testctx.MakeInt(4),
				}),
			}),
			"[[1, 2], [3, 4]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := showImpl(ctx.EffContext, []eval.Value{tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, testctx.GetString(result))
		})
	}
}

func TestShow_Records(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		input    eval.Value
		expected string
	}{
		{
			"empty record",
			testctx.MakeRecord(map[string]eval.Value{}),
			"{}",
		},
		{
			"single field",
			testctx.MakeRecord(map[string]eval.Value{
				"x": testctx.MakeInt(42),
			}),
			"{x: 42}",
		},
		{
			"multiple fields (sorted)",
			testctx.MakeRecord(map[string]eval.Value{
				"name": testctx.MakeString("Alice"),
				"age":  testctx.MakeInt(30),
			}),
			"{age: 30, name: Alice}", // Sorted alphabetically
		},
		{
			"nested record",
			testctx.MakeRecord(map[string]eval.Value{
				"person": testctx.MakeRecord(map[string]eval.Value{
					"name": testctx.MakeString("Bob"),
					"age":  testctx.MakeInt(25),
				}),
			}),
			"{person: {age: 25, name: Bob}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := showImpl(ctx.EffContext, []eval.Value{tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, testctx.GetString(result))
		})
	}
}

func TestShow_Constructors(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		input    eval.Value
		expected string
	}{
		{
			"constructor without args",
			&eval.TaggedValue{
				CtorName: "None",
				Fields:   []eval.Value{},
			},
			"None",
		},
		{
			"constructor with single arg",
			&eval.TaggedValue{
				CtorName: "Some",
				Fields:   []eval.Value{testctx.MakeInt(42)},
			},
			"Some(42)",
		},
		{
			"constructor with multiple args",
			&eval.TaggedValue{
				CtorName: "Pair",
				Fields: []eval.Value{
					testctx.MakeInt(1),
					testctx.MakeString("hello"),
				},
			},
			"Pair(1, hello)",
		},
		{
			"nested constructors",
			&eval.TaggedValue{
				CtorName: "Some",
				Fields: []eval.Value{
					&eval.TaggedValue{
						CtorName: "Just",
						Fields:   []eval.Value{testctx.MakeInt(42)},
					},
				},
			},
			"Some(Just(42))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := showImpl(ctx.EffContext, []eval.Value{tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, testctx.GetString(result))
		})
	}
}

func TestShow_DepthLimit(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	// Create deeply nested list: [[[[42]]]]
	deepList := testctx.MakeInt(42)
	for i := 0; i < 5; i++ {
		deepList = testctx.MakeList([]eval.Value{deepList})
	}

	result, err := showImpl(ctx.EffContext, []eval.Value{deepList})
	require.NoError(t, err)

	// Should hit depth limit and show "..."
	assert.Contains(t, testctx.GetString(result), "...")
}

func TestShow_LongStrings(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	// Create a very long list
	elements := make([]eval.Value, 50)
	for i := 0; i < 50; i++ {
		elements[i] = testctx.MakeInt(i)
	}
	longList := testctx.MakeList(elements)

	result, err := showImpl(ctx.EffContext, []eval.Value{longList})
	require.NoError(t, err)

	output := testctx.GetString(result)

	// Should truncate with "..." in the middle
	assert.Contains(t, output, "...")
	assert.LessOrEqual(t, len(output), maxWidth+10) // Some tolerance
}

func TestShow_FunctionValue(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	funcVal := &eval.FunctionValue{
		Params: []string{"x"},
		Body:   nil, // Not important for show
		Env:    nil,
	}

	result, err := showImpl(ctx.EffContext, []eval.Value{funcVal})
	require.NoError(t, err)
	assert.Equal(t, "<function>", testctx.GetString(result))
}

func TestShow_ErrorValue(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	errVal := &eval.ErrorValue{
		Message: "something went wrong",
	}

	result, err := showImpl(ctx.EffContext, []eval.Value{errVal})
	require.NoError(t, err)
	assert.Equal(t, "Error: something went wrong", testctx.GetString(result))
}

func TestShow_TypeRegistration(t *testing.T) {
	// Verify show is registered in the builtin registry
	spec, exists := GetSpec("show")
	require.True(t, exists, "show should be registered")

	assert.Equal(t, "show", spec.Name)
	assert.Equal(t, "$builtin", spec.Module)
	assert.Equal(t, 1, spec.NumArgs)
	assert.True(t, spec.IsPure)
	assert.Equal(t, "", spec.Effect)
}
