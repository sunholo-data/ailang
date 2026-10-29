package smt

import "github.com/sunholo-data/ailang/internal/core"

var activeHelperConditions map[string]string

// The condition helper has the exact value helper signature, including any
// already-lifted environment parameters. Encode each actual once and share it
// between value and condition calls; never merge distinct actual environments.
func encodeResolvedApplication(name string, args []core.CoreExpr) (string, error) {
	var operands []core.CoreExpr
	for _, arg := range args {
		if isUnitLiteral(arg) {
			operands = append(operands, arg)
			continue
		}
		value, err := EncodeExpr(arg)
		if err != nil {
			return "", err
		}
		operands = append(operands, encodedCore(value))
	}
	value, err := encodeUserFunctionCall(name, operands)
	if err != nil {
		return "", err
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
