package builtins

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// String builtin functions for AILANG
// These provide UTF-8 aware string operations

func init() {
	registerStringLen()
	registerStringCompare()
	registerStringEq()
	registerStringFind()
	registerStringSlice()
	registerStringTrim()
	registerStringUpper()
	registerStringLower()
	registerStringConcat()
	registerStringToInt()
	registerStringToFloat()
	registerStrSplit()
}

// ============================================================================
// String Primitive Builtins
// ============================================================================

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

		Metadata: &BuiltinMetadata{
			Description: "Get the length of a string in Unicode characters",
			LongDesc:    "Returns the number of Unicode code points (runes) in the string, not bytes. This correctly handles multi-byte UTF-8 characters like emoji.",
			Params: []ParamDoc{
				{Name: "s", Description: "The string to measure"},
			},
			Returns: "Number of Unicode characters in the string",
			Examples: []Example{
				{Code: `_str_len("hello")`, Description: "Returns 5"},
				{Code: `_str_len("🎉")`, Description: "Returns 1 (not 4 bytes)"},
			},
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      []string{"string", "length", "unicode", "utf8"},
			Category:  "string",
		},
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

// registerStringCompare registers the _str_compare builtin
func registerStringCompare() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_compare",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrCompareType,
		Impl:    strCompareImpl,

		Metadata: &BuiltinMetadata{
			Description: "Compare two strings lexicographically",
			LongDesc:    "Returns -1 if first string is less than second, 0 if equal, 1 if greater. Uses Unicode code point comparison.",
			Params: []ParamDoc{
				{Name: "a", Description: "First string"},
				{Name: "b", Description: "Second string"},
			},
			Returns: "-1 (a < b), 0 (a == b), or 1 (a > b)",
			Examples: []Example{
				{Code: `_str_compare("apple", "banana")`, Description: "Returns -1"},
				{Code: `_str_compare("hello", "hello")`, Description: "Returns 0"},
			},
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      []string{"string", "compare", "order", "sort"},
			Category:  "string",
		},
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
	a := args[0].(*eval.StringValue).Value
	b := args[1].(*eval.StringValue).Value
	result := 0
	if a < b {
		result = -1
	} else if a > b {
		result = 1
	}
	return &eval.IntValue{Value: result}, nil
}

// registerStringEq registers the _str_eq builtin
func registerStringEq() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_eq",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrEqType,
		Impl:    strEqImpl,

		Metadata: &BuiltinMetadata{
			Description: "Test if two strings are equal",
			Params: []ParamDoc{
				{Name: "a", Description: "First string"},
				{Name: "b", Description: "Second string"},
			},
			Returns: "true if strings are equal, false otherwise",
			Examples: []Example{
				{Code: `_str_eq("hello", "hello")`, Description: "Returns true"},
			},
			Since:     "v0.3.14",
			Stability: StabilityStable,
			Tags:      []string{"string", "equality", "compare"},
			Category:  "string",
		},
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
	a := args[0].(*eval.StringValue).Value
	b := args[1].(*eval.StringValue).Value
	return &eval.BoolValue{Value: a == b}, nil
}

// registerStringFind registers the _str_find builtin
func registerStringFind() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_find",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrFindType,
		Impl:    strFindImpl,

		Metadata: &BuiltinMetadata{
			Description: "Find the index of a substring within a string",
			Params: []ParamDoc{
				{Name: "haystack", Description: "String to search in"},
				{Name: "needle", Description: "Substring to find"},
			},
			Returns: "Index of first occurrence, or -1 if not found",
			Examples: []Example{
				{Code: `_str_find("hello world", "world")`, Description: "Returns 6"},
				{Code: `_str_find("hello", "xyz")`, Description: "Returns -1"},
			},
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      []string{"string", "search", "find", "indexOf"},
			Category:  "string",
		},
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
	haystack := args[0].(*eval.StringValue).Value
	needle := args[1].(*eval.StringValue).Value
	idx := strings.Index(haystack, needle)
	return &eval.IntValue{Value: idx}, nil
}

// registerStringSlice registers the _str_slice builtin
func registerStringSlice() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_slice",
		NumArgs: 3,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrSliceType,
		Impl:    strSliceImpl,

		Metadata: &BuiltinMetadata{
			Description: "Extract a substring by character range",
			LongDesc:    "Returns characters from start index (inclusive) to end index (exclusive). Indices are clamped to valid ranges. Works with Unicode characters correctly.",
			Params: []ParamDoc{
				{Name: "s", Description: "String to slice"},
				{Name: "start", Description: "Start index (inclusive), clamped to 0 if negative"},
				{Name: "end", Description: "End index (exclusive), clamped to length if too large"},
			},
			Returns: "Substring from start to end",
			Examples: []Example{
				{Code: `_str_slice("hello", 1, 4)`, Description: `Returns "ell"`},
				{Code: `_str_slice("hello", 0, 100)`, Description: `Returns "hello" (clamped)`},
			},
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      []string{"string", "slice", "substring", "range"},
			Category:  "string",
		},
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
	str := args[0].(*eval.StringValue).Value
	start := args[1].(*eval.IntValue).Value
	end := args[2].(*eval.IntValue).Value

	runes := []rune(str)
	length := len(runes)

	// Clamp indices
	if start < 0 {
		start = 0
	}
	if end > length {
		end = length
	}
	if start > end {
		start = end
	}

	return &eval.StringValue{Value: string(runes[start:end])}, nil
}

// registerStringTrim registers the _str_trim builtin
func registerStringTrim() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_trim",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrTrimType,
		Impl:    strTrimImpl,

		Metadata: &BuiltinMetadata{
			Description: "Remove leading and trailing whitespace",
			Params: []ParamDoc{
				{Name: "s", Description: "String to trim"},
			},
			Returns: "String with whitespace removed from both ends",
			Examples: []Example{
				{Code: `_str_trim("  hello  ")`, Description: `Returns "hello"`},
			},
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      []string{"string", "trim", "whitespace", "strip"},
			Category:  "string",
		},
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
	str := args[0].(*eval.StringValue).Value
	return &eval.StringValue{Value: strings.TrimSpace(str)}, nil
}

// registerStringUpper registers the _str_upper builtin
func registerStringUpper() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_upper",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrUpperType,
		Impl:    strUpperImpl,

		Metadata: &BuiltinMetadata{
			Description: "Convert string to uppercase",
			Params: []ParamDoc{
				{Name: "s", Description: "String to convert"},
			},
			Returns: "String with all characters converted to uppercase",
			Examples: []Example{
				{Code: `_str_upper("hello")`, Description: `Returns "HELLO"`},
			},
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      []string{"string", "uppercase", "case", "convert"},
			Category:  "string",
		},
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
	str := args[0].(*eval.StringValue).Value
	return &eval.StringValue{Value: strings.ToUpper(str)}, nil
}

// registerStringLower registers the _str_lower builtin
func registerStringLower() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_lower",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrLowerType,
		Impl:    strLowerImpl,

		Metadata: &BuiltinMetadata{
			Description: "Convert string to lowercase",
			Params: []ParamDoc{
				{Name: "s", Description: "String to convert"},
			},
			Returns: "String with all characters converted to lowercase",
			Examples: []Example{
				{Code: `_str_lower("HELLO")`, Description: `Returns "hello"`},
			},
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      []string{"string", "lowercase", "case", "convert"},
			Category:  "string",
		},
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
	str := args[0].(*eval.StringValue).Value
	return &eval.StringValue{Value: strings.ToLower(str)}, nil
}

// ============================================================================
// String Operations (concat)
// ============================================================================

// registerStringConcat registers the concat_String builtin
func registerStringConcat() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "concat_String",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrConcatType,
		Impl:    strConcatImpl,

		Metadata: &BuiltinMetadata{
			Description: "Concatenate two strings",
			Params: []ParamDoc{
				{Name: "a", Description: "First string"},
				{Name: "b", Description: "Second string"},
			},
			Returns: "String formed by appending b to a",
			Examples: []Example{
				{Code: `concat_String("hello", " world")`, Description: `Returns "hello world"`},
			},
			SeeAlso:   []string{"_str_slice", "_str_find"},
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      []string{"string", "concat", "append", "join"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register concat_String: %v", err))
	}
}

func makeStrConcatType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(T.String()).Build()
}

func strConcatImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	a := args[0].(*eval.StringValue).Value
	b := args[1].(*eval.StringValue).Value
	return &eval.StringValue{Value: a + b}, nil
}

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
	// Extract string arguments (note: pointers!)
	str, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_str_split: first argument must be string, got %T", args[0])
	}

	delim, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_str_split: second argument must be string, got %T", args[1])
	}

	// Use Go's strings.Split for exact standard behavior
	// This handles all edge cases including split("", "") -> []
	parts := strings.Split(str.Value, delim.Value)

	// Convert []string to [string] (ListValue with Elements slice)
	elements := make([]eval.Value, len(parts))
	for i, part := range parts {
		elements[i] = &eval.StringValue{Value: part}
	}

	return &eval.ListValue{Elements: elements}, nil
}
