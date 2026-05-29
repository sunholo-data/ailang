package builtins

// ============================================================================
// std/math builtins — Go codegen specs
// ============================================================================

func registerMathCodegenSpecs() {
	mathFuncs := map[string]struct{ goExpr, stdlibName string }{
		"_math_sin":       {"math.Sin({{arg0}}.(float64))", "sin"},
		"_math_cos":       {"math.Cos({{arg0}}.(float64))", "cos"},
		"_math_tan":       {"math.Tan({{arg0}}.(float64))", "tan"},
		"_math_asin":      {"math.Asin({{arg0}}.(float64))", "asin"},
		"_math_acos":      {"math.Acos({{arg0}}.(float64))", "acos"},
		"_math_atan":      {"math.Atan({{arg0}}.(float64))", "atan"},
		"_math_atan2":     {"math.Atan2({{arg0}}.(float64), {{arg1}}.(float64))", "atan2"},
		"_math_exp":       {"math.Exp({{arg0}}.(float64))", "exp"},
		"_math_log":       {"math.Log({{arg0}}.(float64))", "log"},
		"_math_log10":     {"math.Log10({{arg0}}.(float64))", "log10"},
		"_math_pow":       {"math.Pow({{arg0}}.(float64), {{arg1}}.(float64))", "pow"},
		"_math_sqrt":      {"math.Sqrt({{arg0}}.(float64))", "sqrt"},
		"_math_ceil":      {"math.Ceil({{arg0}}.(float64))", "ceil"},
		"_math_floor":     {"math.Floor({{arg0}}.(float64))", "floor"},
		"_math_round":     {"math.Round({{arg0}}.(float64))", "round"},
		"_math_abs_Float": {"math.Abs({{arg0}}.(float64))", "absFloat"},
		"_math_abs_Int":   {"int64(math.Abs(float64(toInt64({{arg0}}))))", "absInt"},
	}
	for name, spec := range mathFuncs {
		numArgs := 1
		if name == "_math_atan2" || name == "_math_pow" {
			numArgs = 2
		}
		registerIfMissing(name, numArgs, true, &GoCodegenSpec{
			Inline:     spec.goExpr,
			Imports:    []string{"math"},
			StdlibName: spec.stdlibName,
		})
	}
	// Math constants
	registerIfMissing("_math_PI", 0, true, &GoCodegenSpec{
		Inline:     `math.Pi`,
		Imports:    []string{"math"},
		StdlibName: "PI",
	})
	registerIfMissing("_math_E", 0, true, &GoCodegenSpec{
		Inline:     `math.E`,
		Imports:    []string{"math"},
		StdlibName: "E",
	})
	// Conversion builtins used by math
	registerIfMissing("_int_to_float", 1, true, &GoCodegenSpec{
		Inline:     `float64(toInt64({{arg0}}))`,
		StdlibName: "intToFloat",
	})
	registerIfMissing("_float_to_int", 1, true, &GoCodegenSpec{
		Inline:     `int64({{arg0}}.(float64))`,
		StdlibName: "floatToInt",
	})
}

// ============================================================================
// Conversion builtins — Go codegen specs
// ============================================================================

func registerConversionCodegenSpecs() {
	setSpec("intToFloat", &GoCodegenSpec{
		Inline:     `float64(toInt64({{arg0}}))`,
		StdlibName: "intToFloat",
	})
	setSpec("floatToInt", &GoCodegenSpec{
		Inline:     `int64({{arg0}}.(float64))`,
		StdlibName: "floatToInt",
	})
}
