package smt

import (
	"fmt"
	"strings"
)

// DeclareParametricDatatype uses native Z3 binders. Monomorphic callers retain
// the exact existing declaration shape.
func DeclareParametricDatatype(name string, params []string, variants []ADTVariant) string {
	base := DeclareDatatype(name, variants)
	if len(params) == 0 {
		return base
	}
	_, body := splitDeclareDatatype(base)
	return fmt.Sprintf("(declare-datatype %s (par (%s) %s))", name, strings.Join(params, " "), body)
}
