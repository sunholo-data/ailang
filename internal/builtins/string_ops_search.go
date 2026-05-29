package builtins

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// ============================================================================
// String Character Operations (M-DX-CHARS)
// ============================================================================

// registerStringChars registers the _str_chars builtin
func registerStringChars() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_chars",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrCharsType,
		Impl:    strCharsImpl,

		Metadata: &BuiltinMetadata{
			Description: "Convert string to list of single-character strings",
			LongDesc:    "Splits a string into individual Unicode characters, returning each as a single-character string. Correctly handles multi-byte UTF-8 characters like emoji.",
			Params: []ParamDoc{
				{Name: "s", Description: "String to convert to character list"},
			},
			Returns: "[string] - List of single-character strings",
			Examples: []Example{
				{Code: `_str_chars("abc")`, Description: `Returns ["a", "b", "c"]`},
				{Code: `_str_chars("hello")`, Description: `Returns ["h", "e", "l", "l", "o"]`},
				{Code: `_str_chars("")`, Description: `Returns [] (empty list)`},
				{Code: `_str_chars("🎉")`, Description: `Returns ["🎉"] (single character)`},
				{Code: `_str_chars("a🎉b")`, Description: `Returns ["a", "🎉", "b"]`},
			},
			SeeAlso:   []string{"_str_split", "_str_len", "_str_slice"},
			Since:     "v0.6.5",
			Stability: StabilityStable,
			Tags:      []string{"string", "characters", "list", "unicode", "utf8"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_chars: %v", err))
	}
}

// makeStrCharsType builds the type signature for _str_chars
// Type: string -> [string]
func makeStrCharsType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.List(T.String())).Build()
}

// strCharsImpl is the implementation for _str_chars
// Converts string to list of single-character strings (Unicode-aware)
func strCharsImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Extract string argument
	str, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_str_chars: arg 0 - %w", err)
	}

	// Convert string to runes for proper Unicode handling
	runes := []rune(str)

	// Build list of single-character strings
	elements := make([]eval.Value, len(runes))
	for i, r := range runes {
		elements[i] = &eval.StringValue{Value: string(r)}
	}

	return &eval.ListValue{Elements: elements}, nil
}

// registerStringStartsWith registers the _str_startsWith builtin
func registerStringStartsWith() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_startsWith",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrStartsWithType,
		Impl:    strStartsWithImpl,

		Metadata: &BuiltinMetadata{
			Description: "Check if a string starts with a given prefix",
			Params: []ParamDoc{
				{Name: "s", Description: "String to check"},
				{Name: "prefix", Description: "Prefix to look for"},
			},
			Returns: "true if s starts with prefix, false otherwise",
			Examples: []Example{
				{Code: `_str_startsWith("hello world", "hello")`, Description: "Returns true"},
				{Code: `_str_startsWith("hello world", "world")`, Description: "Returns false"},
				{Code: `_str_startsWith("hello", "")`, Description: "Returns true (empty prefix)"},
			},
			SeeAlso:   []string{"_str_endsWith", "_str_find", "_str_slice"},
			Since:     "v0.7.4",
			Stability: StabilityStable,
			Tags:      []string{"string", "prefix", "search", "match"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_startsWith: %v", err))
	}
}

func makeStrStartsWithType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(T.Bool()).Build()
}

func strStartsWithImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	s, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_str_startsWith: arg 0 - %w", err)
	}
	prefix, err := SafeAsString(args[1])
	if err != nil {
		return nil, fmt.Errorf("_str_startsWith: arg 1 - %w", err)
	}
	return &eval.BoolValue{Value: strings.HasPrefix(s, prefix)}, nil
}

// registerStringEndsWith registers the _str_endsWith builtin
func registerStringEndsWith() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_endsWith",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrEndsWithType,
		Impl:    strEndsWithImpl,

		Metadata: &BuiltinMetadata{
			Description: "Check if a string ends with a given suffix",
			Params: []ParamDoc{
				{Name: "s", Description: "String to check"},
				{Name: "suffix", Description: "Suffix to look for"},
			},
			Returns: "true if s ends with suffix, false otherwise",
			Examples: []Example{
				{Code: `_str_endsWith("hello world", "world")`, Description: "Returns true"},
				{Code: `_str_endsWith("hello world", "hello")`, Description: "Returns false"},
				{Code: `_str_endsWith("hello", "")`, Description: "Returns true (empty suffix)"},
			},
			SeeAlso:   []string{"_str_startsWith", "_str_find", "_str_slice"},
			Since:     "v0.7.4",
			Stability: StabilityStable,
			Tags:      []string{"string", "suffix", "search", "match"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_endsWith: %v", err))
	}
}

func makeStrEndsWithType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(T.Bool()).Build()
}

func strEndsWithImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	s, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_str_endsWith: arg 0 - %w", err)
	}
	suffix, err := SafeAsString(args[1])
	if err != nil {
		return nil, fmt.Errorf("_str_endsWith: arg 1 - %w", err)
	}
	return &eval.BoolValue{Value: strings.HasSuffix(s, suffix)}, nil
}

// ============================================================================
// Case-Insensitive String Matching (M-STD-STRING-PERF M5)
// ============================================================================

// registerStringStartsWithIC registers the _str_startsWithIC builtin
func registerStringStartsWithIC() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_startsWithIC",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrStartsWithICType,
		Impl:    strStartsWithICImpl,

		Metadata: &BuiltinMetadata{
			Description: "Check if a string starts with a given prefix (case-insensitive)",
			LongDesc:    "Case-insensitive prefix check using strings.EqualFold on the prefix-length slice of the input. Handles Unicode case folding correctly (e.g., German ß ↔ SS).",
			Params: []ParamDoc{
				{Name: "s", Description: "String to check"},
				{Name: "prefix", Description: "Prefix to look for (case-insensitive)"},
			},
			Returns: "true if s starts with prefix ignoring case, false otherwise",
			Examples: []Example{
				{Code: `_str_startsWithIC("Hello World", "hello")`, Description: "Returns true"},
				{Code: `_str_startsWithIC("Content-Type", "content-type")`, Description: "Returns true"},
				{Code: `_str_startsWithIC("abc", "ABCD")`, Description: "Returns false (prefix longer than string)"},
			},
			SeeAlso:   []string{"_str_startsWith", "_str_endsWith", "_str_find"},
			Since:     "v0.11.0",
			Stability: StabilityStable,
			Tags:      []string{"string", "prefix", "search", "match", "case-insensitive"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_startsWithIC: %v", err))
	}
}

func makeStrStartsWithICType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(T.Bool()).Build()
}

func strStartsWithICImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	s, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_str_startsWithIC: arg 0 - %w", err)
	}
	prefix, err := SafeAsString(args[1])
	if err != nil {
		return nil, fmt.Errorf("_str_startsWithIC: arg 1 - %w", err)
	}

	if len(prefix) > len(s) {
		return &eval.BoolValue{Value: false}, nil
	}

	// Use EqualFold on the prefix-length slice for correct Unicode case folding
	return &eval.BoolValue{Value: strings.EqualFold(s[:len(prefix)], prefix)}, nil
}
