package types

import (
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/typedast"
)

// #2273 for code checked against a declaration's parameters outside its body:
// a `requires`/`ensures` clause or a test (PR #2283). Such code sees the
// parameters at the declaration's rigid types, so it may not specialise them
// either: `ident[a](x: a)` with `requires { x + 1 > 0 }` is TC_RIGID_001.
//
// Use: sig := tc.RigidSignatureOf(fn); bind each parameter at
// sig.Instantiate(scheme)'s parameter types; then InferWithRigid instead of
// InferWithConstraints.

// RigidSignatureOf returns the rigid signature of the top-level declaration
// named fn in the module being checked, or nil.
func (tc *CoreTypeChecker) RigidSignatureOf(fn string) *RigidSig {
	for _, sig := range tc.rigidSigs {
		if sig != nil && sig.Func == fn {
			return sig
		}
	}
	return nil
}

// Instantiate returns scheme's type with the declaration's signature
// variables written as their rigid names (the scheme in the environment uses
// the written names, displayRigidScheme). Other variables are left as they are.
func (sig *RigidSig) Instantiate(scheme *Scheme) Type {
	if sig == nil || scheme == nil {
		return nil
	}
	quantified := make(map[string]bool, len(scheme.TypeVars))
	for _, v := range scheme.TypeVars {
		quantified[v] = true
	}
	rename := map[string]Type{}
	for _, v := range sig.Vars {
		if v.Row {
			continue
		}
		if quantified[v.Display] {
			rename[v.Display] = &TVar2{Name: v.Name, Kind: Star}
		}
	}
	return scheme.Type.Substitute(rename)
}

// InferWithRigid is InferWithConstraints for an expression that must keep the
// variables of sig general (TC_RIGID_001 otherwise).
func (tc *CoreTypeChecker) InferWithRigid(expr core.CoreExpr, env *TypeEnv, sig *RigidSig) (typedast.TypedNode, *TypeEnv, Type, []Constraint, error) {
	prev := tc.extraRigid
	tc.extraRigid = sig
	defer func() { tc.extraRigid = prev }()
	return tc.InferWithConstraints(expr, env)
}

// FreeTypeVarNames returns the type variables (not rows) occurring in t, in
// first-appearance order, by the same walk generalization uses.
func FreeTypeVarNames(t Type) []string {
	vars, _ := generalizableVars(t)
	return vars
}

// markDefaulted records the nodes of the current declaration whose type
// top-level defaulting is about to pin (a `Num a` variable := int). A generic
// body — a signature's rigid `a`, a let-bound lambda's own variable, an
// unannotated function's parameter — is compiled once, at that defaulted type.
// The class methods are corrected from the run-time values (evalDictApp's
// M-POLY-ARITH), but a consumer that picks an implementation from the recorded
// type is not: ShowNormalizer turned `show(x * y)` into intToStr, and a float
// call crashed (#2273 review: `label[a](x: a, y: a) -> string`). The consumers
// ask DefaultedNodes and stay dynamic there (ShowNormalizer.Generic).
func (tc *CoreTypeChecker) markDefaulted(sub, defaulting Substitution) {
	nodes := tc.declNodes
	tc.declNodes = tc.declNodes[:0]
	if len(defaulting) == 0 {
		return
	}
	for _, id := range nodes {
		t, ok := tc.CoreTI[id]
		if !ok {
			continue
		}
		for _, v := range FreeTypeVarNames(ApplySubstitution(sub, t)) {
			if defaulting[v] != nil {
				if tc.defaulted == nil {
					tc.defaulted = make(map[uint64]bool)
				}
				tc.defaulted[id] = true
				break
			}
		}
	}
}

// setCoreTI records a node's inferred type, and the node as one this
// declaration typed (markDefaulted).
func (tc *CoreTypeChecker) setCoreTI(id uint64, t Type) {
	tc.CoreTI.Set(id, t)
	if tc.declNodes != nil {
		tc.declNodes = append(tc.declNodes, id)
	}
}

// DefaultedNodes is the set markDefaulted built: nodes whose recorded type is
// a default standing in for a type variable of a generic body.
func (tc *CoreTypeChecker) DefaultedNodes() map[uint64]bool { return tc.defaulted }

// InferPredicateRigid is InferPredicate (a contract clause, typed bool) for a
// clause over a declaration's parameters bound at its rigid types (sig): the
// clause may not specialise them (#2273 for #2232/#2248's contract check).
// A nil sig is plain InferPredicate.
func (tc *CoreTypeChecker) InferPredicateRigid(expr core.CoreExpr, env *TypeEnv, sig *RigidSig) (Type, error) {
	prev := tc.extraRigid
	tc.extraRigid = sig
	defer func() { tc.extraRigid = prev }()
	return tc.InferPredicate(expr, env)
}

// pushRigidGroup records letrec's binding names as the group being inferred
// and returns the function that restores the previous group.
func (ctx *InferenceContext) pushRigidGroup(letrec *core.LetRec) func() {
	prev := ctx.rigidGroup
	ctx.rigidGroup = make(map[string]bool, len(letrec.Bindings))
	for _, b := range letrec.Bindings {
		ctx.rigidGroup[b.Name] = true
	}
	return func() { ctx.rigidGroup = prev }
}

// envOwnedVars is the type variables env's bindings above the declaration's
// base environment own, under sub: exactly the variables generalization
// withholds (generalizeWithConstraints), so defaulting at an inner `let` leaves
// them to the enclosing binding. Empty for a top-level binding (env is the base).
func (ctx *InferenceContext) envOwnedVars(sub Substitution, env *TypeEnv) map[string]bool {
	if env == nil || ctx.baseEnvFreeVars == nil {
		return nil
	}
	owned := make(map[string]bool)
	for v := range env.ownedVars().types {
		if !ctx.baseEnvFreeVars[v] {
			owned[v] = true
		}
	}
	for v := range env.FreeTypeVarsAbove(ctx.baseEnv) {
		owned[v] = true
	}
	if len(owned) == 0 {
		return nil
	}
	return varsUnderSubstitution(sub, owned)
}

// solveDeclConstraintsRigid is solveDeclConstraints followed by the
// InferWithRigid check (a no-op when no signature is set).
func (tc *CoreTypeChecker) solveDeclConstraintsRigid(ctx *InferenceContext) (Substitution, []ClassConstraint, error) {
	sub, unsolved, err := ctx.solveDeclConstraints()
	if err != nil {
		return nil, nil, err
	}
	if err := ctx.checkExtraRigid(tc.extraRigid, sub, unsolved); err != nil {
		return nil, nil, err
	}
	return sub, unsolved, nil
}

// checkExtraRigid enforces sig (when set) on a solved expression: its
// variables are free in the environment, so only "still a distinct
// variable, no literal" is asked, not "quantified".
func (ctx *InferenceContext) checkExtraRigid(sig *RigidSig, sub Substitution, unsolved []ClassConstraint) error {
	if sig == nil {
		return nil
	}
	v := findRigidViolation(sig, nil, unsolved, func(t Type) Type { return ApplySubstitution(sub, t) })
	if v == nil {
		return nil
	}
	if v.kind == "concrete" || v.kind == "equal" {
		v.at = ctx.rigidCulprit(sig, v, false)
	}
	v.clause = true
	return v.toError(sig)
}
