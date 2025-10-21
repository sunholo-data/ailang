package eval

import "math"

// registerArithmeticBuiltins registers integer and float arithmetic operations
func registerArithmeticBuiltins() {
	// Integer operations
	Builtins["add_Int"] = &BuiltinFunc{
		Name:    "add_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*IntValue, error) {
			return &IntValue{Value: a.Value + b.Value}, nil
		},
	}

	Builtins["sub_Int"] = &BuiltinFunc{
		Name:    "sub_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*IntValue, error) {
			return &IntValue{Value: a.Value - b.Value}, nil
		},
	}

	Builtins["mul_Int"] = &BuiltinFunc{
		Name:    "mul_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*IntValue, error) {
			return &IntValue{Value: a.Value * b.Value}, nil
		},
	}

	Builtins["div_Int"] = &BuiltinFunc{
		Name:    "div_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*IntValue, error) {
			if b.Value == 0 {
				return nil, NewRuntimeError("RT_DIV0", "Division by zero", nil)
			}
			return &IntValue{Value: a.Value / b.Value}, nil
		},
	}

	Builtins["mod_Int"] = &BuiltinFunc{
		Name:    "mod_Int",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *IntValue) (*IntValue, error) {
			if b.Value == 0 {
				return nil, NewRuntimeError("RT_DIV0", "Modulo by zero", nil)
			}
			return &IntValue{Value: a.Value % b.Value}, nil
		},
	}

	Builtins["neg_Int"] = &BuiltinFunc{
		Name:    "neg_Int",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(a *IntValue) (*IntValue, error) {
			return &IntValue{Value: -a.Value}, nil
		},
	}

	// Float operations
	Builtins["add_Float"] = &BuiltinFunc{
		Name:    "add_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*FloatValue, error) {
			return &FloatValue{Value: a.Value + b.Value}, nil
		},
	}

	Builtins["sub_Float"] = &BuiltinFunc{
		Name:    "sub_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*FloatValue, error) {
			return &FloatValue{Value: a.Value - b.Value}, nil
		},
	}

	Builtins["mul_Float"] = &BuiltinFunc{
		Name:    "mul_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*FloatValue, error) {
			return &FloatValue{Value: a.Value * b.Value}, nil
		},
	}

	Builtins["div_Float"] = &BuiltinFunc{
		Name:    "div_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*FloatValue, error) {
			if b.Value == 0.0 {
				// IEEE 754 behavior: division by zero produces infinity
				if a.Value >= 0 {
					return &FloatValue{Value: math.Inf(1)}, nil
				} else {
					return &FloatValue{Value: math.Inf(-1)}, nil
				}
			}
			return &FloatValue{Value: a.Value / b.Value}, nil
		},
	}

	Builtins["mod_Float"] = &BuiltinFunc{
		Name:    "mod_Float",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *FloatValue) (*FloatValue, error) {
			if b.Value == 0.0 {
				return &FloatValue{Value: math.NaN()}, nil
			}
			return &FloatValue{Value: math.Mod(a.Value, b.Value)}, nil
		},
	}

	Builtins["neg_Float"] = &BuiltinFunc{
		Name:    "neg_Float",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(a *FloatValue) (*FloatValue, error) {
			return &FloatValue{Value: -a.Value}, nil
		},
	}
}
