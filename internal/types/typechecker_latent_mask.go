package types

import (
	"github.com/sunholo-data/ailang/internal/core"
)

// LatentParamMask publishes callback openness BEFORE application unification.
// A present all-false mask differs from absent metadata (a compiler invariant failure).
// #616 can extend this single publication with the resolved call row.
func (tc *CoreTypeChecker) LatentParamMask(appID uint64) ([]bool, bool) {
	mask, ok := tc.latentParamMasks[appID]
	return append([]bool(nil), mask...), ok
}

func callbackMask(t Type) []bool {
	fn, ok := t.(*TFunc2)
	if !ok {
		return nil
	}
	mask := make([]bool, len(fn.Params))
	for i, param := range fn.Params {
		if callback, ok := param.(*TFunc2); ok {
			mask[i] = callback.EffectRow != nil && callback.EffectRow.Tail != nil && !callback.ConcreteEffectContract
		}
	}
	return mask
}

// Capture the scheme before instantiation; other callee expressions use their
// inferred type before arguments add application constraints.
func (tc *CoreTypeChecker) preApplicationMask(ctx *InferenceContext, expr core.CoreExpr, inferred Type) []bool {
	var binding interface{}
	switch e := expr.(type) {
	case *core.Var:
		binding, _ = ctx.env.Lookup(e.Name)
	case *core.VarGlobal:
		binding = tc.globalTypes[e.Ref.Module+"."+e.Ref.Name]
	}
	if scheme, ok := binding.(*Scheme); ok && scheme != nil {
		return tc.resolvedCallbackMask(scheme.Type)
	}
	if t, ok := binding.(Type); ok {
		return tc.resolvedCallbackMask(t)
	}
	return tc.resolvedCallbackMask(inferred)
}

func (tc *CoreTypeChecker) publishLatentMask(id uint64, mask []bool) {
	if tc.latentParamMasks == nil {
		tc.latentParamMasks = make(map[uint64][]bool)
	}
	tc.latentParamMasks[id] = append([]bool(nil), mask...)
}

// EffectValueType resolves alias heads for the post-inference effect validator.
// Structural CoreTI keeps aliases for lowering; a function alias must still
// expose its latent row when applied or passed as a callback.
func (tc *CoreTypeChecker) EffectValueType(id uint64) (Type, bool) {
	typ, ok := tc.CoreTI.Get(id)
	if !ok {
		return nil, false
	}
	return tc.effectAliasHead(typ), true
}

func (tc *CoreTypeChecker) effectAliasHead(typ Type) Type {
	u := NewUnifier()
	u.aliasEnv, u.aliasParams, u.aliasCaptures = tc.aliasEnv, tc.aliasParams, tc.aliasCaptures
	return u.expandAlias(typ)
}

func (tc *CoreTypeChecker) resolvedCallbackMask(typ Type) []bool {
	fn, ok := tc.effectAliasHead(typ).(*TFunc2)
	if !ok {
		return nil
	}
	resolved := *fn
	resolved.Params = make([]Type, len(fn.Params))
	for i, p := range fn.Params {
		resolved.Params[i] = tc.effectAliasHead(p)
	}
	return callbackMask(&resolved)
}
