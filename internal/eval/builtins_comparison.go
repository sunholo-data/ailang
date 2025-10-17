package eval

import "math"

// registerComparisonBuiltins registers comparison operations
func registerComparisonBuiltins() {
	// Integer comparisons
	Builtins["eq_Int"] = &BuiltinFunc{
		Name:    "eq_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value == b.Value}, nil
		},
	}

	Builtins["ne_Int"] = &BuiltinFunc{
		Name:    "ne_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value != b.Value}, nil
		},
	}

	Builtins["lt_Int"] = &BuiltinFunc{
		Name:    "lt_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value < b.Value}, nil
		},
	}

	Builtins["le_Int"] = &BuiltinFunc{
		Name:    "le_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value <= b.Value}, nil
		},
	}

	Builtins["gt_Int"] = &BuiltinFunc{
		Name:    "gt_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value > b.Value}, nil
		},
	}

	Builtins["ge_Int"] = &BuiltinFunc{
		Name:    "ge_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value >= b.Value}, nil
		},
	}

	// Float comparisons with NaN handling
	// NaN comparisons: all comparisons with NaN return false except !=
	Builtins["eq_Float"] = &BuiltinFunc{
		Name:    "eq_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*BoolValue, error) {
			// NaN is not equal to anything, including itself
			if math.IsNaN(a.Value) || math.IsNaN(b.Value) {
				return &BoolValue{Value: false}, nil
			}
			return &BoolValue{Value: a.Value == b.Value}, nil
		},
	}

	Builtins["ne_Float"] = &BuiltinFunc{
		Name:    "ne_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*BoolValue, error) {
			// NaN is not equal to anything, so != returns true for any NaN
			if math.IsNaN(a.Value) || math.IsNaN(b.Value) {
				return &BoolValue{Value: true}, nil
			}
			return &BoolValue{Value: a.Value != b.Value}, nil
		},
	}

	Builtins["lt_Float"] = &BuiltinFunc{
		Name:    "lt_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*BoolValue, error) {
			// Any comparison with NaN returns false
			if math.IsNaN(a.Value) || math.IsNaN(b.Value) {
				return &BoolValue{Value: false}, nil
			}
			return &BoolValue{Value: a.Value < b.Value}, nil
		},
	}

	Builtins["le_Float"] = &BuiltinFunc{
		Name:    "le_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*BoolValue, error) {
			if math.IsNaN(a.Value) || math.IsNaN(b.Value) {
				return &BoolValue{Value: false}, nil
			}
			return &BoolValue{Value: a.Value <= b.Value}, nil
		},
	}

	Builtins["gt_Float"] = &BuiltinFunc{
		Name:    "gt_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*BoolValue, error) {
			if math.IsNaN(a.Value) || math.IsNaN(b.Value) {
				return &BoolValue{Value: false}, nil
			}
			return &BoolValue{Value: a.Value > b.Value}, nil
		},
	}

	Builtins["ge_Float"] = &BuiltinFunc{
		Name:    "ge_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*BoolValue, error) {
			if math.IsNaN(a.Value) || math.IsNaN(b.Value) {
				return &BoolValue{Value: false}, nil
			}
			return &BoolValue{Value: a.Value >= b.Value}, nil
		},
	}
}
