package builtins

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// This file contains builtin registrations using the new spec-based registry.
// Builtins are migrated here from their old scattered locations.

func init() {
	// Register string primitive builtins
	registerStringLen()
	registerStringCompare()
	registerStringEq()
	registerStringFind()
	registerStringSlice()
	registerStringTrim()
	registerStringUpper()
	registerStringLower()

	// Register arithmetic builtins
	registerArithmetic()

	// Register comparison builtins
	registerComparisons()

	// Register logic builtins
	registerLogic()

	// Register conversion builtins
	registerConversions()

	// Register string operations
	registerStringConcat()

	// Register IO effect builtins
	registerIO()

	// Register JSON builtins
	registerJSON()

	// Register Net effect builtins
	registerNetHTTPRequest()
}

// registerStringLen registers the _str_len builtin
// Old location: internal/eval/builtins.go
func registerStringLen() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_len",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "", // Pure function
		Type:    makeStrLenType,
		Impl:    strLenImpl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_len: %v", err))
	}
}

// makeStrLenType builds the type signature for _str_len
// Type: (String) -> Int
func makeStrLenType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.Int()).Build()
}

// strLenImpl is the implementation for _str_len
// UTF-8 aware string length (returns number of runes, not bytes)
func strLenImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Extract string argument
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_str_len: expected String, got %T", args[0])
	}

	// Count UTF-8 runes
	count := utf8.RuneCountInString(strVal.Value)

	return &eval.IntValue{Value: count}, nil
}

// registerNetHTTPRequest registers the _net_httpRequest builtin
// Old location: internal/effects/net.go
func registerNetHTTPRequest() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/net",
		Name:    "_net_httpRequest",
		NumArgs: 4,
		IsPure:  false,
		Effect:  "Net",
		Type:    makeHTTPRequestType,
		Impl:    effects.NetHTTPRequest,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _net_httpRequest: %v", err))
	}
}

// makeHTTPRequestType builds the type signature for _net_httpRequest
// Type: (String, String, List<{name: String, value: String}>, String)
//
//	-> Result<{status: Int, headers: List<{name: String, value: String}>, body: String, ok: Bool}, NetError>
//	! {Net}
func makeHTTPRequestType() types.Type {
	T := types.NewBuilder()

	// Header type: {name: String, value: String}
	headerType := T.Record(
		types.Field("name", T.String()),
		types.Field("value", T.String()),
	)

	// Response type: {status: Int, headers: List<Header>, body: String, ok: Bool}
	responseType := T.Record(
		types.Field("status", T.Int()),
		types.Field("headers", T.List(headerType)),
		types.Field("body", T.String()),
		types.Field("ok", T.Bool()),
	)

	// Function signature with effects
	return T.Func(
		T.String(),         // method
		T.String(),         // url
		T.List(headerType), // headers
		T.String(),         // body
	).Returns(
		T.App("Result", responseType, T.Con("NetError")),
	).Effects("Net")
}

// ============================================================================
// String Primitive Builtins
// ============================================================================

// registerStringCompare registers the _str_compare builtin
func registerStringCompare() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_compare",
		NumArgs: 2,
		IsPure:  true,
		Type:    makeStrCompareType,
		Impl:    strCompareImpl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_compare: %v", err))
	}
}

func makeStrCompareType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(T.Int()).Build()
}

func strCompareImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	a := args[0].(*eval.StringValue)
	b := args[1].(*eval.StringValue)

	if a.Value < b.Value {
		return &eval.IntValue{Value: -1}, nil
	} else if a.Value > b.Value {
		return &eval.IntValue{Value: 1}, nil
	}
	return &eval.IntValue{Value: 0}, nil
}

// registerStringEq registers the _str_eq builtin (for JSON accessors)
func registerStringEq() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_eq",
		NumArgs: 2,
		IsPure:  true,
		Type:    makeStrEqType,
		Impl:    strEqImpl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_eq: %v", err))
	}
}

func makeStrEqType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(T.Bool()).Build()
}

func strEqImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	a := args[0].(*eval.StringValue)
	b := args[1].(*eval.StringValue)
	return &eval.BoolValue{Value: a.Value == b.Value}, nil
}

// registerStringFind registers the _str_find builtin
func registerStringFind() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_find",
		NumArgs: 2,
		IsPure:  true,
		Type:    makeStrFindType,
		Impl:    strFindImpl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_find: %v", err))
	}
}

func makeStrFindType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(T.Int()).Build()
}

func strFindImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	s := args[0].(*eval.StringValue)
	sub := args[1].(*eval.StringValue)

	// Find byte index first
	byteIdx := strings.Index(s.Value, sub.Value)
	if byteIdx == -1 {
		return &eval.IntValue{Value: -1}, nil
	}

	// Convert byte index to rune index
	runeIdx := utf8.RuneCountInString(s.Value[:byteIdx])
	return &eval.IntValue{Value: runeIdx}, nil
}

// registerStringSlice registers the _str_slice builtin
func registerStringSlice() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_slice",
		NumArgs: 3,
		IsPure:  true,
		Type:    makeStrSliceType,
		Impl:    strSliceImpl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_slice: %v", err))
	}
}

func makeStrSliceType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.Int(), T.Int()).Returns(T.String()).Build()
}

func strSliceImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	s := args[0].(*eval.StringValue)
	start := args[1].(*eval.IntValue)
	end := args[2].(*eval.IntValue)

	runes := []rune(s.Value)
	length := len(runes)

	// Clamp indices to valid range
	st := start.Value
	if st < 0 {
		st = 0
	}
	if st > length {
		st = length
	}

	en := end.Value
	if en < st {
		en = st
	}
	if en > length {
		en = length
	}

	return &eval.StringValue{Value: string(runes[st:en])}, nil
}

// registerStringTrim registers the _str_trim builtin
func registerStringTrim() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_trim",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeStrTrimType,
		Impl:    strTrimImpl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_trim: %v", err))
	}
}

func makeStrTrimType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.String()).Build()
}

func strTrimImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	s := args[0].(*eval.StringValue)
	return &eval.StringValue{Value: strings.TrimSpace(s.Value)}, nil
}

// registerStringUpper registers the _str_upper builtin
func registerStringUpper() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_upper",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeStrUpperType,
		Impl:    strUpperImpl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_upper: %v", err))
	}
}

func makeStrUpperType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.String()).Build()
}

func strUpperImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	s := args[0].(*eval.StringValue)
	return &eval.StringValue{Value: strings.ToUpper(s.Value)}, nil
}

// registerStringLower registers the _str_lower builtin
func registerStringLower() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_lower",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeStrLowerType,
		Impl:    strLowerImpl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_lower: %v", err))
	}
}

func makeStrLowerType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.String()).Build()
}

func strLowerImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	s := args[0].(*eval.StringValue)
	return &eval.StringValue{Value: strings.ToLower(s.Value)}, nil
}

// ============================================================================
// Arithmetic Builtins (Int and Float operations)
// ============================================================================

func registerArithmetic() {
	// Integer arithmetic
	registerBuiltin("add_Int", 2, true, intIntToInt(func(a, b int) int { return a + b }))
	registerBuiltin("sub_Int", 2, true, intIntToInt(func(a, b int) int { return a - b }))
	registerBuiltin("mul_Int", 2, true, intIntToInt(func(a, b int) int { return a * b }))
	registerBuiltin("div_Int", 2, true, intIntToIntErr(func(a, b int) (int, error) {
		if b == 0 {
			return 0, eval.NewRuntimeError("RT_DIV0", "Division by zero", nil)
		}
		return a / b, nil
	}))
	registerBuiltin("mod_Int", 2, true, intIntToIntErr(func(a, b int) (int, error) {
		if b == 0 {
			return 0, eval.NewRuntimeError("RT_DIV0", "Modulo by zero", nil)
		}
		return a % b, nil
	}))
	registerBuiltin("neg_Int", 1, true, intToInt(func(a int) int { return -a }))

	// Float arithmetic (with special IEEE 754 behavior)
	registerBuiltin("add_Float", 2, true, floatFloatToFloat(func(a, b float64) float64 { return a + b }))
	registerBuiltin("sub_Float", 2, true, floatFloatToFloat(func(a, b float64) float64 { return a - b }))
	registerBuiltin("mul_Float", 2, true, floatFloatToFloat(func(a, b float64) float64 { return a * b }))
	registerBuiltin("div_Float", 2, true, floatDivFloat)
	registerBuiltin("mod_Float", 2, true, floatModFloat)
	registerBuiltin("neg_Float", 1, true, floatNegFloat)
}

// floatDivFloat: division with IEEE 754 behavior (returns Inf for div-by-zero)
func floatDivFloat(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	a := args[0].(*eval.FloatValue)
	b := args[1].(*eval.FloatValue)
	if b.Value == 0.0 {
		// IEEE 754 behavior: return +/-Inf
		if a.Value >= 0 {
			return &eval.FloatValue{Value: math.Inf(1)}, nil
		}
		return &eval.FloatValue{Value: math.Inf(-1)}, nil
	}
	return &eval.FloatValue{Value: a.Value / b.Value}, nil
}

// floatModFloat: modulo with IEEE 754 behavior (returns NaN for mod-by-zero)
func floatModFloat(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	a := args[0].(*eval.FloatValue)
	b := args[1].(*eval.FloatValue)
	if b.Value == 0.0 {
		return &eval.FloatValue{Value: math.NaN()}, nil
	}
	return &eval.FloatValue{Value: math.Mod(a.Value, b.Value)}, nil
}

// floatNegFloat: negation
func floatNegFloat(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	a := args[0].(*eval.FloatValue)
	return &eval.FloatValue{Value: -a.Value}, nil
}

// Helper: wrap a simple int->int function
func intToInt(fn func(int) int) func(*effects.EffContext, []eval.Value) (eval.Value, error) {
	return func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.IntValue)
		return &eval.IntValue{Value: fn(a.Value)}, nil
	}
}

// Helper: wrap a simple (int,int)->int function
func intIntToInt(fn func(int, int) int) func(*effects.EffContext, []eval.Value) (eval.Value, error) {
	return func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.IntValue)
		b := args[1].(*eval.IntValue)
		return &eval.IntValue{Value: fn(a.Value, b.Value)}, nil
	}
}

// Helper: wrap a (int,int)->(int,error) function
func intIntToIntErr(fn func(int, int) (int, error)) func(*effects.EffContext, []eval.Value) (eval.Value, error) {
	return func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.IntValue)
		b := args[1].(*eval.IntValue)
		result, err := fn(a.Value, b.Value)
		if err != nil {
			return nil, err
		}
		return &eval.IntValue{Value: result}, nil
	}
}

// Helper: wrap a (float,float)->float function
func floatFloatToFloat(fn func(float64, float64) float64) func(*effects.EffContext, []eval.Value) (eval.Value, error) {
	return func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.FloatValue)
		b := args[1].(*eval.FloatValue)
		return &eval.FloatValue{Value: fn(a.Value, b.Value)}, nil
	}
}

// registerBuiltin is a convenience wrapper for simple arithmetic builtins
func registerBuiltin(name string, numArgs int, isPure bool, impl func(*effects.EffContext, []eval.Value) (eval.Value, error)) {
	// Determine type based on name suffix
	var typeFunc func() types.Type
	if strings.HasSuffix(name, "_Int") {
		if numArgs == 1 {
			typeFunc = func() types.Type {
				T := types.NewBuilder()
				return T.Func(T.Int()).Returns(T.Int()).Build()
			}
		} else if numArgs == 2 {
			typeFunc = func() types.Type {
				T := types.NewBuilder()
				return T.Func(T.Int(), T.Int()).Returns(T.Int()).Build()
			}
		}
	} else if strings.HasSuffix(name, "_Float") {
		if numArgs == 1 {
			typeFunc = func() types.Type {
				T := types.NewBuilder()
				return T.Func(T.Float()).Returns(T.Float()).Build()
			}
		} else if numArgs == 2 {
			typeFunc = func() types.Type {
				T := types.NewBuilder()
				return T.Func(T.Float(), T.Float()).Returns(T.Float()).Build()
			}
		}
	}

	if typeFunc == nil {
		panic(fmt.Sprintf("cannot infer type for builtin %s", name))
	}

	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/math", // or "std/prelude" - arithmetic is fundamental
		Name:    name,
		NumArgs: numArgs,
		IsPure:  isPure,
		Type:    typeFunc,
		Impl:    impl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// ============================================================================
// Comparison Builtins (eq, ne, lt, le, gt, ge for Int, Float, String, Bool)
// ============================================================================

func registerComparisons() {
	// Int comparisons
	registerCmp("eq_Int", func(a, b int) bool { return a == b })
	registerCmp("ne_Int", func(a, b int) bool { return a != b })
	registerCmp("lt_Int", func(a, b int) bool { return a < b })
	registerCmp("le_Int", func(a, b int) bool { return a <= b })
	registerCmp("gt_Int", func(a, b int) bool { return a > b })
	registerCmp("ge_Int", func(a, b int) bool { return a >= b })

	// Float comparisons
	registerCmpFloat("eq_Float", func(a, b float64) bool { return a == b })
	registerCmpFloat("ne_Float", func(a, b float64) bool { return a != b })
	registerCmpFloat("lt_Float", func(a, b float64) bool { return a < b })
	registerCmpFloat("le_Float", func(a, b float64) bool { return a <= b })
	registerCmpFloat("gt_Float", func(a, b float64) bool { return a > b })
	registerCmpFloat("ge_Float", func(a, b float64) bool { return a >= b })

	// String comparisons
	registerCmpString("eq_String", func(a, b string) bool { return a == b })
	registerCmpString("ne_String", func(a, b string) bool { return a != b })
	registerCmpString("lt_String", func(a, b string) bool { return a < b })
	registerCmpString("le_String", func(a, b string) bool { return a <= b })
	registerCmpString("gt_String", func(a, b string) bool { return a > b })
	registerCmpString("ge_String", func(a, b string) bool { return a >= b })

	// Bool comparisons
	registerCmpBool("eq_Bool", func(a, b bool) bool { return a == b })
	registerCmpBool("ne_Bool", func(a, b bool) bool { return a != b })
}

func registerCmp(name string, fn func(int, int) bool) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.IntValue)
		b := args[1].(*eval.IntValue)
		return &eval.BoolValue{Value: fn(a.Value, b.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Int(), T.Int()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/prelude", Name: name, NumArgs: 2, IsPure: true, Type: typeFunc, Impl: impl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

func registerCmpFloat(name string, fn func(float64, float64) bool) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.FloatValue)
		b := args[1].(*eval.FloatValue)
		return &eval.BoolValue{Value: fn(a.Value, b.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Float(), T.Float()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/prelude", Name: name, NumArgs: 2, IsPure: true, Type: typeFunc, Impl: impl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

func registerCmpString(name string, fn func(string, string) bool) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.StringValue)
		b := args[1].(*eval.StringValue)
		return &eval.BoolValue{Value: fn(a.Value, b.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String(), T.String()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/prelude", Name: name, NumArgs: 2, IsPure: true, Type: typeFunc, Impl: impl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

func registerCmpBool(name string, fn func(bool, bool) bool) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.BoolValue)
		b := args[1].(*eval.BoolValue)
		return &eval.BoolValue{Value: fn(a.Value, b.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Bool(), T.Bool()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/prelude", Name: name, NumArgs: 2, IsPure: true, Type: typeFunc, Impl: impl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// ============================================================================
// Logic Builtins (and, or, not)
// ============================================================================

func registerLogic() {
	registerLogicOp("and_Bool", func(a, b bool) bool { return a && b })
	registerLogicOp("or_Bool", func(a, b bool) bool { return a || b })
	registerLogicUnary("not_Bool", func(a bool) bool { return !a })
}

func registerLogicOp(name string, fn func(bool, bool) bool) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.BoolValue)
		b := args[1].(*eval.BoolValue)
		return &eval.BoolValue{Value: fn(a.Value, b.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Bool(), T.Bool()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/prelude", Name: name, NumArgs: 2, IsPure: true, Type: typeFunc, Impl: impl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

func registerLogicUnary(name string, fn func(bool) bool) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.BoolValue)
		return &eval.BoolValue{Value: fn(a.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Bool()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/prelude", Name: name, NumArgs: 1, IsPure: true, Type: typeFunc, Impl: impl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// ============================================================================
// Conversion Builtins (intToFloat, floatToInt)
// ============================================================================

func registerConversions() {
	// intToFloat
	impl1 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.IntValue)
		return &eval.FloatValue{Value: float64(a.Value)}, nil
	}
	type1 := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Int()).Returns(T.Float()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/prelude", Name: "intToFloat", NumArgs: 1, IsPure: true, Type: type1, Impl: impl1,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register intToFloat: %v", err))
	}

	// floatToInt
	impl2 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.FloatValue)
		return &eval.IntValue{Value: int(a.Value)}, nil
	}
	type2 := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Float()).Returns(T.Int()).Build()
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/prelude", Name: "floatToInt", NumArgs: 1, IsPure: true, Type: type2, Impl: impl2,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register floatToInt: %v", err))
	}
}

// ============================================================================
// String Operations (concat)
// ============================================================================

func registerStringConcat() {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.StringValue)
		b := args[1].(*eval.StringValue)
		return &eval.StringValue{Value: a.Value + b.Value}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String(), T.String()).Returns(T.String()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/string", Name: "concat_String", NumArgs: 2, IsPure: true, Type: typeFunc, Impl: impl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register concat_String: %v", err))
	}
}

// ============================================================================
// IO Effect Builtins (_io_print, _io_println, _io_readLine)
// ============================================================================

func registerIO() {
	// _io_print
	impl1 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		s := args[0].(*eval.StringValue)
		fmt.Print(s.Value)
		return &eval.UnitValue{}, nil
	}
	type1 := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String()).Returns(T.Unit()).Effects("IO")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/io", Name: "_io_print", NumArgs: 1, IsPure: false, Effect: "IO", Type: type1, Impl: impl1,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _io_print: %v", err))
	}

	// _io_println
	impl2 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		s := args[0].(*eval.StringValue)
		fmt.Println(s.Value)
		return &eval.UnitValue{}, nil
	}
	type2 := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String()).Returns(T.Unit()).Effects("IO")
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/io", Name: "_io_println", NumArgs: 1, IsPure: false, Effect: "IO", Type: type2, Impl: impl2,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _io_println: %v", err))
	}

	// _io_readLine (stub for v0.3.10)
	impl3 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		// Stub: return empty string
		return &eval.StringValue{Value: ""}, nil
	}
	type3 := func() types.Type {
		T := types.NewBuilder()
		return T.Func().Returns(T.String()).Effects("IO")
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/io", Name: "_io_readLine", NumArgs: 0, IsPure: false, Effect: "IO", Type: type3, Impl: impl3,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _io_readLine: %v", err))
	}
}

// ============================================================================
// JSON Builtins (_json_encode)
// ============================================================================

func registerJSON() {
	// _json_encode is already registered in internal/eval/builtins.go
	// It has complex logic for encoding Json ADT, so we'll keep it there for now
	// TODO: Migrate in future iteration
}
