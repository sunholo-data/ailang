package builtins

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/eval"
)

// ============================================================================
// Trigonometric Functions Tests
// ============================================================================

func TestSinRegistered(t *testing.T) {
	spec, ok := GetSpec("sin")
	require.True(t, ok, "sin should be registered")

	assert.Equal(t, "std/math", spec.Module)
	assert.Equal(t, "sin", spec.Name)
	assert.Equal(t, 1, spec.NumArgs)
	assert.True(t, spec.IsPure)
}

func TestSinImpl(t *testing.T) {
	spec, ok := GetSpec("sin")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    float64
		expected float64
		tol      float64
	}{
		{"zero", 0.0, 0.0, 1e-10},
		{"pi/2", math.Pi / 2, 1.0, 1e-10},
		{"pi", math.Pi, 0.0, 1e-10},
		{"-pi/2", -math.Pi / 2, -1.0, 1e-10},
		{"pi/6", math.Pi / 6, 0.5, 1e-10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.FloatValue{Value: tt.input}}
			result, err := spec.Impl(nil, args)

			require.NoError(t, err)
			floatVal, ok := result.(*eval.FloatValue)
			require.True(t, ok, "result should be FloatValue")
			assert.InDelta(t, tt.expected, floatVal.Value, tt.tol)
		})
	}
}

func TestCosRegistered(t *testing.T) {
	spec, ok := GetSpec("cos")
	require.True(t, ok, "cos should be registered")

	assert.Equal(t, "std/math", spec.Module)
	assert.Equal(t, 1, spec.NumArgs)
	assert.True(t, spec.IsPure)
}

func TestCosImpl(t *testing.T) {
	spec, ok := GetSpec("cos")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    float64
		expected float64
		tol      float64
	}{
		{"zero", 0.0, 1.0, 1e-10},
		{"pi/2", math.Pi / 2, 0.0, 1e-10},
		{"pi", math.Pi, -1.0, 1e-10},
		{"pi/3", math.Pi / 3, 0.5, 1e-10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.FloatValue{Value: tt.input}}
			result, err := spec.Impl(nil, args)

			require.NoError(t, err)
			floatVal, ok := result.(*eval.FloatValue)
			require.True(t, ok)
			assert.InDelta(t, tt.expected, floatVal.Value, tt.tol)
		})
	}
}

func TestTanImpl(t *testing.T) {
	spec, ok := GetSpec("tan")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    float64
		expected float64
		tol      float64
	}{
		{"zero", 0.0, 0.0, 1e-10},
		{"pi/4", math.Pi / 4, 1.0, 1e-10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.FloatValue{Value: tt.input}}
			result, err := spec.Impl(nil, args)

			require.NoError(t, err)
			floatVal, ok := result.(*eval.FloatValue)
			require.True(t, ok)
			assert.InDelta(t, tt.expected, floatVal.Value, tt.tol)
		})
	}
}

func TestSqrtImpl(t *testing.T) {
	spec, ok := GetSpec("sqrt")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    float64
		expected float64
		tol      float64
	}{
		{"zero", 0.0, 0.0, 1e-10},
		{"one", 1.0, 1.0, 1e-10},
		{"four", 4.0, 2.0, 1e-10},
		{"two", 2.0, math.Sqrt2, 1e-10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.FloatValue{Value: tt.input}}
			result, err := spec.Impl(nil, args)

			require.NoError(t, err)
			floatVal, ok := result.(*eval.FloatValue)
			require.True(t, ok)
			assert.InDelta(t, tt.expected, floatVal.Value, tt.tol)
		})
	}
}

func TestSqrtNegative(t *testing.T) {
	spec, ok := GetSpec("sqrt")
	require.True(t, ok)

	args := []eval.Value{&eval.FloatValue{Value: -1.0}}
	result, err := spec.Impl(nil, args)

	require.NoError(t, err)
	floatVal, ok := result.(*eval.FloatValue)
	require.True(t, ok)
	assert.True(t, math.IsNaN(floatVal.Value), "sqrt(-1) should be NaN")
}

func TestAtan2Impl(t *testing.T) {
	spec, ok := GetSpec("atan2")
	require.True(t, ok)
	assert.Equal(t, 2, spec.NumArgs)

	tests := []struct {
		name     string
		y, x     float64
		expected float64
		tol      float64
	}{
		{"origin_x", 0.0, 1.0, 0.0, 1e-10},
		{"origin_y", 1.0, 0.0, math.Pi / 2, 1e-10},
		{"45deg", 1.0, 1.0, math.Pi / 4, 1e-10},
		{"neg_x", 0.0, -1.0, math.Pi, 1e-10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{
				&eval.FloatValue{Value: tt.y},
				&eval.FloatValue{Value: tt.x},
			}
			result, err := spec.Impl(nil, args)

			require.NoError(t, err)
			floatVal, ok := result.(*eval.FloatValue)
			require.True(t, ok)
			assert.InDelta(t, tt.expected, floatVal.Value, tt.tol)
		})
	}
}

func TestAbsFloatImpl(t *testing.T) {
	spec, ok := GetSpec("abs_Float")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"positive", 3.14, 3.14},
		{"negative", -3.14, 3.14},
		{"zero", 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.FloatValue{Value: tt.input}}
			result, err := spec.Impl(nil, args)

			require.NoError(t, err)
			floatVal, ok := result.(*eval.FloatValue)
			require.True(t, ok)
			assert.Equal(t, tt.expected, floatVal.Value)
		})
	}
}

func TestAbsIntImpl(t *testing.T) {
	spec, ok := GetSpec("abs_Int")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"positive", 42, 42},
		{"negative", -42, 42},
		{"zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.IntValue{Value: tt.input}}
			result, err := spec.Impl(nil, args)

			require.NoError(t, err)
			intVal, ok := result.(*eval.IntValue)
			require.True(t, ok)
			assert.Equal(t, tt.expected, intVal.Value)
		})
	}
}

// ============================================================================
// Constants Tests
// ============================================================================

func TestPIRegistered(t *testing.T) {
	spec, ok := GetSpec("PI")
	require.True(t, ok, "PI should be registered")

	assert.Equal(t, "std/math", spec.Module)
	assert.Equal(t, 0, spec.NumArgs) // Zero-arg function
	assert.True(t, spec.IsPure)
}

func TestPIImpl(t *testing.T) {
	spec, ok := GetSpec("PI")
	require.True(t, ok)

	result, err := spec.Impl(nil, []eval.Value{})

	require.NoError(t, err)
	floatVal, ok := result.(*eval.FloatValue)
	require.True(t, ok)
	assert.Equal(t, math.Pi, floatVal.Value)
}

func TestERegistered(t *testing.T) {
	spec, ok := GetSpec("E")
	require.True(t, ok, "E should be registered")

	assert.Equal(t, "std/math", spec.Module)
	assert.Equal(t, 0, spec.NumArgs)
	assert.True(t, spec.IsPure)
}

func TestEImpl(t *testing.T) {
	spec, ok := GetSpec("E")
	require.True(t, ok)

	result, err := spec.Impl(nil, []eval.Value{})

	require.NoError(t, err)
	floatVal, ok := result.(*eval.FloatValue)
	require.True(t, ok)
	assert.Equal(t, math.E, floatVal.Value)
}

// ============================================================================
// Advanced Math Functions Tests
// ============================================================================

func TestPowImpl(t *testing.T) {
	spec, ok := GetSpec("pow")
	require.True(t, ok)

	tests := []struct {
		name     string
		x, y     float64
		expected float64
		tol      float64
	}{
		{"2^3", 2.0, 3.0, 8.0, 1e-10},
		{"4^0.5", 4.0, 0.5, 2.0, 1e-10},
		{"any^0", 5.0, 0.0, 1.0, 1e-10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{
				&eval.FloatValue{Value: tt.x},
				&eval.FloatValue{Value: tt.y},
			}
			result, err := spec.Impl(nil, args)

			require.NoError(t, err)
			floatVal, ok := result.(*eval.FloatValue)
			require.True(t, ok)
			assert.InDelta(t, tt.expected, floatVal.Value, tt.tol)
		})
	}
}

func TestExpImpl(t *testing.T) {
	spec, ok := GetSpec("exp")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    float64
		expected float64
		tol      float64
	}{
		{"zero", 0.0, 1.0, 1e-10},
		{"one", 1.0, math.E, 1e-10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.FloatValue{Value: tt.input}}
			result, err := spec.Impl(nil, args)

			require.NoError(t, err)
			floatVal, ok := result.(*eval.FloatValue)
			require.True(t, ok)
			assert.InDelta(t, tt.expected, floatVal.Value, tt.tol)
		})
	}
}

func TestLogImpl(t *testing.T) {
	spec, ok := GetSpec("log")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    float64
		expected float64
		tol      float64
	}{
		{"one", 1.0, 0.0, 1e-10},
		{"e", math.E, 1.0, 1e-10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.FloatValue{Value: tt.input}}
			result, err := spec.Impl(nil, args)

			require.NoError(t, err)
			floatVal, ok := result.(*eval.FloatValue)
			require.True(t, ok)
			assert.InDelta(t, tt.expected, floatVal.Value, tt.tol)
		})
	}
}

func TestFloorImpl(t *testing.T) {
	spec, ok := GetSpec("floor")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"positive", 3.7, 3.0},
		{"negative", -2.3, -3.0},
		{"integer", 5.0, 5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.FloatValue{Value: tt.input}}
			result, err := spec.Impl(nil, args)

			require.NoError(t, err)
			floatVal, ok := result.(*eval.FloatValue)
			require.True(t, ok)
			assert.Equal(t, tt.expected, floatVal.Value)
		})
	}
}

func TestCeilImpl(t *testing.T) {
	spec, ok := GetSpec("ceil")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"positive", 3.2, 4.0},
		{"negative", -2.7, -2.0},
		{"integer", 5.0, 5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.FloatValue{Value: tt.input}}
			result, err := spec.Impl(nil, args)

			require.NoError(t, err)
			floatVal, ok := result.(*eval.FloatValue)
			require.True(t, ok)
			assert.Equal(t, tt.expected, floatVal.Value)
		})
	}
}

func TestRoundImpl(t *testing.T) {
	spec, ok := GetSpec("round")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"half_up", 3.5, 4.0},
		{"half_down", -2.5, -3.0},
		{"round_down", 3.4, 3.0},
		{"round_up", 3.6, 4.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.FloatValue{Value: tt.input}}
			result, err := spec.Impl(nil, args)

			require.NoError(t, err)
			floatVal, ok := result.(*eval.FloatValue)
			require.True(t, ok)
			assert.Equal(t, tt.expected, floatVal.Value)
		})
	}
}

// ============================================================================
// Inverse Trig Functions Tests
// ============================================================================

func TestAsinImpl(t *testing.T) {
	spec, ok := GetSpec("asin")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    float64
		expected float64
		tol      float64
	}{
		{"zero", 0.0, 0.0, 1e-10},
		{"one", 1.0, math.Pi / 2, 1e-10},
		{"half", 0.5, math.Pi / 6, 1e-10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.FloatValue{Value: tt.input}}
			result, err := spec.Impl(nil, args)

			require.NoError(t, err)
			floatVal, ok := result.(*eval.FloatValue)
			require.True(t, ok)
			assert.InDelta(t, tt.expected, floatVal.Value, tt.tol)
		})
	}
}

func TestAcosImpl(t *testing.T) {
	spec, ok := GetSpec("acos")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    float64
		expected float64
		tol      float64
	}{
		{"one", 1.0, 0.0, 1e-10},
		{"zero", 0.0, math.Pi / 2, 1e-10},
		{"half", 0.5, math.Pi / 3, 1e-10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.FloatValue{Value: tt.input}}
			result, err := spec.Impl(nil, args)

			require.NoError(t, err)
			floatVal, ok := result.(*eval.FloatValue)
			require.True(t, ok)
			assert.InDelta(t, tt.expected, floatVal.Value, tt.tol)
		})
	}
}

func TestAtanImpl(t *testing.T) {
	spec, ok := GetSpec("atan")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    float64
		expected float64
		tol      float64
	}{
		{"zero", 0.0, 0.0, 1e-10},
		{"one", 1.0, math.Pi / 4, 1e-10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.FloatValue{Value: tt.input}}
			result, err := spec.Impl(nil, args)

			require.NoError(t, err)
			floatVal, ok := result.(*eval.FloatValue)
			require.True(t, ok)
			assert.InDelta(t, tt.expected, floatVal.Value, tt.tol)
		})
	}
}
