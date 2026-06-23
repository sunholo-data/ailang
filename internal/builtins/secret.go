package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Secret effect builtins for AILANG (M-SECRET-EFFECT, v0.26.0).
//
// _secret_read resolves an op:// reference to its plaintext value, gated behind
// a (possibly remote, human) approval and the Secret capability. The surface
// `secret(ref)` in std/secret wraps this builtin.

func init() {
	registerSecret()
}

func registerSecret() {
	// _secret_read — routes through effects.Call() for capability check, the
	// approval gate, trace recording, and (M5) the <secret> label on the result.
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "Secret", "read", args)
	}
	typ := func() types.Type {
		T := types.NewBuilder()
		// Return type carries the <secret> IFC label so the value can be tracked
		// by M-TAINT-TYPES. NOTE: label *enforcement* (sink/declassify checks)
		// is not yet wired into the type checker (M-TAINT-TYPES Phase 2); the
		// label propagates and is observable, and Z3 contracts provide today's
		// leak guarantee. See design_docs/.../m-secret-effect-remote-approval.md.
		secretString := types.WithLabel(T.String(), types.LabelConst("secret"))
		return T.Func(T.String(), T.String()).Returns(secretString).Effects("Secret")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/secret", Name: "_secret_read", NumArgs: 2, IsPure: false, Effect: "Secret", Type: typ, Impl: impl,

		Metadata: &BuiltinMetadata{
			Description: "Resolve a secret reference (op://vault/item/field) to its value",
			LongDesc:    "Gated secret acquisition: blocks on a (possibly remote, human) approval, then resolves the reference just-in-time via the 1Password CLI. The reference is safe to log; the resolved value is labelled <secret> and redacted from traces. The purpose is the human-readable reason shown in the approval request.",
			Params: []ParamDoc{
				{Name: "ref", Description: "1Password secret reference, e.g. \"op://Prod/stripe/api-key\""},
				{Name: "purpose", Description: "Human-readable reason shown in the approval request, e.g. \"charge order #1234\""},
			},
			Returns: "The resolved secret value (labelled <secret>)",
			Examples: []Example{
				{Code: `_secret_read("op://Prod/stripe/api-key", "charge order #1234")`, Description: "Resolve after approval"},
			},
			Since:     "v0.26.0",
			Stability: StabilityExperimental,
			Tags:      []string{"secret", "credential", "1password", "approval", "security"},
			Category:  "secret",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _secret_read: %v", err))
	}
}
