package core

import (
	"fmt"
	"sort"
	"strings"
)

// CheckNoShadowing verifies, on one top-level declaration, the property
// UniqueLocals establishes and the passes after type checking rely on (see
// UniqueLocals): no local binder is introduced while a binder of the same
// name is in scope, and no local binder has the name of a free (top-level or
// prelude) reference of the declaration. Binders in sibling scopes (two match
// arms, two copies of a lambda body that monomorphisation placed at two call
// sites) may share a name. `_` and the synthetic dictionary and quantifier
// binders are exempt. Tests run it over pipeline output; the pipeline itself
// does not call it.
func CheckNoShadowing(decl CoreExpr) (err error) {
	defer func() {
		if r := recover(); r != nil { // FreeVars panics on an unknown node type
			err = fmt.Errorf("%v", r)
		}
	}()
	c := &shadowChecker{free: FreeVars(decl), scope: map[string]int{}}
	c.expr(decl)
	if len(c.errs) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(c.errs, "; "))
}

// FreeVars returns the names a declaration reads without binding them.
func FreeVars(e CoreExpr) map[string]bool {
	u := &uniquifier{free: map[string]bool{}}
	u.expr(e, nil)
	return u.free
}

type shadowChecker struct {
	free  map[string]bool
	scope map[string]int
	errs  []string
}

func (c *shadowChecker) bind(names ...string) func() {
	for _, n := range names {
		if n == "_" || n == "" {
			continue
		}
		if c.scope[n] > 0 {
			c.errs = append(c.errs, fmt.Sprintf("%q is bound while an outer %q is in scope", n, n))
		} else if c.free[n] {
			c.errs = append(c.errs, fmt.Sprintf("%q is bound although the declaration also reads a free %q", n, n))
		}
		c.scope[n]++
	}
	return func() {
		for _, n := range names {
			if n != "_" && n != "" {
				c.scope[n]--
			}
		}
	}
}

func (c *shadowChecker) exprs(es ...CoreExpr) {
	for _, e := range es {
		c.expr(e)
	}
}

func (c *shadowChecker) fieldValues(m map[string]CoreExpr) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		c.expr(m[k])
	}
}

func (c *shadowChecker) expr(e CoreExpr) {
	switch x := e.(type) {
	case nil, *Var, *VarGlobal, *Lit, *DictRef:
	case *Lambda:
		done := c.bind(x.Params...)
		c.expr(x.Body)
		done()
	case *Let:
		c.expr(x.Value)
		done := c.bind(x.Name)
		c.expr(x.Body)
		done()
	case *LetRec:
		names := make([]string, len(x.Bindings))
		for i, b := range x.Bindings {
			names[i] = b.Name
		}
		done := c.bind(names...)
		for _, b := range x.Bindings {
			c.expr(b.Value)
		}
		c.expr(x.Body)
		done()
	case *Match:
		c.expr(x.Scrutinee)
		for _, arm := range x.Arms {
			done := c.bind(patternNames(arm.Pattern)...)
			c.exprs(arm.Guard, arm.Body)
			done()
		}
	case *App:
		c.expr(x.Func)
		c.exprs(x.Args...)
	case *If:
		c.exprs(x.Cond, x.Then, x.Else)
	case *BinOp:
		c.exprs(x.Left, x.Right)
	case *UnOp:
		c.expr(x.Operand)
	case *Intrinsic:
		c.exprs(x.Args...)
	case *Record:
		c.fieldValues(x.Fields)
	case *RecordAccess:
		c.expr(x.Record)
	case *RecordUpdate:
		c.expr(x.Base)
		c.fieldValues(x.Updates)
	case *List:
		c.exprs(x.Elements...)
	case *Array:
		c.exprs(x.Elements...)
	case *Tuple:
		c.exprs(x.Elements...)
	case *DictAbs:
		c.expr(x.Body)
	case *DictApp:
		c.expr(x.Dict)
		c.exprs(x.Args...)
	case *Forall:
		c.exprs(x.Lo, x.Hi, x.Body)
	default:
		c.errs = append(c.errs, fmt.Sprintf("unhandled Core expression type %T", e))
	}
}

func patternNames(p CorePattern) []string {
	switch x := p.(type) {
	case *VarPattern:
		return []string{x.Name}
	case *ConstructorPattern:
		return patternListNames(x.Args)
	case *TuplePattern:
		return patternListNames(x.Elements)
	case *ListPattern:
		out := patternListNames(x.Elements)
		if x.Tail != nil {
			out = append(out, patternNames(*x.Tail)...)
		}
		return out
	case *RecordPattern:
		keys := make([]string, 0, len(x.Fields))
		for k := range x.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var out []string
		for _, k := range keys {
			out = append(out, patternNames(x.Fields[k])...)
		}
		return out
	}
	return nil
}

func patternListNames(ps []CorePattern) []string {
	var out []string
	for _, p := range ps {
		out = append(out, patternNames(p)...)
	}
	return out
}
