package smt

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// HOF lambda captures (M-VERIFY-SOUNDNESS-COVERAGE M2).
//
// A literal lambda passed to map/filter/foldl/any/flatMap often refers to the
// enclosing function's parameters or lets (`filter(\r. r.session == s, rs)`).
// The specialised chain is a separate define-fun, so each such free variable
// becomes an extra parameter of every level, is passed on at each recursive
// call, and is supplied by the original variable at the call site.

// hofCapture is one free variable of a HOF lambda.
type hofCapture struct {
	Name string
	Node *core.Var // first occurrence: carries the elaborated type
	Type types.Type
}

// lambdaCaptures returns the free value variables of lam's body in first-use
// order. A plain Var in call-head position is a function (a callee), not a
// captured value, and is left to callee resolution.
func lambdaCaptures(lam *core.Lambda) ([]hofCapture, error) {
	bound := make(map[string]int)
	for _, p := range lam.Params {
		bound[p]++
	}
	seen := make(map[string]bool)
	var caps []hofCapture
	var walk func(e core.CoreExpr, head bool)
	bind := func(names []string, f func()) {
		for _, n := range names {
			bound[n]++
		}
		f()
		for _, n := range names {
			bound[n]--
		}
	}
	walk = func(expr core.CoreExpr, head bool) {
		switch e := expr.(type) {
		case nil:
		case *core.Var:
			if head || bound[e.Name] > 0 || seen[e.Name] {
				return
			}
			seen[e.Name] = true
			caps = append(caps, hofCapture{Name: e.Name, Node: e})
		case *core.App:
			walk(e.Func, true)
			for _, a := range e.Args {
				walk(a, false)
			}
		case *core.Lambda:
			bind(e.Params, func() { walk(e.Body, false) })
		case *core.Let:
			walk(e.Value, false)
			bind([]string{e.Name}, func() { walk(e.Body, false) })
		case *core.LetRec:
			names := make([]string, len(e.Bindings))
			for i, b := range e.Bindings {
				names[i] = b.Name
			}
			bind(names, func() {
				for _, b := range e.Bindings {
					walk(b.Value, false)
				}
				walk(e.Body, false)
			})
		case *core.If:
			walk(e.Cond, false)
			walk(e.Then, false)
			walk(e.Else, false)
		case *core.Match:
			walk(e.Scrutinee, false)
			for _, arm := range e.Arms {
				bind(patternBinders(arm.Pattern), func() {
					walk(arm.Guard, false)
					walk(arm.Body, false)
				})
			}
		case *core.BinOp:
			walk(e.Left, false)
			walk(e.Right, false)
		case *core.UnOp:
			walk(e.Operand, false)
		case *core.Record:
			for _, k := range sortedFieldKeys(e.Fields) {
				walk(e.Fields[k], false)
			}
		case *core.RecordAccess:
			walk(e.Record, false)
		case *core.RecordUpdate:
			walk(e.Base, false)
			for _, k := range sortedFieldKeys(e.Updates) {
				walk(e.Updates[k], false)
			}
		case *core.List:
			for _, x := range e.Elements {
				walk(x, false)
			}
		case *core.Tuple:
			for _, x := range e.Elements {
				walk(x, false)
			}
		case *core.Intrinsic:
			for _, x := range e.Args {
				walk(x, false)
			}
		case *core.DictApp:
			for _, x := range e.Args {
				walk(x, false)
			}
		case *core.Forall:
			walk(e.Lo, false)
			walk(e.Hi, false)
			bind([]string{e.Var}, func() { walk(e.Body, false) })
		}
	}
	walk(lam.Body, false)
	for i := range caps {
		t := hofExprType(caps[i].Node)
		if t == nil {
			return nil, fmt.Errorf("the lambda captures %q, whose type is unknown", caps[i].Name)
		}
		if _, err := MapType(t); err != nil {
			return nil, fmt.Errorf("the lambda captures %q, whose type has no SMT sort: %v", caps[i].Name, err)
		}
		caps[i].Type = t
	}
	return caps, nil
}

// appendCaptureArgs appends the captured variables to every call of specName in
// body (the specialisation's recursive calls).
func appendCaptureArgs(body core.CoreExpr, specName string, caps []hofCapture) core.CoreExpr {
	if len(caps) == 0 {
		return body
	}
	extra := make([]core.CoreExpr, len(caps))
	for i, c := range caps {
		v := *c.Node
		v.Name = captureSymbol(c.Name)
		extra[i] = &v
	}
	var rewrite func(e core.CoreExpr) core.CoreExpr
	rewrite = func(expr core.CoreExpr) core.CoreExpr {
		switch e := expr.(type) {
		case *core.App:
			n := *e
			n.Func = rewrite(e.Func)
			n.Args = make([]core.CoreExpr, len(e.Args))
			for i, a := range e.Args {
				n.Args[i] = rewrite(a)
			}
			if v, ok := e.Func.(*core.Var); ok && v.Name == specName {
				n.Args = append(n.Args, extra...)
			}
			return &n
		case *core.If:
			n := *e
			n.Cond, n.Then, n.Else = rewrite(e.Cond), rewrite(e.Then), rewrite(e.Else)
			return &n
		case *core.Let:
			n := *e
			n.Value, n.Body = rewrite(e.Value), rewrite(e.Body)
			return &n
		}
		return expr // the skeleton's recursive call sits under If/Let/App only
	}
	return rewrite(body)
}

func sortedFieldKeys(m map[string]core.CoreExpr) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// captureSymbol is the parameter name a captured variable is bound under inside
// a HOF specialisation (see letSymbol for the same hygiene on lets).
func captureSymbol(name string) string {
	return "$c_" + name
}
