package vm

import (
	"fmt"
	"math"

	"github.com/sunholo/ailang/internal/bytecode"
)

// M-BYTECODE-STDLIB-BUILTINS M2: Math + type conversion builtins wired to VM.

// --- Unary float→float math functions ----------------------------------------

func builtinMathSin(args []bytecode.Value) (bytecode.Value, error) {
	return unaryFloat("__math_sin", math.Sin, args)
}
func builtinMathCos(args []bytecode.Value) (bytecode.Value, error) {
	return unaryFloat("__math_cos", math.Cos, args)
}
func builtinMathTan(args []bytecode.Value) (bytecode.Value, error) {
	return unaryFloat("__math_tan", math.Tan, args)
}
func builtinMathAsin(args []bytecode.Value) (bytecode.Value, error) {
	return unaryFloat("__math_asin", math.Asin, args)
}
func builtinMathAcos(args []bytecode.Value) (bytecode.Value, error) {
	return unaryFloat("__math_acos", math.Acos, args)
}
func builtinMathAtan(args []bytecode.Value) (bytecode.Value, error) {
	return unaryFloat("__math_atan", math.Atan, args)
}
func builtinMathSqrt(args []bytecode.Value) (bytecode.Value, error) {
	return unaryFloat("__math_sqrt", math.Sqrt, args)
}
func builtinMathExp(args []bytecode.Value) (bytecode.Value, error) {
	return unaryFloat("__math_exp", math.Exp, args)
}
func builtinMathLog(args []bytecode.Value) (bytecode.Value, error) {
	return unaryFloat("__math_log", math.Log, args)
}
func builtinMathLog10(args []bytecode.Value) (bytecode.Value, error) {
	return unaryFloat("__math_log10", math.Log10, args)
}
func builtinMathFloor(args []bytecode.Value) (bytecode.Value, error) {
	return unaryFloat("__math_floor", math.Floor, args)
}
func builtinMathCeil(args []bytecode.Value) (bytecode.Value, error) {
	return unaryFloat("__math_ceil", math.Ceil, args)
}
func builtinMathRound(args []bytecode.Value) (bytecode.Value, error) {
	return unaryFloat("__math_round", math.Round, args)
}
func builtinMathAbsFloat(args []bytecode.Value) (bytecode.Value, error) {
	return unaryFloat("__math_abs_Float", math.Abs, args)
}

// --- Binary float→float math functions ---------------------------------------

func builtinMathAtan2(args []bytecode.Value) (bytecode.Value, error) {
	return binaryFloat("__math_atan2", math.Atan2, args)
}
func builtinMathPow(args []bytecode.Value) (bytecode.Value, error) {
	return binaryFloat("__math_pow", math.Pow, args)
}

// --- Integer math ------------------------------------------------------------

func builtinMathAbsInt(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__math_abs_Int: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagInt {
		return bytecode.Value{}, fmt.Errorf("__math_abs_Int: expected int")
	}
	v := args[0].Int
	if v < 0 {
		v = -v
	}
	return bytecode.NewInt(v), nil
}

func builtinModInt(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("_mod_Int: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagInt || args[1].Tag != bytecode.TagInt {
		return bytecode.Value{}, fmt.Errorf("_mod_Int: expected ints")
	}
	if args[1].Int == 0 {
		return bytecode.Value{}, fmt.Errorf("_mod_Int: division by zero")
	}
	return bytecode.NewInt(args[0].Int % args[1].Int), nil
}

// --- Constants (take unit arg) -----------------------------------------------

func builtinMathPI(args []bytecode.Value) (bytecode.Value, error) {
	// PI() takes a unit argument
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__math_PI: expected 1 arg, got %d", len(args))
	}
	return bytecode.NewFloat(math.Pi), nil
}

func builtinMathE(args []bytecode.Value) (bytecode.Value, error) {
	// E() takes a unit argument
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__math_E: expected 1 arg, got %d", len(args))
	}
	return bytecode.NewFloat(math.E), nil
}

// --- Type conversion ---------------------------------------------------------

func builtinFloatToInt(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("_floatToInt: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagFloat {
		return bytecode.Value{}, fmt.Errorf("_floatToInt: expected float")
	}
	return bytecode.NewInt(int64(args[0].Flt)), nil
}

// __float_to_int is the numeric.go alias for floatToInt
func builtinFloatToInt2(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__float_to_int: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagFloat {
		return bytecode.Value{}, fmt.Errorf("__float_to_int: expected float")
	}
	return bytecode.NewInt(int64(args[0].Flt)), nil
}

// __int_to_float is the numeric.go alias for intToFloat
func builtinIntToFloat2(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__int_to_float: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagInt {
		return bytecode.Value{}, fmt.Errorf("__int_to_float: expected int")
	}
	return bytecode.NewFloat(float64(args[0].Int)), nil
}

// --- helpers -----------------------------------------------------------------

func unaryFloat(name string, fn func(float64) float64, args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("%s: expected 1 arg, got %d", name, len(args))
	}
	if args[0].Tag != bytecode.TagFloat {
		return bytecode.Value{}, fmt.Errorf("%s: expected float", name)
	}
	return bytecode.NewFloat(fn(args[0].Flt)), nil
}

func binaryFloat(name string, fn func(float64, float64) float64, args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("%s: expected 2 args, got %d", name, len(args))
	}
	if args[0].Tag != bytecode.TagFloat || args[1].Tag != bytecode.TagFloat {
		return bytecode.Value{}, fmt.Errorf("%s: expected floats", name)
	}
	return bytecode.NewFloat(fn(args[0].Flt, args[1].Flt)), nil
}
