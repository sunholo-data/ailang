package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// findIndex(\x -> p, xs) (M-VERIFY-SOUNDNESS-COVERAGE). Every std/json
// accessor (get, getString, getObject, …) reaches it through findInList, so
// every function reading a JSON field was skipped on it.
//
// The specialisation works over the index alone, -1 for "not found":
//
//	findIndex_spec(xs) = if len(xs) == 0 then -1
//	                     else if p(head(xs)) then 0
//	                     else let r = findIndex_spec(tail(xs)) in
//	                          if r < 0 then -1 else r + 1
//
// and the call site rebuilds the Option: `if r < 0 then None else Some(r)`.
// Built in the specialiser, Some and None would have no checked type to
// name their sort ((Option Int)); the rebuilding node carries the original
// call's node ID, so its elaborated type gives it, as for head/nth/last.

// optFromIndex is the rebuilding node's builtin name. It never appears in
// source: InlineHOFCalls creates it, after every encodability gate has run.
const optFromIndex = "$opt_from_index"

// indexSucc is the specialisation's r + 1. It is no AILANG `+`: the runtime
// findIndex is an iterative Go loop returning a position in the list, so the
// int64 overflow obligation an add_Int carries would be a caveat about
// arithmetic that never runs.
const indexSucc = "$index_succ"

// SpecializeFindIndex builds the index-returning recursion above.
func SpecializeFindIndex(funcName string, lambda *core.Lambda, listParam string) (core.CoreExpr, string) {
	specName := fmt.Sprintf("_findIndex_spec_%s", funcName)
	if len(lambda.Params) == 0 {
		return nil, ""
	}
	intOp := func(op string, a, b core.CoreExpr) core.CoreExpr {
		return &core.App{Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: op}}, Args: []core.CoreExpr{a, b}}
	}
	lit := func(n int64) core.CoreExpr { return &core.Lit{Kind: core.IntLit, Value: n} }
	r := &core.Var{Name: "_hof_r"}
	pred := SubstituteLambdaVar(lambda.Body, lambda.Params[0], &core.Var{Name: "_hof_h"})
	step := &core.If{
		Cond: pred,
		Then: lit(0),
		Else: &core.Let{
			Name:  r.Name,
			Value: &core.App{Func: &core.Var{Name: specName}, Args: []core.CoreExpr{&core.Var{Name: "_hof_t"}}},
			Body: &core.If{Cond: intOp("lt_Int", r, lit(0)), Then: lit(-1),
				Else: &core.App{Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: indexSucc}}, Args: []core.CoreExpr{r}}},
		},
	}
	return hofListStep(&core.Var{Name: listParam}, lit(-1), step), specName
}

// findIndexFrontier: beyond the unrolling depth the index is unknown, but
// always -1 or a valid position (u := the real result is an instance: sound).
func findIndexFrontier(level0 string, sorts []string, _ string) []string {
	binders := make([]string, len(sorts))
	vars := make([]string, len(sorts))
	for i, s := range sorts {
		vars[i] = fmt.Sprintf("$fx%d", i)
		binders[i] = fmt.Sprintf("(%s %s)", vars[i], s)
	}
	u := level0 + "$u"
	call := fmt.Sprintf("(%s %s)", u, strings.Join(vars, " "))
	return []string{
		fmt.Sprintf("(declare-fun %s (%s) Int)", u, strings.Join(sorts, " ")),
		fmt.Sprintf("(define-fun %s (%s) Int (ite (and (>= %[3]s (- 1)) (< %[3]s (seq.len %[4]s))) %[3]s (- 1)))", level0, strings.Join(binders, " "), call, vars[0]),
	}
}

// optionFromIndex wraps the specialised call: the node has the original
// findIndex call's ID, so its type is the call's Option[int].
func optionFromIndex(original *core.App, index core.CoreExpr) core.CoreExpr {
	return &core.App{
		CoreNode: core.CoreNode{NodeID: original.ID()},
		Func:     &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: optFromIndex}},
		Args:     []core.CoreExpr{index},
	}
}

// encodeFindIndexNode encodes the two nodes only the findIndex specialisation
// builds. ok is false for any other builtin.
func encodeFindIndexNode(name string, app *core.App) (string, bool, error) {
	switch name {
	case optFromIndex:
		enc, err := encodeOptionFromIndex(app)
		return enc, true, err
	case indexSucc:
		if len(app.Args) != 1 {
			return "", true, fmt.Errorf("%w: %s expects 1 arg, got %d", ErrUnsupportedConstruct, indexSucc, len(app.Args))
		}
		r, err := EncodeExpr(app.Args[0])
		return fmt.Sprintf("(+ %s 1)", r), true, err
	}
	return "", false, nil
}

// encodeOptionFromIndex encodes optionFromIndex's node.
func encodeOptionFromIndex(app *core.App) (string, error) {
	if len(app.Args) != 1 {
		return "", fmt.Errorf("%w: %s expects 1 arg, got %d", ErrUnsupportedConstruct, optFromIndex, len(app.Args))
	}
	t := hofExprType(app)
	if t == nil {
		return "", fmt.Errorf("%w: the Option result type of a findIndex call is unknown", ErrUnsupportedConstruct)
	}
	sort, err := MapType(t)
	if err != nil {
		return "", fmt.Errorf("%w: the result type of a findIndex call has no SMT sort: %v", ErrUnsupportedConstruct, err)
	}
	r, err := EncodeExpr(app.Args[0])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(let (($fi_r %s)) (ite (< $fi_r 0) (as None %s) ((as Some %s) $fi_r)))", r, sort, sort), nil
}
