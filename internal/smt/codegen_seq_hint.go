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
		if activeCallFacts != nil {
			return encodeFactIfHint(e, expectedSort)
		}
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

	case *core.Let:
		// The let's body is the value, so it inherits the hint: an
		// `if room <= 0 then [] else take(…)` under a let otherwise typed its
		// [] as (Seq Int) whatever the declared element sort. Same binding
		// hygiene and call-facts scoping as encodeLet / encodeFactLet.
		value, err := encodeLetValue(e)
		if err != nil {
			return "", fmt.Errorf("let value: %w", err)
		}
		symbol := letSymbol(e.Name)
		encodeBody := func() (string, error) {
			previous := activeParamRenames
			local := make(map[string]string, len(previous)+1)
			for k, v := range previous {
				local[k] = v
			}
			local[e.Name] = symbol
			activeParamRenames = local
			defer func() { activeParamRenames = previous }()
			return encodeExprWithSortHint(e.Body, expectedSort)
		}
		if activeCallFacts == nil {
			body, err := encodeBody()
			if err != nil {
				return "", fmt.Errorf("let body: %w", err)
			}
			return fmt.Sprintf("(let ((%s %s)) %s)", symbol, value, body), nil
		}
		body, facts, err := captureCallFacts(encodeBody)
		if err != nil {
			return "", fmt.Errorf("let body: %w", err)
		}
		if facts != "true" {
			addCallFact(fmt.Sprintf("(let ((%s %s)) %s)", symbol, value, facts))
		}
		return fmt.Sprintf("(let ((%s %s)) %s)", symbol, value, body), nil

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

// encodeLetValue encodes a let-bound value. An empty list literal carries an
// unresolved element type (`[α]`) in Core's type info, so when ANF hoists it
// (`let $t = [] in … $other == $t …`) its sort is taken from its first binary
// use whose other operand has a known list type. Otherwise it is encoded as
// usual.
func encodeLetValue(let *core.Let) (string, error) {
	if isEmptyListLiteral(let.Value) {
		if sort := emptyListSortFromUse(let.Name, let.Body); sort != "" {
			return fmt.Sprintf("(as seq.empty %s)", sort), nil
		}
	}
	return EncodeExpr(let.Value)
}

// emptyListSortFromUse finds, in body, the first two-operand application with
// Var(name) as one operand and a list-typed other operand, and returns that
// list's SMT sort ("" when none is found).
func emptyListSortFromUse(name string, body core.CoreExpr) string {
	found := ""
	var walk func(e core.CoreExpr)
	sortOf := seqSortOf
	pair := func(a, b core.CoreExpr) {
		if found != "" {
			return
		}
		if v, ok := a.(*core.Var); ok && v.Name == name {
			found = sortOf(b)
		} else if v, ok := b.(*core.Var); ok && v.Name == name {
			found = sortOf(a)
		}
	}
	walk = func(expr core.CoreExpr) {
		if found != "" || expr == nil {
			return
		}
		switch e := expr.(type) {
		case *core.App:
			if len(e.Args) == 2 {
				pair(e.Args[0], e.Args[1])
			}
			walk(e.Func)
			for _, a := range e.Args {
				walk(a)
			}
		case *core.DictApp:
			if len(e.Args) == 2 {
				pair(e.Args[0], e.Args[1])
			}
			for _, a := range e.Args {
				walk(a)
			}
		case *core.BinOp:
			pair(e.Left, e.Right)
			walk(e.Left)
			walk(e.Right)
		case *core.Let:
			if e.Name == name {
				walk(e.Value)
				return // shadowed below
			}
			walk(e.Value)
			walk(e.Body)
		case *core.If:
			walk(e.Cond)
			walk(e.Then)
			walk(e.Else)
		case *core.Match:
			walk(e.Scrutinee)
			for _, arm := range e.Arms {
				walk(arm.Guard)
				walk(arm.Body)
			}
		case *core.Record:
			// `{cells: $t, idx: 0}`: the field's declared sort. ANF hoists the
			// [] of a foldl accumulator or state record out of the literal,
			// where encodeRecord's own field hint cannot reach it.
			found = recordFieldSortOfVar(e, name)
			for _, v := range e.Fields {
				walk(v)
			}
		}
	}
	walk(body)
	return found
}

// recordFieldSortOfVar is the declared sort of the field of rec whose value is
// Var(name), when rec's field set names a declared record ("" otherwise).
func recordFieldSortOfVar(rec *core.Record, name string) string {
	info := lookupRecordByFields(rec.Fields)
	if info == nil {
		return ""
	}
	for field, v := range rec.Fields {
		if vv, ok := v.(*core.Var); ok && vv.Name == name {
			if _, isSeq := seqElemSort(info.FieldSorts[field]); isSeq {
				return info.FieldSorts[field]
			}
		}
	}
	return ""
}

// seqSortOf returns expr's SMT sequence sort from its elaborated type, or ""
// when it has none or is not a list (an empty list literal is typed [α]).
func seqSortOf(expr core.CoreExpr) string {
	if t := hofExprType(expr); t != nil {
		if s, err := MapType(t); err == nil {
			if _, isSeq := seqElemSort(s); isSeq {
				return s
			}
		}
	}
	return ""
}
