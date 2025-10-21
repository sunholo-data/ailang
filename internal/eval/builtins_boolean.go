package eval

// registerBooleanBuiltins registers boolean operations
func registerBooleanBuiltins() {
	Builtins["and_Bool"] = &BuiltinFunc{
		Name:    "and_Bool",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *BoolValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value && b.Value}, nil
		},
	}

	Builtins["or_Bool"] = &BuiltinFunc{
		Name:    "or_Bool",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *BoolValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value || b.Value}, nil
		},
	}

	Builtins["not_Bool"] = &BuiltinFunc{
		Name:    "not_Bool",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(a *BoolValue) (*BoolValue, error) {
			return &BoolValue{Value: !a.Value}, nil
		},
	}

	Builtins["eq_Bool"] = &BuiltinFunc{
		Name:    "eq_Bool",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *BoolValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value == b.Value}, nil
		},
	}

	Builtins["ne_Bool"] = &BuiltinFunc{
		Name:    "ne_Bool",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *BoolValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value != b.Value}, nil
		},
	}
}
