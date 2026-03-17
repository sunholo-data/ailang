package builtins

import (
	"fmt"
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
	registerStringReverse()
	registerStringChars()
	registerStringStartsWith()
	registerStringEndsWith()
	registerStrJoin()
	registerStrWords()
	registerStrSplitAny()
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
	a, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_str_compare: arg 0 - %w", err)
	}
	b, err := SafeAsString(args[1])
	if err != nil {
		return nil, fmt.Errorf("_str_compare: arg 1 - %w", err)
	}
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
	a, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_str_eq: arg 0 - %w", err)
	}
	b, err := SafeAsString(args[1])
	if err != nil {
		return nil, fmt.Errorf("_str_eq: arg 1 - %w", err)
	}
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
	haystack, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_str_find: arg 0 - %w", err)
	}
	needle, err := SafeAsString(args[1])
	if err != nil {
		return nil, fmt.Errorf("_str_find: arg 1 - %w", err)
	}
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
	str, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_str_slice: arg 0 - %w", err)
	}
	start, err := SafeAsInt(args[1])
	if err != nil {
		return nil, fmt.Errorf("_str_slice: arg 1 - %w", err)
	}
	end, err := SafeAsInt(args[2])
	if err != nil {
		return nil, fmt.Errorf("_str_slice: arg 2 - %w", err)
	}

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
	str, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_str_trim: arg 0 - %w", err)
	}
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
	str, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_str_upper: arg 0 - %w", err)
	}
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
	str, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_str_lower: arg 0 - %w", err)
	}
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
	a, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("concat_String: arg 0 - %w", err)
	}
	b, err := SafeAsString(args[1])
	if err != nil {
		return nil, fmt.Errorf("concat_String: arg 1 - %w", err)
	}
	return &eval.StringValue{Value: a + b}, nil
}
