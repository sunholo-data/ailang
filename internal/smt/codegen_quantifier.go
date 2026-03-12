package smt

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
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

	body, err := EncodeExpr(f.Body)
	if err != nil {
		return "", fmt.Errorf("forall body: %w", err)
	}

	// (forall ((i Int)) (=> (and (>= i lo) (< i hi)) body))
	return fmt.Sprintf("(forall ((%s Int)) (=> (and (>= %s %s) (< %s %s)) %s))",
		f.Var, f.Var, lo, f.Var, hi, body), nil
}
