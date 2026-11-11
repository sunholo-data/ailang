package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	registerUnreachable()
}

// unreachable(msg) — M-V1-MATCH-EXHAUSTIVENESS §8. A `match` must cover every
// value (TC_MATCH_002); for a case the author knows cannot happen, the arm's
// body is `unreachable("why")` instead of a dummy value, so a broken
// invariant stops the program loudly (RT010) rather than flowing on as a
// wrong answer. It is pure and its result type is any type: it never returns.
func registerUnreachable() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "unreachable",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeUnreachableType,
		Impl:    unreachableImpl,

		Metadata: &BuiltinMetadata{
			Description: "Stop with RT010: a case the author has ruled out was reached",
			LongDesc: `unreachable(msg) never returns: evaluating it stops the program with
RT010 "unreachable code reached: <msg>" on every engine. Its result type is any
type, so it fits any match arm. Use it for a match case that cannot happen, so
the match is exhaustive (TC_MATCH_002) without inventing a dummy value.`,
			Params: []ParamDoc{
				{Name: "msg", Description: "Why the case cannot happen"},
			},
			Returns: "Never returns",
			Examples: []Example{
				{Code: `match xs { [] => unreachable("caller checked non-empty"), [h, ...t] => h }`, Description: "An impossible match case"},
			},
			Since:     "v0.61.0",
			Stability: StabilityStable,
			Tags:      []string{"match", "exhaustiveness", "invariant", "abort"},
			Category:  "control",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register unreachable: %v", err))
	}
}

func makeUnreachableType() types.Type {
	// unreachable : ∀α. string -> α
	alpha := &types.TVar2{Name: "α", Kind: types.Star}
	return &types.TFunc2{
		Params:    []types.Type{types.TString},
		Return:    alpha,
		EffectRow: types.EmptyEffectRow(),
	}
}

func unreachableImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	msg, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("unreachable: expected a string, got %s", args[0].Type())
	}
	return nil, &ailerrors.UnreachableError{Msg: msg.Value}
}

// registerUnreachableCodegenSpec: generated Go panics with the RT010 text the
// interpreters raise.
func registerUnreachableCodegenSpec() {
	registerIfMissing("unreachable", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Unreachable",
			Signature: "func Unreachable(msg interface{}) interface{}",
			Body:      `panic("RT010: unreachable code reached: " + msg.(string))`,
		},
	})
}
