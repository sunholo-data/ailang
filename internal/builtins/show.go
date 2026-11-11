package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	registerShow()
	registerShowPreview()
}

func registerShow() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "show",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeShowType,
		Impl:    showImpl,

		Metadata: &BuiltinMetadata{
			Description: "Convert any value to its string representation",
			LongDesc: `Polymorphic function that converts any value to its canonical string form.
Total: every element and every level is rendered; nothing is elided.
A top-level string is returned unchanged; a string nested inside a list, tuple,
record, map or constructor is double-quoted and escaped, so different values
never print the same text. Floats print shortest round-trip digits (NaN, Inf,
-Inf, -0.0). For bounded output use std/string.showPreview(value, maxChars).`,
			Params: []ParamDoc{
				{Name: "value", Description: "Any value to convert to string"},
			},
			Returns: "String representation of the value",
			Examples: []Example{
				{Code: `show(42)`, Description: `Returns "42"`},
				{Code: `show(3.14)`, Description: `Returns "3.14"`},
				{Code: `show(true)`, Description: `Returns "true"`},
				{Code: `show("hi")`, Description: `Returns "hi" (a top-level string is unchanged)`},
				{Code: `show([1, 2, 3])`, Description: `Returns "[1, 2, 3]"`},
				{Code: `show(["a", "b, c"])`, Description: `Returns "[\"a\", \"b, c\"]" (nested strings are quoted)`},
				{Code: `show({x: 5, y: 10})`, Description: `Returns "{x: 5, y: 10}"`},
			},
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      []string{"conversion", "string", "debug", "display", "polymorphic"},
			Category:  "conversion",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register show: %v", err))
	}
}

func makeShowType() types.Type {
	// show : ∀α. α -> string
	// For now, we use a type variable directly which will be generalized
	// by the type system. This is similar to how v0.3.9 worked.
	alpha := &types.TVar2{Name: "α", Kind: types.Star}
	return &types.TFunc2{
		Params:    []types.Type{alpha},
		Return:    types.TString,
		EffectRow: types.EmptyEffectRow(),
	}
}

func showImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	return &eval.StringValue{Value: eval.Show(args[0])}, nil
}

// registerShowPreview registers _show_preview, the builtin behind
// std/string.showPreview (M-V1-SHOW-TOTAL M3).
func registerShowPreview() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_show_preview",
		NumArgs: 2,
		IsPure:  true,
		Type:    makeShowPreviewType,
		Impl:    showPreviewImpl,
		Metadata: &BuiltinMetadata{
			Description: "Bounded show: the first maxChars code points of show(value)",
			LongDesc: `Renders value as show does, except that a string is quoted even at the
top level, so a cut preview never reads as complete. If that rendering is at
most maxChars code points it is returned whole; otherwise the first maxChars
code points are followed by "…(+N chars)", N being the exact number left out.
Never builds more than maxChars of output. maxChars <= 0 keeps only the marker.`,
			Params: []ParamDoc{
				{Name: "value", Description: "Any value"},
				{Name: "maxChars", Description: "Maximum code points of show(value) to keep"},
			},
			Returns: "show(value), cut head-only with a count marker",
			Examples: []Example{
				{Code: `_show_preview([1, 2, 3], 100)`, Description: `Returns "[1, 2, 3]"`},
				{Code: `_show_preview([1, 2, 3], 5)`, Description: `Returns "[1, 2…(+4 chars)"`},
				{Code: `_show_preview("abc", 10)`, Description: `Returns "\"abc\"" (strings are quoted in a preview)`},
			},
			Since:     "v0.61.0",
			Stability: StabilityStable,
			Tags:      []string{"conversion", "string", "debug", "display", "polymorphic"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _show_preview: %v", err))
	}
}

// _show_preview : ∀α. (α, int) -> string
func makeShowPreviewType() types.Type {
	alpha := &types.TVar2{Name: "α", Kind: types.Star}
	return &types.TFunc2{
		Params:    []types.Type{alpha, types.TInt},
		Return:    types.TString,
		EffectRow: types.EmptyEffectRow(),
	}
}

func showPreviewImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	n, err := SafeAsInt(args[1])
	if err != nil {
		return nil, fmt.Errorf("_show_preview: arg 1 (maxChars) - %w", err)
	}
	return &eval.StringValue{Value: eval.ShowPreview(args[0], n)}, nil
}
