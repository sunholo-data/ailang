package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// Opaque helpers (M-VERIFY-SOUNDNESS-COVERAGE, ruled by Mark 2026-11-25).
//
// A helper whose body cannot be encoded (a builtin with no SMT form, a tuple,
// a call cycle…) and that has no ensures used to skip every function calling
// it, although most contracts do not depend on its value: a renderer's
// `length(result) > 0` or a flush's `listLength(result.textLines) == 0`.
// Such a helper is now an uninterpreted function of its signature:
//
//   - Sound for proofs: the helper's result is any value of its sort, which
//     over-approximates what the real body returns.
//   - A model through it is inconclusive, never a counterexample (its symbol is
//     in EncodeResult.Uninterpreted); witness replay runs the real function.
//   - Its requires is still checked at each call (contractCallObligation).
//   - Its OWN obligations (requires of the calls inside it, overflow, NaN) are
//     not checked, since its body is not encoded: a VERIFIED result that
//     reaches one carries the CaveatOpaqueHelpers caveat naming it and why.
//
// A helper is opaque only when its signature has an SMT sort: the surface
// return type is known and every parameter maps. Otherwise it still fails.

// CaveatOpaqueHelpers marks a VERIFIED result reached through opaque helpers.
const CaveatOpaqueHelpers = "helpers_not_checked"

// OpaqueHelper is a helper called as an uninterpreted function, and why.
type OpaqueHelper struct {
	Name   string
	Reason string
}

// appendSpecOrOpaque stands in for a callee that is not defined: by its
// contract when it has an ensures (as for any unencodable callee, and the
// induction hypothesis when the callee is the verified function reached back
// through a helper), else as an opaque helper. It records why the callee was
// not defined when neither works (the reason a caller that reaches it reports).
func appendSpecOrOpaque(defs []CalleeDef, name string, meta *core.DeclMeta, params []FunctionParam, returnSort, reason string) []CalleeDef {
	if spec := BuildContractSpec(name, meta, params, returnSort); returnSort != "" && len(spec.Ensures) > 0 {
		declaration, err := declareContractFunction(spec)
		if err != nil {
			recordCalleeFailure(name, reason)
			return defs
		}
		activeContractSpecs[name] = &spec
		activeContractCallees[name] = contractSymbol(spec)
		return append(defs, CalleeDef{Name: name, SMTLib: declaration, IsContract: true, ResultConst: contractSymbol(spec), Contract: &spec})
	}
	def, err := declareOpaqueHelper(name, meta, params, returnSort, reason)
	if err != nil {
		recordCalleeFailure(name, reason)
		return defs
	}
	return append(defs, def)
}

// declareOpaqueHelper is the contract path with no ensures: a typed UF and the
// requires template (checked at each call), with no fact about the result.
func declareOpaqueHelper(name string, meta *core.DeclMeta, params []FunctionParam, returnSort, reason string) (CalleeDef, error) {
	if returnSort == "" {
		return CalleeDef{}, fmt.Errorf("%w: %s has no known return sort", ErrUnresolvableTypes, name)
	}
	for _, p := range params {
		if p.Type == nil && !isUnitFunctionParam(p) {
			return CalleeDef{}, fmt.Errorf("%w: parameter %q of %s has no type", ErrUnresolvableTypes, p.Name, name)
		}
	}
	spec := BuildContractSpec(name, meta, params, returnSort)
	spec.Ensures = nil
	declaration, err := declareContractFunction(spec)
	if err != nil {
		return CalleeDef{}, err
	}
	activeContractSpecs[name] = &spec
	activeContractCallees[name] = contractSymbol(spec)
	if reason == "" {
		reason = "its body could not be encoded"
	}
	return CalleeDef{
		Name:        name,
		SMTLib:      declaration,
		IsContract:  true,
		ResultConst: contractSymbol(spec),
		Contract:    &spec,
		Opaque:      reason,
	}, nil
}

// opaqueSymbols are the UF symbols of the opaque helpers.
func opaqueSymbols(hs []OpaqueHelper) []string {
	out := make([]string, len(hs))
	for i, h := range hs {
		out[i] = contractSymbol(ContractSpec{FuncName: h.Name})
	}
	return out
}

// opaqueHelpersCaveat names, on a VERIFIED result, the helpers whose bodies
// were not encoded: the proof holds for every value they could return, but
// the obligations inside them were not checked.
func opaqueHelpersCaveat(vr VerifyResult, hs []OpaqueHelper) VerifyResult {
	if vr.Status != "verified" || len(hs) == 0 {
		return vr
	}
	parts := make([]string, len(hs))
	for i, h := range hs {
		parts[i] = fmt.Sprintf("%s (%s)", h.Name, h.Reason)
	}
	vr.Caveats = append(vr.Caveats, Caveat{
		Kind:   CaveatOpaqueHelpers,
		Detail: "proved for every value these helpers could return; their bodies were not encoded, so the checks inside them (callee requires, overflow, NaN) were not made: " + strings.Join(parts, "; "),
	})
	return vr
}
