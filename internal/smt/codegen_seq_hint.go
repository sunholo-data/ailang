package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// An empty list literal carries no element type of its own, but SMT-LIB is
// monomorphically sorted: `(as seq.empty (Seq X))` fixes X at the term, and Z3
// will NOT unify it with a differently-sorted sequence. Encoding every empty
// list as `(Seq Int)` therefore produces an ill-sorted term wherever the
// expected element type is anything but int, and Z3 rejects the whole query
// with a raw sort error (ailang#689).
//
// The expected sort is available at the points where a *declared* type is
// known — a record field's sort, a function's return sort — so it is threaded
// down from there as a hint. Where no hint is available the behaviour is
// unchanged, so this can only replace an ill-sorted `(Seq Int)` with the sort
// the declaration already committed to.

// seqElemSort returns the element sort of an SMT-LIB sequence sort:
// "(Seq String)" → "String", "(Seq (Seq String))" → "(Seq String)".
// The second result is false when sort is not a sequence sort.
func seqElemSort(sort string) (string, bool) {
	s := strings.TrimSpace(sort)
	if !strings.HasPrefix(s, "(Seq ") || !strings.HasSuffix(s, ")") {
		return "", false
	}
	elem := strings.TrimSpace(s[len("(Seq ") : len(s)-1])
	if elem == "" {
		return "", false
	}
	return elem, true
}

// encodeExprWithSortHint encodes expr exactly as EncodeExpr does, except that
// empty list literals reachable from expr are given expectedSort's element sort
// instead of the Int default. The hint propagates through the constructs that
// pass a value through unchanged (if-branches, let bodies) and into list
// elements; every other expression is delegated to EncodeExpr untouched.
//
// An empty hint, or a hint that is not a sequence sort, reproduces EncodeExpr's
// behaviour exactly.
func encodeExprWithSortHint(expr core.CoreExpr, expectedSort string) (string, error) {
	if expr == nil {
		return "", fmt.Errorf("nil expression")
	}
	if expectedSort == "" {
		return EncodeExpr(expr)
	}

	// ANF hoists every literal into a temporary, so an empty list literal is
	// never syntactically AT the site whose declared type would give it an
	// element sort. Put it back there first.
	expr = inlineEmptyListBindings(expr)

	switch e := expr.(type) {
	case *core.List:
		return encodeListWithSortHint(e, expectedSort)

	case *core.If:
		// Both branches produce the function's value, so both inherit the hint.
		cond, err := EncodeExpr(e.Cond)
		if err != nil {
			return "", fmt.Errorf("if condition: %w", err)
		}
		then, err := encodeExprWithSortHint(e.Then, expectedSort)
		if err != nil {
			return "", fmt.Errorf("if then: %w", err)
		}
		els, err := encodeExprWithSortHint(e.Else, expectedSort)
		if err != nil {
			return "", fmt.Errorf("if else: %w", err)
		}
		return fmt.Sprintf("(ite %s %s %s)", cond, then, els), nil

	default:
		return EncodeExpr(expr)
	}
}

// encodeListWithSortHint encodes a list literal, using expectedSort to resolve
// the element sort of an empty literal and to propagate into nested elements.
func encodeListWithSortHint(list *core.List, expectedSort string) (string, error) {
	elemSort, ok := seqElemSort(expectedSort)
	if !ok {
		// Not a sequence sort — no usable hint, so behave exactly as before.
		return encodeList(list)
	}

	if len(list.Elements) == 0 {
		return fmt.Sprintf("(as seq.empty (Seq %s))", elemSort), nil
	}

	// Non-empty: element sorts are inferable from the elements themselves, but
	// an element may ITSELF be an empty list (e.g. [[]] at list[list[string]]),
	// so the hint has to reach them too.
	encoded := make([]string, len(list.Elements))
	for i, elem := range list.Elements {
		enc, err := encodeExprWithSortHint(elem, elemSort)
		if err != nil {
			return "", fmt.Errorf("list element %d: %w", i, err)
		}
		encoded[i] = fmt.Sprintf("(seq.unit %s)", enc)
	}
	if len(encoded) == 1 {
		return encoded[0], nil
	}
	return fmt.Sprintf("(seq.++ %s)", strings.Join(encoded, " ")), nil
}

// isEmptyListLiteral reports whether expr is a literal `[]`.
func isEmptyListLiteral(expr core.CoreExpr) bool {
	l, ok := expr.(*core.List)
	return ok && len(l.Elements) == 0
}

// inlineEmptyListBindings rewrites `let x = [] in body` to `body[x := []]`
// along the ANF let-spine.
//
// The rewrite is semantically neutral: an empty list literal is a pure,
// constant, argument-free value, so substituting it for its single binding
// changes no evaluation order and duplicates no work. What it buys is
// position — after inlining, the literal sits at the record field / cons /
// element position whose declared sort can supply its element type.
//
// Shadowing is handled by SubstituteLambdaVar, which stops at any binder that
// rebinds the name.
func inlineEmptyListBindings(expr core.CoreExpr) core.CoreExpr {
	let, ok := expr.(*core.Let)
	if !ok {
		return expr
	}
	body := inlineEmptyListBindings(let.Body)
	if isEmptyListLiteral(let.Value) {
		return SubstituteLambdaVar(body, let.Name, let.Value)
	}
	if body == let.Body {
		return expr
	}
	return &core.Let{Name: let.Name, Value: let.Value, Body: body}
}
