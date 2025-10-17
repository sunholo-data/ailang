package eval

// registerConversionBuiltins registers numeric type conversion functions
func registerConversionBuiltins() {
	Builtins["intToFloat"] = &BuiltinFunc{
		Name:    "intToFloat",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(a *IntValue) (*FloatValue, error) {
			return &FloatValue{Value: float64(a.Value)}, nil
		},
	}

	Builtins["floatToInt"] = &BuiltinFunc{
		Name:    "floatToInt",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(a *FloatValue) (*IntValue, error) {
			// Truncate towards zero (standard Go conversion)
			return &IntValue{Value: int(a.Value)}, nil
		},
	}
}
