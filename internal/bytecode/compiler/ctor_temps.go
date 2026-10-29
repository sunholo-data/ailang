package compiler

import (
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// inlineCtorTemps puts single-use compiler temps back into the constructor
// literal that consumes them. Elaboration binds every non-atomic element of a
// list, tuple, record or ADT literal to a `$tmpN` local first, so a literal
// of 128 calls became 128 pinned locals plus the literal: the 64-register
// chunk of a long list then found no room and the function went
// evaluator-only under --strict-bytecode (#1865; constants are atoms, so
// they never hit this). Compiling each value straight into its slot needs
// one transient register instead of one pinned local per element.
//
// The rewrite only fires when it cannot reorder evaluation: the temps moved
// into a statement are the ones declared immediately before it, each is read
// exactly once in the whole block, the read sits in a constructor position
// (reached only through literals, never through a call or operator), and the
// order the compiler evaluates those positions in is the declaration order.
func inlineCtorTemps(stmts []stmt.Stmt) []stmt.Stmt {
	if !hasCtorTempCandidate(stmts) {
		return stmts
	}
	reads := map[string]int{}
	v := &freeVarVisitor{bound: map[string]bool{}, seen: map[string]bool{}, counts: reads}
	for _, s := range stmts {
		v.visitStmt(s)
	}
	out := make([]stmt.Stmt, 0, len(stmts))
	for _, s := range stmts {
		// The pending run: trailing single-use $tmp decls already in out.
		start := len(out)
		for start > 0 {
			d, ok := out[start-1].(stmt.VarDecl)
			if !ok || !strings.HasPrefix(d.Name, "$tmp") || reads[d.Name] != 1 {
				break
			}
			start--
		}
		// Try the longest suffix of the run first; a shorter suffix is still
		// order-safe because it is the latest-declared part.
		for lo := start; lo < len(out); lo++ {
			if merged, ok := absorbTemps(out[lo:], s); ok {
				out = append(out[:lo], merged)
				goto next
			}
		}
		out = append(out, s)
	next:
	}
	return out
}

// inlineBodyCtorTemps applies inlineCtorTemps to a function or lambda body
// whose final expression is kept apart from its statements (FuncDecl.Return,
// Lambda.Return), so temps consumed by that expression are covered too.
func inlineBodyCtorTemps(body []stmt.Stmt, ret stmt.Expr) ([]stmt.Stmt, stmt.Expr) {
	if ret == nil || !hasCtorTempCandidate(body) {
		return inlineCtorTemps(body), ret
	}
	all := inlineCtorTemps(append(append([]stmt.Stmt(nil), body...), stmt.ReturnStmt{Value: ret}))
	return all[:len(all)-1], all[len(all)-1].(stmt.ReturnStmt).Value
}

// hasCtorTempCandidate is a cheap prefilter: any $tmp decl at all.
func hasCtorTempCandidate(stmts []stmt.Stmt) bool {
	for _, s := range stmts {
		if d, ok := s.(stmt.VarDecl); ok && strings.HasPrefix(d.Name, "$tmp") {
			return true
		}
	}
	return false
}

// absorbTemps substitutes every decl in run into s, or reports false.
func absorbTemps(run []stmt.Stmt, s stmt.Stmt) (stmt.Stmt, bool) {
	vals := make(map[string]stmt.Expr, len(run))
	order := make([]string, len(run))
	for i, r := range run {
		d := r.(stmt.VarDecl)
		vals[d.Name] = d.Value
		order[i] = d.Name
	}
	sub := &ctorSubst{vals: vals}
	var out stmt.Stmt
	switch st := s.(type) {
	case stmt.VarDecl:
		st.Value = sub.expr(st.Value)
		out = st
	case stmt.ReturnStmt:
		st.Value = sub.expr(st.Value)
		out = st
	case stmt.ExprStmt:
		st.Value = sub.expr(st.Value)
		out = st
	case stmt.AssignStmt:
		st.Value = sub.expr(st.Value)
		out = st
	default:
		return nil, false
	}
	if sub.bad || len(sub.used) != len(order) {
		return nil, false
	}
	for i, name := range order {
		if sub.used[i] != name {
			return nil, false
		}
	}
	return out, true
}

// ctorSubst replaces temp reads in constructor positions. used records the
// substituted temps in evaluation-completion order (a temp nested inside
// another temp's value completes first, as it did when declared first). Any
// temp read outside a constructor position marks the substitution bad.
type ctorSubst struct {
	vals map[string]stmt.Expr
	used []string
	bad  bool
}

// expr walks the constructor spine of e. Leaves that are not constructors
// are kept as written; they must not read a temp of the run.
func (c *ctorSubst) expr(e stmt.Expr) stmt.Expr {
	switch x := e.(type) {
	case stmt.VarRef:
		if val, ok := c.vals[x.Name]; ok {
			delete(c.vals, x.Name)
			inner := c.expr(val)
			c.used = append(c.used, x.Name)
			return inner
		}
		return x
	case stmt.ListLit:
		x.Elems = c.exprs(x.Elems)
		return x
	case stmt.ArrayLit:
		x.Elems = c.exprs(x.Elems)
		return x
	case stmt.TupleLit:
		x.Elems = c.exprs(x.Elems)
		return x
	case stmt.ADTConstructor:
		x.Args = c.exprs(x.Args)
		return x
	case stmt.RecordLit:
		// compileRecordLit evaluates fields in sorted name order.
		idx := make([]int, len(x.Fields))
		for i := range idx {
			idx[i] = i
		}
		sort.SliceStable(idx, func(a, b int) bool { return x.Fields[idx[a]].Name < x.Fields[idx[b]].Name })
		fields := append([]stmt.FieldInit(nil), x.Fields...)
		for _, i := range idx {
			fields[i].Value = c.expr(fields[i].Value)
		}
		x.Fields = fields
		return x
	}
	if c.readsRun(e) {
		c.bad = true
	}
	return e
}

func (c *ctorSubst) exprs(es []stmt.Expr) []stmt.Expr {
	out := make([]stmt.Expr, len(es))
	for i, e := range es {
		out[i] = c.expr(e)
	}
	return out
}

// readsRun reports whether an opaque expression reads a not-yet-used temp.
func (c *ctorSubst) readsRun(e stmt.Expr) bool {
	v := &freeVarVisitor{bound: map[string]bool{}, seen: map[string]bool{}}
	v.visitExpr(e)
	for _, name := range v.order {
		if _, ok := c.vals[name]; ok {
			return true
		}
	}
	return false
}
