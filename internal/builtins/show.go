package builtins

import (
	"fmt"
	"sort"
	"strconv"

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

// showValue renders an evaluator value through the shared show renderer
// (show_render.go), which the bytecode VM uses too. depth > 0 renders v as if
// nested that deep.
func showValue(v eval.Value, depth int) string {
	return renderShow(v, depth, inspectEvalShow)
}

func evalItems(vs []eval.Value) []any {
	out := make([]any, len(vs))
	for i, x := range vs {
		out[i] = x
	}
	return out
}

// inspectEvalShow describes an evaluator value for RenderShow.
func inspectEvalShow(x any) ShowNode {
	switch val := x.(type) {
	case *eval.IntValue:
		return ShowNode{Text: strconv.Itoa(val.Value)}
	case *eval.FloatValue:
		return ShowNode{Kind: ShowFloat, Float: val.Value}
	case *eval.BoolValue:
		return ShowNode{Text: strconv.FormatBool(val.Value)}
	case *eval.StringValue:
		return ShowNode{Text: val.Value} // identity for strings, no quotes
	case *eval.ListValue:
		return ShowNode{Kind: ShowList, Items: evalItems(val.Elements)}
	case *eval.ArrayValue:
		return ShowNode{Kind: ShowArray, Items: evalItems(val.Elements())}
	case *eval.TupleValue:
		return ShowNode{Kind: ShowTuple, Items: evalItems(val.Elements)}
	case *eval.MapValue:
		keys := make([]string, 0, len(val.Entries))
		for key := range val.Entries {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		items := make([]any, 0, 2*len(keys))
		for _, key := range keys {
			items = append(items, val.Entries[key].Key, val.Entries[key].Value)
		}
		return ShowNode{Kind: ShowMap, Items: items}
	case *eval.RecordValue:
		names := make([]string, 0, len(val.Fields))
		items := make([]any, 0, len(val.Fields))
		for k, fv := range val.Fields {
			names = append(names, k)
			items = append(items, fv)
		}
		return ShowNode{Kind: ShowRecord, Names: names, Items: items}
	case *eval.TaggedValue:
		return ShowNode{Kind: ShowCtor, Text: val.CtorName, Items: evalItems(val.Fields)}
	case *eval.UnitValue:
		return ShowNode{Text: "()"}
	case *eval.FunctionValue, *eval.BuiltinFunction, *eval.ConstructorClosure:
		return ShowNode{Text: "<function>"}
	case *eval.BytesValue:
		return ShowNode{Text: val.String()}
	case *eval.IndirectValue:
		if val.Cell == nil || !val.Cell.Init || val.Cell.Val == nil {
			return ShowNode{Text: "<uninitialized>"}
		}
		return ShowNode{Kind: ShowSame, Items: []any{val.Cell.Val}}
	case *eval.ErrorValue:
		return ShowNode{Text: "Error: " + val.Message}
	}
	return ShowNode{Text: "<unknown>"}
}
