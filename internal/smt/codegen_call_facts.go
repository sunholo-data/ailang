package smt

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/core"
)

// Finite facts belong to the expression that evaluates a call. They must never
// escape a let binder, a selected branch, or a helper's formal parameters.
var activeCallFacts *[]string
var activeContractSpecs map[string]*ContractSpec
var expandingContractSpecs map[string]bool

func captureCallFacts(encode func() (string, error)) (string, string, error) {
	previous := activeCallFacts
	var facts []string
	activeCallFacts = &facts
	defer func() { activeCallFacts = previous }()
	value, err := encode()
	return value, conjoinFacts(facts...), err
}

func addCallFact(fact string) {
	if activeCallFacts != nil && fact != "true" && fact != "" {
		*activeCallFacts = append(*activeCallFacts, fact)
	}
}

func encodeContractApplication(name string, args []core.CoreExpr) (string, error) {
	spec := activeContractSpecs[name]
	if spec == nil {
		return encodeUserFunctionCall(activeContractCallees[name], args)
	}
	subs, err := buildParamSubstitution(spec.Params, args)
	if err != nil {
		return "", err
	}
	// Reuse the already encoded operands, avoiding duplicate nested-call expansion.
	var encodedArgs []core.CoreExpr
	for i, p := range spec.Params {
		if isUnitLiteral(args[i]) {
			encodedArgs = append(encodedArgs, args[i])
			continue
		}
		encodedArgs = append(encodedArgs, &core.VarGlobal{Ref: core.GlobalRef{Module: "$encoded", Name: subs[p.Name]}})
	}
	value, err := encodeUserFunctionCall(contractSymbol(*spec), encodedArgs)
	if err != nil {
		return "", err
	}
	if expandingContractSpecs == nil {
		expandingContractSpecs = make(map[string]bool)
	}
	if !expandingContractSpecs[name] {
		expandingContractSpecs[name] = true
		fact := instantiateContract(*spec, subs, value)
		delete(expandingContractSpecs, name)
		addCallFact(fact)
	}
	return value, nil
}

func encodeFactIf(e *core.If) (string, error) {
	return encodeFactIfHint(e, "")
}

func encodeFactIfHint(e *core.If, hint string) (string, error) {
	condition, err := EncodeExpr(e.Cond)
	if err != nil {
		return "", err
	}
	yes, yfacts, err := captureCallFacts(func() (string, error) { return encodeExprWithSortHint(e.Then, hint) })
	if err != nil {
		return "", err
	}
	no, nfacts, err := captureCallFacts(func() (string, error) { return encodeExprWithSortHint(e.Else, hint) })
	if err != nil {
		return "", err
	}
	if yfacts != "true" || nfacts != "true" {
		addCallFact(fmt.Sprintf("(ite %s %s %s)", condition, yfacts, nfacts))
	}
	return fmt.Sprintf("(ite %s %s %s)", condition, yes, no), nil
}

func encodeFactLet(e *core.Let) (string, error) {
	value, err := EncodeExpr(e.Value)
	if err != nil {
		return "", err
	}
	body, facts, err := captureCallFacts(func() (string, error) { return encodeLetScoped(e.Body, e.Name) })
	if err != nil {
		return "", err
	}
	if facts != "true" {
		addCallFact(fmt.Sprintf("(let ((%s %s)) %s)", e.Name, value, facts))
	}
	return fmt.Sprintf("(let ((%s %s)) %s)", e.Name, value, body), nil
}
