package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-PERF7: Go-level character iteration builtins
//
// These provide direct rune iteration over strings without allocating
// intermediate list nodes. Eliminates the overhead of chars(s) + foldl
// for character-level parsers (e.g., DocParse markdown: 3.4s → <0.5s).

func init() {
	registerStrFoldChars()
	registerStrCharAt()
	registerStrCharCode()
}

// ============================================================================
// _str_foldChars: Fold over string characters without list allocation
// ============================================================================

func registerStrFoldChars() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_foldChars",
		NumArgs: 3,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrFoldCharsType,
		Impl:    strFoldCharsImpl,
		Metadata: &BuiltinMetadata{
			Description: "Fold over characters of a string without list allocation",
			LongDesc:    "Iterates over Unicode runes in Go, calling an AILANG closure per character. Avoids allocating a list[string] intermediate, making it ideal for character-level parsers. Each character is passed as a single-character string.",
			Params: []ParamDoc{
				{Name: "f", Description: "Accumulation function (acc, char) -> acc"},
				{Name: "acc", Description: "Initial accumulator value"},
				{Name: "s", Description: "String to fold over"},
			},
			Returns: "Final accumulator value after processing all characters",
			Examples: []Example{
				{Code: `foldChars(\acc c -> acc ++ c, "", "abc")`, Description: `Returns "abc"`},
			},
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"string", "fold", "characters", "iterative", "performance"},
			Category:  "string",
			SeeAlso:   []string{"_str_chars", "_list_foldl", "_str_charAt"},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_foldChars: %v", err))
	}
}

// Type: forall a. ((a, string) -> a, a, string) -> a
func makeStrFoldCharsType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	str := T.String()
	fn := T.Func(a, str).Returns(a).Build()
	return T.Func(fn, a, str).Returns(a).Build()
}

func strFoldCharsImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	fn := args[0]
	acc := args[1]
	str, err := SafeAsString(args[2])
	if err != nil {
		return nil, fmt.Errorf("_str_foldChars: expected string for third argument: %w", err)
	}

	if ctx == nil || ctx.FnCallerN == nil {
		return nil, fmt.Errorf("_str_foldChars: FnCallerN not set (evaluator not wired)")
	}

	for i, r := range str {
		charVal := &eval.StringValue{Value: string(r)}
		acc, err = ctx.FnCallerN(fn, []eval.Value{acc, charVal})
		if err != nil {
			return nil, fmt.Errorf("_str_foldChars: callback error at byte offset %d: %w", i, err)
		}
	}
	return acc, nil
}

// ============================================================================
// _str_charAt: Rune-safe character indexing
// ============================================================================

func registerStrCharAt() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_charAt",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrCharAtType,
		Impl:    strCharAtImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get character at rune index (0-based)",
			LongDesc:    "Returns the character at the given rune index as a single-character string. O(n) rune conversion per call — use foldChars for sequential access over all characters.",
			Params: []ParamDoc{
				{Name: "s", Description: "The string to index into"},
				{Name: "i", Description: "Zero-based rune index"},
			},
			Returns: "Single-character string at the given index",
			Examples: []Example{
				{Code: `charAt("hello", 1)`, Description: `Returns "e"`},
			},
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"string", "index", "character", "unicode"},
			Category:  "string",
			SeeAlso:   []string{"_str_foldChars", "_str_chars", "_str_len"},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_charAt: %v", err))
	}
}

// Type: (string, int) -> string
func makeStrCharAtType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.Int()).Returns(T.String()).Build()
}

func strCharAtImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	str, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_str_charAt: expected string for first argument: %w", err)
	}
	idx, err := SafeAsInt(args[1])
	if err != nil {
		return nil, fmt.Errorf("_str_charAt: expected int for second argument: %w", err)
	}

	runes := []rune(str)
	if idx < 0 || idx >= len(runes) {
		return nil, fmt.Errorf("_str_charAt: index %d out of bounds for string of length %d", idx, len(runes))
	}
	return &eval.StringValue{Value: string(runes[idx])}, nil
}

// ============================================================================
// _str_charCode: Convert single-character string to Unicode code point (int)
// ============================================================================

func registerStrCharCode() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_charCode",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrCharCodeType,
		Impl:    strCharCodeImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get Unicode code point of a single-character string",
			LongDesc:    "Returns the integer Unicode code point (0–1114111) of a single-character string. Errors if the string is empty or has more than one character. For ASCII, this is the byte value (e.g., charCode(\"a\") = 97).",
			Params: []ParamDoc{
				{Name: "c", Description: "A single-character string"},
			},
			Returns:   "Integer code point",
			Examples:  []Example{{Code: `charCode("a")`, Description: "Returns 97"}},
			Since:     "v0.11.0",
			Stability: StabilityStable,
			Tags:      []string{"string", "character", "unicode", "ord", "codepoint"},
			Category:  "string",
			SeeAlso:   []string{"_str_charAt", "_str_foldChars"},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_charCode: %v", err))
	}
}

func makeStrCharCodeType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.Int()).Build()
}

func strCharCodeImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	str, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_str_charCode: expected string argument: %w", err)
	}
	runes := []rune(str)
	if len(runes) != 1 {
		return nil, fmt.Errorf("_str_charCode: expected single-character string, got %d characters", len(runes))
	}
	return &eval.IntValue{Value: int(runes[0])}, nil
}
