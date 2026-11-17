package lower

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/core"
)

// uniqueLocals renames the local binders of one top-level declaration so
// that, within it, no two binders share a name and no binder shares a name
// with a free (top-level or prelude) reference.
//
// Why: the Statement IR this package produces has one flat name space per
// function. FlattenBlock and flattenValue hoist the bindings of a nested
// let-chain (`let s = { let r = 2; r * 10 }`) into the enclosing statement
// list, the bytecode compiler maps each name to one register per function
// frame, and lambda capture analysis (freeVarsLambda) treats a name as bound
// from its first declaration on. Lexical scope is therefore only preserved
// when names are unique: with `let r = 1` outside the block, the hoisted
// inner `r` overwrote the outer one and every later read of `r` — including
// a closure capture and a call of a top-level function `r` — saw the inner
// value (#2375, a silent wrong answer on the VM only; the evaluator scopes
// Core directly).
//
// Renaming at the Core level makes every consumer of the flat name space
// correct at once: block lets, letrec groups, match-pattern variables,
// lambda parameters and captures. A binder keeps its name unless an earlier
// binder of the function (or a free reference anywhere in it) already uses
// that name, so code without shadowing lowers exactly as before.
//
// The input is never mutated (the evaluator runs the same Core); changed
// nodes are copied with their NodeID, so CoreTypeInfo lookups still hit.
func uniqueLocals(e core.CoreExpr) core.CoreExpr {
	if e == nil {
		return nil
	}
	// Pass 1: every name read but not bound inside the declaration.
	collect := &uniquifier{free: map[string]bool{}}
	collect.expr(e, nil)
	// Pass 2: rename binders that collide with an earlier binder or a free name.
	rename := &uniquifier{taken: collect.free}
	return rename.expr(e, nil)
}

// scopeEnv is a persistent map from a source name to the name it lowers to.
type scopeEnv struct {
	from, to string
	next     *scopeEnv
}

func (s *scopeEnv) lookup(name string) (string, bool) {
	for ; s != nil; s = s.next {
		if s.from == name {
			return s.to, true
		}
	}
	return "", false
}

// uniquifier runs in one of two modes. With free set it only records free
// names (binders map to themselves). With taken set it renames.
type uniquifier struct {
	free  map[string]bool // pass 1 output
	taken map[string]bool // pass 2: names in use by a binder or a free reference
	n     int
}

// bind opens a scope for name and returns the name it lowers to. `_` binds
// nothing that can be read (match lowering skips a `_` pattern), so it keeps
// its name.
func (u *uniquifier) bind(name string, env *scopeEnv) (string, *scopeEnv) {
	to := name
	if u.taken != nil && name != "_" {
		if u.taken[name] {
			for {
				u.n++
				to = fmt.Sprintf("%s$s%d", name, u.n)
				if !u.taken[to] {
					break
				}
			}
		}
		u.taken[to] = true
	}
	return to, &scopeEnv{from: name, to: to, next: env}
}

// bindKeep opens a scope for a synthetic binder that must keep its name
// (dictionary parameters, contract quantifiers): it only shadows.
func (u *uniquifier) bindKeep(name string, env *scopeEnv) *scopeEnv {
	return &scopeEnv{from: name, to: name, next: env}
}

func (u *uniquifier) bindAll(names []string, env *scopeEnv) ([]string, *scopeEnv, bool) {
	out := names
	changed := false
	for i, name := range names {
		to, next := u.bind(name, env)
		env = next
		if to != name {
			if !changed {
				out = append([]string(nil), names...)
				changed = true
			}
			out[i] = to
		}
	}
	return out, env, changed
}

func (u *uniquifier) exprs(es []core.CoreExpr, env *scopeEnv) ([]core.CoreExpr, bool) {
	var out []core.CoreExpr
	for i, e := range es {
		ne := u.expr(e, env)
		if ne != e && out == nil {
			out = append([]core.CoreExpr(nil), es...)
		}
		if out != nil {
			out[i] = ne
		}
	}
	if out == nil {
		return es, false
	}
	return out, true
}

// fields visits a record's field values in sorted key order, so the names
// chosen for binders inside them do not depend on map iteration order.
func (u *uniquifier) fields(m map[string]core.CoreExpr, env *scopeEnv) (map[string]core.CoreExpr, bool) {
	var out map[string]core.CoreExpr
	for _, k := range sortedKeys(m) {
		v := m[k]
		nv := u.expr(v, env)
		if nv == v {
			continue
		}
		if out == nil {
			out = make(map[string]core.CoreExpr, len(m))
			for k2, v2 := range m {
				out[k2] = v2
			}
		}
		out[k] = nv
	}
	if out == nil {
		return m, false
	}
	return out, true
}

// expr returns e with references and binders renamed under env. It returns
// e itself when nothing in it changed. An unknown node type panics: a
// silently skipped subtree would keep a stale name and reintroduce the
// capture this pass exists to prevent (lowerTopLevelDeclSafe turns the panic
// into an evaluator-only function).
func (u *uniquifier) expr(e core.CoreExpr, env *scopeEnv) core.CoreExpr {
	if e == nil {
		return nil
	}
	switch x := e.(type) {
	case *core.Var:
		to, ok := env.lookup(x.Name)
		if !ok {
			if u.free != nil {
				u.free[x.Name] = true
			}
			return x
		}
		if to == x.Name {
			return x
		}
		n := *x
		n.Name = to
		return &n

	case *core.VarGlobal, *core.Lit, *core.DictRef:
		return x

	case *core.Lambda:
		params, inner, pc := u.bindAll(x.Params, env)
		body := u.expr(x.Body, inner)
		if !pc && body == x.Body {
			return x
		}
		n := *x
		n.Params, n.Body = params, body
		return &n

	case *core.Let:
		val := u.expr(x.Value, env)
		name, inner := u.bind(x.Name, env)
		body := u.expr(x.Body, inner)
		if val == x.Value && name == x.Name && body == x.Body {
			return x
		}
		n := *x
		n.Value, n.Name, n.Body = val, name, body
		return &n

	case *core.LetRec:
		names := make([]string, len(x.Bindings))
		for i, b := range x.Bindings {
			names[i] = b.Name
		}
		// Every member is in scope in every member's value and in the body.
		renamed, inner, changed := u.bindAll(names, env)
		values := make([]core.CoreExpr, len(x.Bindings))
		for i, b := range x.Bindings {
			values[i] = u.expr(b.Value, inner)
			changed = changed || values[i] != b.Value
		}
		body := u.expr(x.Body, inner)
		if !changed && body == x.Body {
			return x
		}
		bindings := make([]core.RecBinding, len(x.Bindings))
		for i := range x.Bindings {
			bindings[i] = core.RecBinding{Name: renamed[i], Value: values[i]}
		}
		n := *x
		n.Bindings, n.Body = bindings, body
		return &n

	case *core.App:
		fn := u.expr(x.Func, env)
		args, ac := u.exprs(x.Args, env)
		if fn == x.Func && !ac {
			return x
		}
		n := *x
		n.Func, n.Args = fn, args
		return &n

	case *core.If:
		c, t, el := u.expr(x.Cond, env), u.expr(x.Then, env), u.expr(x.Else, env)
		if c == x.Cond && t == x.Then && el == x.Else {
			return x
		}
		n := *x
		n.Cond, n.Then, n.Else = c, t, el
		return &n

	case *core.Match:
		return u.match(x, env)

	case *core.BinOp:
		l, r := u.expr(x.Left, env), u.expr(x.Right, env)
		if l == x.Left && r == x.Right {
			return x
		}
		n := *x
		n.Left, n.Right = l, r
		return &n

	case *core.UnOp:
		o := u.expr(x.Operand, env)
		if o == x.Operand {
			return x
		}
		n := *x
		n.Operand = o
		return &n

	case *core.Intrinsic:
		args, ac := u.exprs(x.Args, env)
		if !ac {
			return x
		}
		n := *x
		n.Args = args
		return &n

	case *core.Record:
		fs, fc := u.fields(x.Fields, env)
		if !fc {
			return x
		}
		n := *x
		n.Fields = fs
		return &n

	case *core.RecordAccess:
		r := u.expr(x.Record, env)
		if r == x.Record {
			return x
		}
		n := *x
		n.Record = r
		return &n

	case *core.RecordUpdate:
		b := u.expr(x.Base, env)
		ups, uc := u.fields(x.Updates, env)
		if b == x.Base && !uc {
			return x
		}
		n := *x
		n.Base, n.Updates = b, ups
		return &n

	case *core.List:
		els, c := u.exprs(x.Elements, env)
		if !c {
			return x
		}
		n := *x
		n.Elements = els
		return &n

	case *core.Array:
		els, c := u.exprs(x.Elements, env)
		if !c {
			return x
		}
		n := *x
		n.Elements = els
		return &n

	case *core.Tuple:
		els, c := u.exprs(x.Elements, env)
		if !c {
			return x
		}
		n := *x
		n.Elements = els
		return &n

	case *core.DictAbs:
		inner := env
		for _, p := range x.Params {
			inner = u.bindKeep(p.Name, inner)
		}
		body := u.expr(x.Body, inner)
		if body == x.Body {
			return x
		}
		n := *x
		n.Body = body
		return &n

	case *core.DictApp:
		d := u.expr(x.Dict, env)
		args, ac := u.exprs(x.Args, env)
		if d == x.Dict && !ac {
			return x
		}
		n := *x
		n.Dict, n.Args = d, args
		return &n

	case *core.Forall:
		lo, hi := u.expr(x.Lo, env), u.expr(x.Hi, env)
		body := u.expr(x.Body, u.bindKeep(x.Var, env))
		if lo == x.Lo && hi == x.Hi && body == x.Body {
			return x
		}
		n := *x
		n.Lo, n.Hi, n.Body = lo, hi, body
		return &n
	}
	panic(fmt.Sprintf("lower: uniqueLocals: unhandled Core expression type %T", e))
}

// match renames each arm's pattern variables; they scope over the guard and
// the body of that arm only.
func (u *uniquifier) match(x *core.Match, env *scopeEnv) core.CoreExpr {
	scrut := u.expr(x.Scrutinee, env)
	changed := scrut != x.Scrutinee
	arms := make([]core.MatchArm, len(x.Arms))
	for i, arm := range x.Arms {
		inner := env
		pat := u.pattern(arm.Pattern, &inner)
		arms[i] = core.MatchArm{Pattern: pat, Guard: u.expr(arm.Guard, inner), Body: u.expr(arm.Body, inner)}
		changed = changed || pat != arm.Pattern || arms[i].Guard != arm.Guard || arms[i].Body != arm.Body
	}
	if !changed {
		return x
	}
	n := *x
	n.Scrutinee, n.Arms = scrut, arms
	return &n
}

// pattern renames the variables p binds, extending *env with each.
func (u *uniquifier) pattern(p core.CorePattern, env **scopeEnv) core.CorePattern {
	switch x := p.(type) {
	case nil:
		return nil
	case *core.VarPattern:
		to, next := u.bind(x.Name, *env)
		*env = next
		if to == x.Name {
			return x
		}
		return &core.VarPattern{Name: to}
	case *core.LitPattern, *core.WildcardPattern:
		return x
	case *core.ConstructorPattern:
		args, c := u.patterns(x.Args, env)
		if !c {
			return x
		}
		n := *x
		n.Args = args
		return &n
	case *core.TuplePattern:
		els, c := u.patterns(x.Elements, env)
		if !c {
			return x
		}
		return &core.TuplePattern{Elements: els}
	case *core.ListPattern:
		els, c := u.patterns(x.Elements, env)
		tail := x.Tail
		if x.Tail != nil {
			if nt := u.pattern(*x.Tail, env); nt != *x.Tail {
				tail = &nt
				c = true
			}
		}
		if !c {
			return x
		}
		return &core.ListPattern{Elements: els, Tail: tail}
	case *core.RecordPattern:
		// Field order is irrelevant to scoping (one name per field, all
		// distinct), but renaming must be deterministic: visit sorted keys.
		keys := sortedKeys(x.Fields)
		var out map[string]core.CorePattern
		for _, k := range keys {
			np := u.pattern(x.Fields[k], env)
			if np == x.Fields[k] {
				continue
			}
			if out == nil {
				out = make(map[string]core.CorePattern, len(x.Fields))
				for k2, v2 := range x.Fields {
					out[k2] = v2
				}
			}
			out[k] = np
		}
		if out == nil {
			return x
		}
		return &core.RecordPattern{Fields: out, Rest: x.Rest}
	}
	panic(fmt.Sprintf("lower: uniqueLocals: unhandled Core pattern type %T", p))
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (u *uniquifier) patterns(ps []core.CorePattern, env **scopeEnv) ([]core.CorePattern, bool) {
	var out []core.CorePattern
	for i, p := range ps {
		np := u.pattern(p, env)
		if np != p && out == nil {
			out = append([]core.CorePattern(nil), ps...)
		}
		if out != nil {
			out[i] = np
		}
	}
	if out == nil {
		return ps, false
	}
	return out, true
}
