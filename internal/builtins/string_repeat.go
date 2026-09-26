package builtins

import (
	"fmt"
	"math"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// #1318: std/string.repeat built its parts list with `s :: _repeat_list(s, n - 1)`
// — one AILANG frame per copy and quadratic cons allocation — while its comment
// claimed an O(n) single allocation. repeat("x", 25000) failed RT_REC_003 at the
// default limit. _str_repeat had a Go codegen helper only; this is the
// interpreter implementation std/string.repeat now delegates to.

func init() {
	registerStrRepeat()
}

func registerStrRepeat() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_repeat",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeStrRepeatType,
		Impl:    strRepeatImpl,
		Metadata: &BuiltinMetadata{
			Description: "Repeat a string n times",
			LongDesc:    "Single allocation via strings.Repeat. n <= 0 returns the empty string.",
			Params: []ParamDoc{
				{Name: "s", Description: "String to repeat"},
				{Name: "n", Description: "Number of copies"},
			},
			Returns: "s concatenated n times",
			Examples: []Example{
				{Code: `_str_repeat("ab", 3)`, Description: `Returns "ababab"`},
				{Code: `_str_repeat("x", 0)`, Description: `Returns ""`},
			},
			Since:     "v0.44.2",
			Stability: StabilityStable,
			Tags:      []string{"string", "repeat"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_repeat: %v", err))
	}
}

// Type: (string, int) -> string
func makeStrRepeatType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.Int()).Returns(T.String()).Build()
}

func strRepeatImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	s, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_str_repeat: arg 0 (s) - %w", err)
	}
	n, err := SafeAsInt(args[1])
	if err != nil {
		return nil, fmt.Errorf("_str_repeat: arg 1 (n) - %w", err)
	}
	if n <= 0 || s == "" {
		return &eval.StringValue{Value: ""}, nil
	}
	// strings.Repeat panics when len(s)*n overflows; report it as an error.
	if n > math.MaxInt/len(s) {
		return nil, fmt.Errorf("_str_repeat: result length overflows (len %d x %d)", len(s), n)
	}
	return &eval.StringValue{Value: strings.Repeat(s, n)}, nil
}
