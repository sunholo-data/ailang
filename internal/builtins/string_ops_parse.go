package builtins

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// ============================================================================
// String Parsing Builtins (M-DX10)
// ============================================================================

// registerStringToInt registers the _stringToInt builtin
func registerStringToInt() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_stringToInt",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeStringToIntType,
		Impl:    stringToIntImpl,

		Metadata: &BuiltinMetadata{
			Description: "Parse string to integer",
			LongDesc:    "Returns Some(n) if the string contains a valid integer, None otherwise. Accepts decimal integers with optional leading sign (+/-).",
			Params: []ParamDoc{
				{Name: "s", Description: "String to parse as integer"},
			},
			Returns: "Option[int]: Some(n) if valid, None if invalid",
			Examples: []Example{
				{Code: `_stringToInt("42")`, Description: "Returns Some(42)"},
				{Code: `_stringToInt("-123")`, Description: "Returns Some(-123)"},
				{Code: `_stringToInt("abc")`, Description: "Returns None"},
				{Code: `_stringToInt("3.14")`, Description: "Returns None (not an integer)"},
			},
			SeeAlso:   []string{"_stringToFloat", "_str_len"},
			Since:     "v0.4.3",
			Stability: StabilityStable,
			Tags:      []string{"string", "parse", "integer", "conversion", "option"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _stringToInt: %v", err))
	}
}

// makeStringToIntType builds the type signature for _stringToInt
// Type: (String) -> Option[Int]
func makeStringToIntType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(
		T.App("Option", T.Int()),
	).Build()
}

// stringToIntImpl is the implementation for _stringToInt
// Parses string to int64, returns Option[int]
func stringToIntImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_stringToInt: expected String, got %T", args[0])
	}

	// Try to parse as int64
	n, err := strconv.ParseInt(strVal.Value, 10, 64)
	if err != nil {
		// Return None
		return &eval.TaggedValue{
			ModulePath: "std/option",
			TypeName:   "Option",
			CtorName:   "None",
			Fields:     []eval.Value{},
		}, nil
	}

	// Return Some(n)
	return &eval.TaggedValue{
		ModulePath: "std/option",
		TypeName:   "Option",
		CtorName:   "Some",
		Fields:     []eval.Value{&eval.IntValue{Value: int(n)}},
	}, nil
}

// registerStringToFloat registers the _stringToFloat builtin
func registerStringToFloat() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_stringToFloat",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeStringToFloatType,
		Impl:    stringToFloatImpl,

		Metadata: &BuiltinMetadata{
			Description: "Parse string to floating-point number",
			LongDesc:    "Returns Some(f) if the string contains a valid floating-point number, None otherwise. Accepts decimal notation (3.14), scientific notation (1e-10), and optional sign.",
			Params: []ParamDoc{
				{Name: "s", Description: "String to parse as float"},
			},
			Returns: "Option[float]: Some(f) if valid, None if invalid",
			Examples: []Example{
				{Code: `_stringToFloat("3.14")`, Description: "Returns Some(3.14)"},
				{Code: `_stringToFloat("-2.5")`, Description: "Returns Some(-2.5)"},
				{Code: `_stringToFloat("1e-10")`, Description: "Returns Some(1e-10)"},
				{Code: `_stringToFloat("abc")`, Description: "Returns None"},
				{Code: `_stringToFloat("")`, Description: "Returns None"},
			},
			SeeAlso:   []string{"_stringToInt", "_str_len"},
			Since:     "v0.4.3",
			Stability: StabilityStable,
			Tags:      []string{"string", "parse", "float", "conversion", "option"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _stringToFloat: %v", err))
	}
}

// makeStringToFloatType builds the type signature for _stringToFloat
// Type: (String) -> Option[Float]
func makeStringToFloatType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(
		T.App("Option", T.Float()),
	).Build()
}

// stringToFloatImpl is the implementation for _stringToFloat
// Parses string to float64, returns Option[float]
func stringToFloatImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_stringToFloat: expected String, got %T", args[0])
	}

	// Try to parse as float64
	// Reject strings with underscores: Go's ParseFloat silently accepts them
	// as digit separators (e.g., "1_000" → 1000.0), which violates user expectations.
	if strings.ContainsRune(strVal.Value, '_') {
		return &eval.TaggedValue{
			ModulePath: "std/option",
			TypeName:   "Option",
			CtorName:   "None",
			Fields:     []eval.Value{},
		}, nil
	}
	f, err := strconv.ParseFloat(strVal.Value, 64)
	if err != nil {
		// Return None
		return &eval.TaggedValue{
			ModulePath: "std/option",
			TypeName:   "Option",
			CtorName:   "None",
			Fields:     []eval.Value{},
		}, nil
	}

	// Return Some(f)
	return &eval.TaggedValue{
		ModulePath: "std/option",
		TypeName:   "Option",
		CtorName:   "Some",
		Fields:     []eval.Value{&eval.FloatValue{Value: f}},
	}, nil
}

// registerStrSplit registers the _str_split builtin
func registerStrSplit() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_split",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrSplitType,
		Impl:    strSplitImpl,

		Metadata: &BuiltinMetadata{
			Description: "Split string by delimiter into list of strings",
			LongDesc:    "Splits a string at each occurrence of the delimiter. Empty delimiter splits into individual UTF-8 codepoints. Matches Go's strings.Split() semantics exactly, including edge cases.",
			Params: []ParamDoc{
				{Name: "s", Description: "String to split"},
				{Name: "delimiter", Description: "Delimiter to split on (empty = split into characters)"},
			},
			Returns: "[string] - List of substrings",
			Examples: []Example{
				{Code: `_str_split("a,b,c", ",")`, Description: `Returns ["a", "b", "c"]`},
				{Code: `_str_split("hello", "")`, Description: `Returns ["h", "e", "l", "l", "o"]`},
				{Code: `_str_split("", ",")`, Description: `Returns [""] (single empty string)`},
				{Code: `_str_split("", "")`, Description: `Returns [] (empty list - special case)`},
			},
			SeeAlso:   []string{"_str_slice", "_str_find", "_str_trim"},
			Since:     "v0.4.7",
			Stability: StabilityStable,
			Tags:      []string{"string", "split", "parse", "delimiter", "list"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_split: %v", err))
	}
}

// makeStrSplitType builds the type signature for _str_split
// Type: string -> string -> [string]
func makeStrSplitType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(T.List(T.String())).Build()
}

// strSplitImpl is the implementation for _str_split
// Uses Go's strings.Split for exact standard behavior
func strSplitImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Extract string arguments with safe casting
	str, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_str_split: arg 0 - %w", err)
	}

	delim, err := SafeAsString(args[1])
	if err != nil {
		return nil, fmt.Errorf("_str_split: arg 1 - %w", err)
	}

	// Use Go's strings.Split for exact standard behavior
	// This handles all edge cases including split("", "") -> []
	parts := strings.Split(str, delim)

	// Convert []string to [string] (ListValue with Elements slice)
	elements := make([]eval.Value, len(parts))
	for i, part := range parts {
		elements[i] = &eval.StringValue{Value: part}
	}

	return &eval.ListValue{Elements: elements}, nil
}
