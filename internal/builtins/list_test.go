package builtins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sunholo/ailang/internal/effects/testctx"
	"github.com/sunholo/ailang/internal/eval"
)

// TestListConcat tests the concat_List builtin implementation
func TestListConcat(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		list1    []eval.Value
		list2    []eval.Value
		expected []eval.Value
	}{
		{
			name:     "concat two non-empty lists",
			list1:    []eval.Value{testctx.MakeInt(1), testctx.MakeInt(2), testctx.MakeInt(3)},
			list2:    []eval.Value{testctx.MakeInt(4), testctx.MakeInt(5)},
			expected: []eval.Value{testctx.MakeInt(1), testctx.MakeInt(2), testctx.MakeInt(3), testctx.MakeInt(4), testctx.MakeInt(5)},
		},
		{
			name:     "concat with empty first list",
			list1:    []eval.Value{},
			list2:    []eval.Value{testctx.MakeInt(1), testctx.MakeInt(2)},
			expected: []eval.Value{testctx.MakeInt(1), testctx.MakeInt(2)},
		},
		{
			name:     "concat with empty second list",
			list1:    []eval.Value{testctx.MakeInt(1), testctx.MakeInt(2)},
			list2:    []eval.Value{},
			expected: []eval.Value{testctx.MakeInt(1), testctx.MakeInt(2)},
		},
		{
			name:     "concat two empty lists",
			list1:    []eval.Value{},
			list2:    []eval.Value{},
			expected: []eval.Value{},
		},
		{
			name: "concat lists of strings",
			list1: []eval.Value{
				testctx.MakeString("hello"),
				testctx.MakeString("world"),
			},
			list2: []eval.Value{
				testctx.MakeString("foo"),
				testctx.MakeString("bar"),
			},
			expected: []eval.Value{
				testctx.MakeString("hello"),
				testctx.MakeString("world"),
				testctx.MakeString("foo"),
				testctx.MakeString("bar"),
			},
		},
		{
			name: "concat nested lists",
			list1: []eval.Value{
				&eval.ListValue{Elements: []eval.Value{testctx.MakeInt(1)}},
				&eval.ListValue{Elements: []eval.Value{testctx.MakeInt(2)}},
			},
			list2: []eval.Value{
				&eval.ListValue{Elements: []eval.Value{testctx.MakeInt(3)}},
			},
			expected: []eval.Value{
				&eval.ListValue{Elements: []eval.Value{testctx.MakeInt(1)}},
				&eval.ListValue{Elements: []eval.Value{testctx.MakeInt(2)}},
				&eval.ListValue{Elements: []eval.Value{testctx.MakeInt(3)}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the builtin implementation
			result, err := listConcatImpl(ctx.EffContext, []eval.Value{
				&eval.ListValue{Elements: tt.list1},
				&eval.ListValue{Elements: tt.list2},
			})

			// Check no error
			assert.NoError(t, err)

			// Extract result list
			resultList := testctx.GetList(result)

			// Check length
			assert.Equal(t, len(tt.expected), len(resultList), "result length mismatch")

			// Check each element
			for i, expectedVal := range tt.expected {
				assert.Equal(t, expectedVal, resultList[i], "element %d mismatch", i)
			}
		})
	}
}

// TestListConcatTypeMismatch tests error handling for wrong types
func TestListConcatTypeMismatch(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name string
		arg1 eval.Value
		arg2 eval.Value
	}{
		{
			name: "first arg is string instead of list",
			arg1: testctx.MakeString("not a list"),
			arg2: &eval.ListValue{Elements: []eval.Value{testctx.MakeInt(1)}},
		},
		{
			name: "second arg is int instead of list",
			arg1: &eval.ListValue{Elements: []eval.Value{testctx.MakeInt(1)}},
			arg2: testctx.MakeInt(42),
		},
		{
			name: "both args are wrong type",
			arg1: testctx.MakeString("not a list"),
			arg2: testctx.MakeInt(42),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := listConcatImpl(ctx.EffContext, []eval.Value{tt.arg1, tt.arg2})
			assert.Error(t, err, "expected error for type mismatch")
		})
	}
}

// TestListConcatRegistration verifies the builtin is properly registered
func TestListConcatRegistration(t *testing.T) {
	// Check the builtin is registered
	spec, ok := GetSpec("concat_List")
	assert.True(t, ok, "concat_List should be registered")
	assert.NotNil(t, spec)

	// Verify metadata
	assert.Equal(t, "std/list", spec.Module)
	assert.Equal(t, "concat_List", spec.Name)
	assert.Equal(t, 2, spec.NumArgs)
	assert.True(t, spec.IsPure)
	assert.Equal(t, "", spec.Effect)

	// Verify metadata is present
	assert.NotNil(t, spec.Metadata)
	assert.NotEmpty(t, spec.Metadata.Description)
	assert.Len(t, spec.Metadata.Params, 2)
	assert.NotEmpty(t, spec.Metadata.Returns)
	assert.NotEmpty(t, spec.Metadata.Examples)
}

// TestListConcatProperties tests mathematical properties of list concatenation
func TestListConcatProperties(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	t.Run("length property: len(xs ++ ys) == len(xs) + len(ys)", func(t *testing.T) {
		xs := []eval.Value{testctx.MakeInt(1), testctx.MakeInt(2), testctx.MakeInt(3)}
		ys := []eval.Value{testctx.MakeInt(4), testctx.MakeInt(5)}

		result, err := listConcatImpl(ctx.EffContext, []eval.Value{
			&eval.ListValue{Elements: xs},
			&eval.ListValue{Elements: ys},
		})

		assert.NoError(t, err)
		resultList := testctx.GetList(result)
		assert.Equal(t, len(xs)+len(ys), len(resultList))
	})

	t.Run("left identity: [] ++ xs == xs", func(t *testing.T) {
		xs := []eval.Value{testctx.MakeInt(1), testctx.MakeInt(2)}

		result, err := listConcatImpl(ctx.EffContext, []eval.Value{
			&eval.ListValue{Elements: []eval.Value{}},
			&eval.ListValue{Elements: xs},
		})

		assert.NoError(t, err)
		resultList := testctx.GetList(result)
		assert.Equal(t, xs, resultList)
	})

	t.Run("right identity: xs ++ [] == xs", func(t *testing.T) {
		xs := []eval.Value{testctx.MakeInt(1), testctx.MakeInt(2)}

		result, err := listConcatImpl(ctx.EffContext, []eval.Value{
			&eval.ListValue{Elements: xs},
			&eval.ListValue{Elements: []eval.Value{}},
		})

		assert.NoError(t, err)
		resultList := testctx.GetList(result)
		assert.Equal(t, xs, resultList)
	})

	t.Run("associativity: (xs ++ ys) ++ zs == xs ++ (ys ++ zs)", func(t *testing.T) {
		xs := []eval.Value{testctx.MakeInt(1)}
		ys := []eval.Value{testctx.MakeInt(2)}
		zs := []eval.Value{testctx.MakeInt(3)}

		// (xs ++ ys) ++ zs
		temp1, _ := listConcatImpl(ctx.EffContext, []eval.Value{
			&eval.ListValue{Elements: xs},
			&eval.ListValue{Elements: ys},
		})
		left, _ := listConcatImpl(ctx.EffContext, []eval.Value{
			temp1,
			&eval.ListValue{Elements: zs},
		})

		// xs ++ (ys ++ zs)
		temp2, _ := listConcatImpl(ctx.EffContext, []eval.Value{
			&eval.ListValue{Elements: ys},
			&eval.ListValue{Elements: zs},
		})
		right, _ := listConcatImpl(ctx.EffContext, []eval.Value{
			&eval.ListValue{Elements: xs},
			temp2,
		})

		assert.Equal(t, testctx.GetList(left), testctx.GetList(right))
	})
}

// TestListConcatEdgeCases tests boundary conditions
func TestListConcatEdgeCases(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	t.Run("both empty lists", func(t *testing.T) {
		result, err := listConcatImpl(ctx.EffContext, []eval.Value{
			&eval.ListValue{Elements: []eval.Value{}},
			&eval.ListValue{Elements: []eval.Value{}},
		})

		assert.NoError(t, err)
		resultList := testctx.GetList(result)
		assert.Equal(t, 0, len(resultList))
	})

	t.Run("large lists", func(t *testing.T) {
		// Create two lists with 1000 elements each
		largeList1 := make([]eval.Value, 1000)
		largeList2 := make([]eval.Value, 1000)
		for i := 0; i < 1000; i++ {
			largeList1[i] = testctx.MakeInt(i)
			largeList2[i] = testctx.MakeInt(i + 1000)
		}

		result, err := listConcatImpl(ctx.EffContext, []eval.Value{
			&eval.ListValue{Elements: largeList1},
			&eval.ListValue{Elements: largeList2},
		})

		assert.NoError(t, err)
		resultList := testctx.GetList(result)
		assert.Equal(t, 2000, len(resultList))
		// Verify first and last elements
		assert.Equal(t, 0, testctx.GetInt(resultList[0]))
		assert.Equal(t, 1999, testctx.GetInt(resultList[1999]))
	})

	t.Run("deeply nested lists", func(t *testing.T) {
		// [[1]], [[2]], [[3]]
		nested1 := &eval.ListValue{Elements: []eval.Value{
			&eval.ListValue{Elements: []eval.Value{testctx.MakeInt(1)}},
		}}
		nested2 := &eval.ListValue{Elements: []eval.Value{
			&eval.ListValue{Elements: []eval.Value{testctx.MakeInt(2)}},
		}}

		result, err := listConcatImpl(ctx.EffContext, []eval.Value{nested1, nested2})

		assert.NoError(t, err)
		resultList := testctx.GetList(result)
		assert.Equal(t, 2, len(resultList))
	})
}

// TestListConcatImmutability verifies that concat does not mutate inputs
func TestListConcatImmutability(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	xs := []eval.Value{testctx.MakeInt(1), testctx.MakeInt(2)}
	ys := []eval.Value{testctx.MakeInt(3), testctx.MakeInt(4)}

	xsList := &eval.ListValue{Elements: xs}
	ysList := &eval.ListValue{Elements: ys}

	// Save original lengths
	xsLen := len(xs)
	ysLen := len(ys)

	// Perform concatenation
	_, err := listConcatImpl(ctx.EffContext, []eval.Value{xsList, ysList})
	assert.NoError(t, err)

	// Verify inputs were not mutated
	assert.Equal(t, xsLen, len(xsList.Elements), "first list should not be mutated")
	assert.Equal(t, ysLen, len(ysList.Elements), "second list should not be mutated")
}
