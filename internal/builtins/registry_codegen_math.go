package builtins

import "strings"

// ============================================================================
// std/math builtins — Go codegen specs
// ============================================================================

func registerMathCodegenSpecs() {
	mathFuncs := map[string]struct{ goExpr, stdlibName string }{
		"_math_sin":       {"ailmathx_Sin({{arg0}}.(float64))", "sin"},
		"_math_cos":       {"ailmathx_Cos({{arg0}}.(float64))", "cos"},
		"_math_tan":       {"ailmathx_Tan({{arg0}}.(float64))", "tan"},
		"_math_asin":      {"ailmathx_Asin({{arg0}}.(float64))", "asin"},
		"_math_acos":      {"ailmathx_Acos({{arg0}}.(float64))", "acos"},
		"_math_atan":      {"ailmathx_Atan({{arg0}}.(float64))", "atan"},
		"_math_atan2":     {"ailmathx_Atan2({{arg0}}.(float64), {{arg1}}.(float64))", "atan2"},
		"_math_exp":       {"ailmathx_Exp({{arg0}}.(float64))", "exp"},
		"_math_log":       {"ailmathx_Log({{arg0}}.(float64))", "log"},
		"_math_log10":     {"ailmathx_Log10({{arg0}}.(float64))", "log10"},
		"_math_pow":       {"ailmathx_Pow({{arg0}}.(float64), {{arg1}}.(float64))", "pow"},
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
		// #1465: transcendentals call the portable ailmathx_* helpers the
		// generator emits from internal/mathx; exact ops stay on host math.
		imports := []string{"math"}
		if strings.HasPrefix(spec.goExpr, "ailmathx_") {
			imports = []string{"mathx"}
		}
		registerIfMissing(name, numArgs, true, &GoCodegenSpec{
			Inline:       spec.goExpr,
			Imports:      imports,
			StdlibName:   spec.stdlibName,
			StdlibModule: "std/math",
		})
	}
	// Math constants
	registerIfMissing("_math_PI", 0, true, &GoCodegenSpec{
		Inline:       `math.Pi`,
		Imports:      []string{"math"},
		StdlibName:   "PI",
		StdlibModule: "std/math",
	})
	registerIfMissing("_math_E", 0, true, &GoCodegenSpec{
		Inline:       `math.E`,
		Imports:      []string{"math"},
		StdlibName:   "E",
		StdlibModule: "std/math",
	})
	// #1481: logical right shift. A negative count panics ("negative shift
	// amount"), as the native >> does in generated Go.
	registerIfMissing("shiftRightLogical_Int", 2, true, &GoCodegenSpec{
		Inline:       `int64(uint64(toInt64({{arg0}})) >> toInt64({{arg1}}))`,
		StdlibName:   "shiftRightLogical",
		StdlibModule: "std/math",
	})
	// Conversion builtins used by math
	registerIfMissing("_int_to_float", 1, true, &GoCodegenSpec{
		Inline:       `float64(toInt64({{arg0}}))`,
		StdlibName:   "intToFloat",
		StdlibModule: "std/math",
	})
	registerIfMissing("_float_to_int", 1, true, &GoCodegenSpec{
		Inline:       `int64({{arg0}}.(float64))`,
		StdlibName:   "floatToInt",
		StdlibModule: "std/math",
	})
}

// ============================================================================
// Conversion builtins — Go codegen specs
// ============================================================================

func registerConversionCodegenSpecs() {
	setSpec("intToFloat", &GoCodegenSpec{
		Inline: `float64(toInt64({{arg0}}))`,
	})
	setSpec("floatToInt", &GoCodegenSpec{
		Inline: `int64({{arg0}}.(float64))`,
	})
}
