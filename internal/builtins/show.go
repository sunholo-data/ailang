package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	registerShow()
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
			LongDesc: `Polymorphic function that converts values to canonical string form.
Works with all types: primitives (int, float, bool, string), lists, records, and ADTs.
Handles special float values (NaN, Inf) and limits depth for recursive structures.`,
			Params: []ParamDoc{
				{Name: "value", Description: "Any value to convert to string"},
			},
			Returns: "String representation of the value",
			Examples: []Example{
				{Code: `show(42)`, Description: `Returns "42"`},
				{Code: `show(3.14)`, Description: `Returns "3.14"`},
				{Code: `show(true)`, Description: `Returns "true"`},
				{Code: `show([1, 2, 3])`, Description: `Returns "[1, 2, 3]"`},
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
	val := args[0]
	return &eval.StringValue{Value: showValue(val, 0)}, nil
}

// showValue renders v with the one shared show renderer (eval.Show), also
// used by the bytecode VM and the REPL. depth > 0 renders v as if nested.
func showValue(v eval.Value, depth int) string {
	return eval.ShowAt(v, depth)
}

// maxWidth is the elision column, kept for this package's tests.
const maxWidth = eval.ShowMaxWidth
