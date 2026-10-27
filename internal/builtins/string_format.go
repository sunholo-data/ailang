package builtins

import (
	"fmt"
	"strconv"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

const formatFixedMaxDecimals = 1000

func init() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/string", Name: "_str_formatFixed", NumArgs: 2, IsPure: true,
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.Float(), T.Int()).Returns(T.String()).Build()
		},
		Impl: stringFormatFixedImpl,
		Metadata: &BuiltinMetadata{
			Description: "Format a float with a fixed number of decimal places",
			LongDesc: "Formats the exact float64 value with strconv.FormatFloat(x, 'f', decimals, 64). " +
				"Decimals must be in 0..1000; other precisions fail before formatting even without runtime contract checks. " +
				"NaN and infinities print as NaN, +Inf and -Inf; negative zero retains its sign. " +
				"Long precisions may expose binary approximation digits; this is not decimal arithmetic.",
			Params: []ParamDoc{
				{Name: "x", Description: "Float value to format"},
				{Name: "decimals", Description: "Number of decimal places, 0..1000 inclusive"},
			},
			Returns: "Fixed-decimal string, or a runtime error for an out-of-range precision",
			Examples: []Example{
				{Code: `formatFixed(42.0, 2)`, Description: `Returns "42.00"`},
				{Code: `formatFixed(0.125, 2)`, Description: `Returns "0.12" (nearest even)`},
			},
			Since: "v0.54.18", Stability: StabilityStable,
			Tags: []string{"string", "float", "format", "decimal"}, Category: "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_formatFixed: %v", err))
	}
}

func stringFormatFixedImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	x, ok := args[0].(*eval.FloatValue)
	if !ok {
		return nil, fmt.Errorf("formatFixed: expected float, got %s", ailangTypeName(args[0], "float"))
	}
	d, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("formatFixed: expected int, got %s", ailangTypeName(args[1], "int"))
	}
	// Do not rely on requires: contracts are optional, and strconv interprets
	// negative precision as shortest form rather than rejecting it.
	if d.Value < 0 || d.Value > formatFixedMaxDecimals {
		return nil, fmt.Errorf("formatFixed: decimals must be in 0..1000 (got %d)", d.Value)
	}
	return &eval.StringValue{Value: strconv.FormatFloat(x.Value, 'f', d.Value, 64)}, nil
}
