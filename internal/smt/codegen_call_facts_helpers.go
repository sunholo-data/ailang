package smt

import "github.com/sunholo-data/ailang/internal/core"

var activeHelperConditions map[string]string

// The condition helper has the exact value helper signature, including any
// already-lifted environment parameters. Encode each actual once and share it
// between value and condition calls; never merge distinct actual environments.
func encodeResolvedApplication(name string, args []core.CoreExpr) (string, error) {
	var operands []core.CoreExpr
	sorts := activeUserParamSorts[name]
	for i, arg := range args {
		if isUnitLiteral(arg) {
			operands = append(operands, arg)
			continue
		}
		hint := "" // `f([])`: the parameter's list sort, not (Seq Int)
		if len(sorts) == len(args) {
			hint = sorts[i]
		}
		value, err := encodeExprWithSortHint(arg, hint)
		if err != nil {
			return "", err
		}
		operands = append(operands, encodedCore(value))
	}
	value, err := encodeUserFunctionCall(name, operands)
	if err != nil {
		return "", err
	}
	if obligationsOn {
		if why, bad := activeUncheckableRequires[name]; bad {
			return "", uncheckableCall(name, why)
		}
		if req := activeCalleeRequires[name]; req != "" {
			obligation, err := encodeUserFunctionCall(req, operands)
			if err != nil {
				return "", err
			}
			addObligation(name, obligation)
		}
	}
	if condition := activeHelperConditions[name]; condition != "" {
		fact, err := encodeUserFunctionCall(condition, operands)
		if err != nil {
			return "", err
		}
		addCallFact(fact)
	}
	return value, nil
}
