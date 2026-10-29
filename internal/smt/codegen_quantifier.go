package smt

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/sunholo-data/ailang/internal/core"
)

// encodeForall encodes a bounded universal quantifier as SMT-LIB.
//
// The encoding follows the standard bounded quantifier pattern:
//
//	forall i: lo..hi => body
//	→ (forall ((i Int)) (=> (and (>= i lo) (< i hi)) body))
//
// This is a standard Z3-supported encoding where the quantifier ranges
// over all integers, but the guard restricts it to the bounded range [lo, hi).
func encodeForall(f *core.Forall) (string, error) {
	lo, err := EncodeExpr(f.Lo)
	if err != nil {
		return "", fmt.Errorf("forall lower bound: %w", err)
	}

	hi, err := EncodeExpr(f.Hi)
	if err != nil {
		return "", fmt.Errorf("forall upper bound: %w", err)
	}

	// Bounds are evaluated in the outer lexical environment. A same-named
	// quantified binder must not capture their already encoded expressions.
	binder := f.Var
	tokens := strings.FieldsFunc(lo+" "+hi, func(r rune) bool { return unicode.IsSpace(r) || r == '(' || r == ')' })
	for _, token := range tokens {
		if token == binder {
			occupied := f.Body.String() + lo + hi
			for _, reserved := range activeParamRenames {
				occupied += reserved
			}
			for i := 0; ; i++ {
				candidate := fmt.Sprintf("$forall_bound_%d", i)
				if !strings.Contains(occupied, candidate) {
					binder = candidate
					break
				}
			}
			break
		}
	}
	restore := withoutParameterRenames([]string{f.Var})
	defer restore()
	if binder != f.Var {
		activeParamRenames[f.Var] = binder
	}
	body, facts, err := captureCallFacts(func() (string, error) { return EncodeExpr(f.Body) })
	if err != nil {
		return "", fmt.Errorf("forall body: %w", err)
	}

	if facts != "true" {
		addCallFact(fmt.Sprintf("(forall ((%s Int)) (=> (and (>= %s %s) (< %s %s)) %s))", binder, binder, lo, binder, hi, facts))
	}

	// (forall ((i Int)) (=> (and (>= i lo) (< i hi)) body))
	return fmt.Sprintf("(forall ((%s Int)) (=> (and (>= %s %s) (< %s %s)) %s))",
		binder, binder, lo, binder, hi, body), nil
}
