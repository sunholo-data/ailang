package builtins

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
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

// registerStringReverse registers the _string_reverse builtin
func registerStringReverse() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_string_reverse",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeStringReverseType,
		Impl:    stringReverseImpl,

		Metadata: &BuiltinMetadata{
			Description: "Reverse a string",
			LongDesc:    "Returns a new string with Unicode characters in reverse order. Correctly handles multi-byte UTF-8 characters like emoji.",
			Params: []ParamDoc{
				{Name: "s", Description: "The string to reverse"},
			},
			Returns: "A new string with characters reversed",
			Examples: []Example{
				{Code: `_string_reverse("hello")`, Description: `Returns "olleh"`},
				{Code: `_string_reverse("")`, Description: `Returns ""`},
				{Code: `_string_reverse("🎉")`, Description: `Returns "🎉" (single character unchanged)`},
			},
			Since:     "v0.6.2",
			Stability: StabilityStable,
			Tags:      []string{"string", "reverse", "unicode", "utf8"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _string_reverse: %v", err))
	}
}

// makeStringReverseType builds the type signature for _string_reverse
// Type: (String) -> String
func makeStringReverseType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.String()).Build()
}

// stringReverseImpl is the implementation for _string_reverse
// Reverses a UTF-8 string at the rune (character) level
func stringReverseImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Extract string argument
	str, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_string_reverse: arg 0 - %w", err)
	}

	// Convert string to runes for proper Unicode handling
	runes := []rune(str)

	// Reverse the rune slice in-place
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	// Convert back to string
	reversed := string(runes)

	return &eval.StringValue{Value: reversed}, nil
}

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
// String Join Builtin (M-PERF5)
// ============================================================================

// registerStrJoin registers the _str_join builtin
// Replaces O(n²) recursive AILANG join with single-allocation strings.Join
func registerStrJoin() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_join",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrJoinType,
		Impl:    strJoinImpl,

		Metadata: &BuiltinMetadata{
			Description: "Join list of strings with separator",
			LongDesc:    "Joins a list of strings with a separator between each element. Uses Go's strings.Join for O(n) performance instead of O(n²) recursive concatenation.",
			Params: []ParamDoc{
				{Name: "parts", Description: "List of strings to join"},
				{Name: "separator", Description: "String to insert between elements"},
			},
			Returns: "Single string with all parts joined by separator",
			Examples: []Example{
				{Code: `_str_join(["a", "b", "c"], ", ")`, Description: `Returns "a, b, c"`},
				{Code: `_str_join([], ", ")`, Description: `Returns ""`},
				{Code: `_str_join(["hello"], "")`, Description: `Returns "hello"`},
			},
			SeeAlso:   []string{"_str_split", "concat_String"},
			Since:     "v0.9.2",
			Stability: StabilityStable,
			Tags:      []string{"string", "join", "concat", "list"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_join: %v", err))
	}
}

// makeStrJoinType builds the type signature for _str_join
// Type: (List[string], string) -> string
func makeStrJoinType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.List(T.String()), T.String()).Returns(T.String()).Build()
}

// strJoinImpl is the implementation for _str_join
// Uses Go's strings.Join for single-allocation O(n) performance
func strJoinImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	listVal, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_str_join: expected List for parts, got %T", args[0])
	}

	sep, err := SafeAsString(args[1])
	if err != nil {
		return nil, fmt.Errorf("_str_join: arg 1 (separator) - %w", err)
	}

	parts := make([]string, len(listVal.Elements))
	for i, elem := range listVal.Elements {
		s, err := SafeAsString(elem)
		if err != nil {
			return nil, fmt.Errorf("_str_join: element %d - %w", i, err)
		}
		parts[i] = s
	}

	return &eval.StringValue{Value: strings.Join(parts, sep)}, nil
}

// ============================================================================
// M-DOCPARSE-DX M3: String Splitting
// ============================================================================

// registerStrWords registers _str_words: split on whitespace
func registerStrWords() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_words",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrWordsType,
		Impl:    strWordsImpl,
		Metadata: &BuiltinMetadata{
			Description: "Split string on whitespace, dropping empty segments",
			Params: []ParamDoc{
				{Name: "s", Description: "String to split"},
			},
			Returns:   "List of non-empty words",
			Examples:  []Example{{Code: `_str_words("hello  world\tfoo")`, Description: `Returns ["hello", "world", "foo"]`}},
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"string", "split", "words", "whitespace"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_words: %v", err))
	}
}

func makeStrWordsType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.List(T.String())).Build()
}

func strWordsImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_str_words: expected String, got %T", args[0])
	}
	words := strings.Fields(strVal.Value)
	result := make([]eval.Value, len(words))
	for i, w := range words {
		result[i] = &eval.StringValue{Value: w}
	}
	return &eval.ListValue{Elements: result}, nil
}

// registerStrSplitAny registers _str_splitAny: split on any of multiple delimiters
func registerStrSplitAny() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_splitAny",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrSplitAnyType,
		Impl:    strSplitAnyImpl,
		Metadata: &BuiltinMetadata{
			Description: "Split string on any of the given delimiter strings",
			Params: []ParamDoc{
				{Name: "s", Description: "String to split"},
				{Name: "delimiters", Description: "List of delimiter strings (single characters)"},
			},
			Returns:   "List of non-empty segments",
			Examples:  []Example{{Code: `_str_splitAny("a,b;c", [",", ";"])`, Description: `Returns ["a", "b", "c"]`}},
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"string", "split", "delimiter", "multi"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_splitAny: %v", err))
	}
}

func makeStrSplitAnyType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.List(T.String())).Returns(T.List(T.String())).Build()
}

func strSplitAnyImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_str_splitAny: expected String, got %T", args[0])
	}
	delimList, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_str_splitAny: expected List for delimiters, got %T", args[1])
	}

	// Collect delimiter runes
	delimRunes := make(map[rune]bool)
	for _, d := range delimList.Elements {
		ds, ok := d.(*eval.StringValue)
		if !ok {
			return nil, fmt.Errorf("_str_splitAny: delimiter must be String, got %T", d)
		}
		for _, r := range ds.Value {
			delimRunes[r] = true
		}
	}

	// Split using FieldsFunc
	parts := strings.FieldsFunc(strVal.Value, func(r rune) bool {
		return delimRunes[r]
	})

	result := make([]eval.Value, len(parts))
	for i, p := range parts {
		result[i] = &eval.StringValue{Value: p}
	}
	return &eval.ListValue{Elements: result}, nil
}
