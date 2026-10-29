package types

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// ifc_summary_local.go — per-module function summaries for the callable
// domain (M-IFC-FUNCTION-VALUE-SINKS M2, #1719).
//
// Every parameter of a body being summarised (a module function, a lambda, a
// function literal) is seeded with a label variable ρ and, as a value, with a
// probe ("parameter i of this body"). Walking the body then records DEFERRED
// APPLICATIONS on the innermost active summary:
//
//   - applying a callable that carries demands to an argument whose label
//     mentions ρ_j records (that demand, args): an INFERRED demand on
//     parameter j (M2b: `wrap(y) = gate(y)` forbids <email> on y);
//   - applying a probe records (probe, args): the APPLIED-AT summary of that
//     function parameter (M2a: `apply1(f, x) = f(x)` applies f to ρ_x).
//
// Applying a summarised callable substitutes ρ ↦ actual argument labels and
// probe ↦ actual argument callables, and replays its deferred applications, so
// the diagnostic lands at the call that supplies the concrete label. Module
// summaries iterate to a fixed point in source order. ρ never leaves this
// machinery: it is erased from every result label, Check B input and message.

type ifcSummary struct {
	key      string
	deferred []ifcDeferred
	seen     map[string]bool
	result   *ifcCallable
}

func newSummary(key string) *ifcSummary {
	return &ifcSummary{key: key, seen: map[string]bool{}}
}

func (s *ifcSummary) record(d ifcDeferred) {
	k := deferredKey(d)
	if !s.seen[k] {
		s.seen[k] = true
		s.deferred = append(s.deferred, d)
	}
}

func (s *ifcSummary) fingerprint() string {
	if s == nil {
		return ""
	}
	parts := make([]string, 0, len(s.seen))
	for k := range s.seen {
		parts = append(parts, k)
	}
	sort.Strings(parts)
	return strings.Join(parts, ";") + "->" + callableKey(s.result)
}

// --- label variables ---------------------------------------------------------

type rhoRef struct {
	owner string
	idx   int
}

func (c *ifcChecker) rho(owner string, idx int) Label {
	name := fmt.Sprintf("ρ%s#%d", owner, idx)
	c.rhos[name] = rhoRef{owner: owner, idx: idx}
	return LabelVar(name)
}

func hasLabelVar(l Label) bool {
	for _, p := range normalisedParts(l) {
		if _, ok := p.(labelVar); ok {
			return true
		}
	}
	return false
}

// mapParts rebuilds l with each constituent replaced by f(part).
func mapParts(l Label, f func(Label) Label) Label {
	out := LabelBottom()
	for _, p := range normalisedParts(l) {
		out = LabelJoin(out, f(p))
	}
	return out
}

// eraseRho drops every label variable: the concrete part of a label.
func eraseRho(l Label) Label {
	if !hasLabelVar(l) {
		return l
	}
	return mapParts(l, func(p Label) Label {
		if _, ok := p.(labelVar); ok {
			return LabelBottom()
		}
		return p
	})
}

// eraseOwner drops the label variables of one summary (a lambda's own params
// when its label leaves the lambda).
func (c *ifcChecker) eraseOwner(l Label, owner string) Label {
	if !hasLabelVar(l) {
		return l
	}
	return mapParts(l, func(p Label) Label {
		if v, ok := p.(labelVar); ok && c.rhos[v.name].owner == owner {
			return LabelBottom()
		}
		return p
	})
}

// activeRho is the part of l made of label variables of active summaries.
func (c *ifcChecker) activeRho(l Label) Label {
	if !hasLabelVar(l) {
		return LabelBottom()
	}
	return mapParts(l, func(p Label) Label {
		if v, ok := p.(labelVar); ok && c.isActive(c.rhos[v.name].owner) {
			return p
		}
		return LabelBottom()
	})
}

func (c *ifcChecker) substLabel(l Label, owner string, args []ifcVal) Label {
	if !hasLabelVar(l) {
		return l
	}
	return mapParts(l, func(p Label) Label {
		v, ok := p.(labelVar)
		if !ok {
			return p
		}
		r := c.rhos[v.name]
		if r.owner != owner {
			return p
		}
		if r.idx < len(args) {
			return args[r.idx].label
		}
		return LabelBottom()
	})
}

// --- the active summary stack ------------------------------------------------

func (c *ifcChecker) isActive(owner string) bool {
	for _, s := range c.owners {
		if s.key == owner {
			return true
		}
	}
	return false
}

func (c *ifcChecker) push(s *ifcSummary) { c.owners = append(c.owners, s) }
func (c *ifcChecker) pop()               { c.owners = c.owners[:len(c.owners)-1] }

// recordDeferred records an application on the innermost active summary.
// With keepConcrete false only the symbolic (active ρ) part of each argument
// label is kept: the concrete part was already checked against the demand
// where it was applied. A probe's target is unknown until the summary is
// replayed, so it keeps the concrete part too (R10: a HOF applying its
// function argument to data it sources itself).
func (c *ifcChecker) recordDeferred(target *ifcCallable, args []ifcVal, via []string, keepConcrete bool) {
	if len(c.owners) == 0 || target == nil {
		return
	}
	trimmed := make([]ifcVal, len(args))
	for i, a := range args {
		l := c.activeRho(a.label)
		if keepConcrete {
			l = LabelJoin(l, eraseRho(a.label))
		}
		trimmed[i] = ifcVal{label: l, fn: a.fn}
	}
	c.owners[len(c.owners)-1].record(ifcDeferred{target: target, args: trimmed, via: append([]string(nil), via...)})
}

func probe(owner string, idx, res int) *ifcCallable {
	return single(&ifcFn{kind: fnProbe, owner: owner, idx: idx, res: res})
}

// --- application -------------------------------------------------------------

// applyCallable checks an application of fn to args (Checks A/C on every
// alternative's demands), replays summaries, and returns the callable of the
// application's result. A nil callee is unknown and takes the fallback.
func (c *ifcChecker) applyCallable(fn *ifcCallable, args []ifcVal, at *ast.FuncCall, calleeLabel Label, via []string, depth int) *ifcCallable {
	c.analysisStep()
	if depth > ifcAnalysisDepth {
		panic(ifcAnalysisLimit{reason: "callable summary nesting"})
	}
	if fn == nil {
		return c.applyOpaque(args, at, calleeLabel, via, depth)
	}
	var result *ifcCallable
	for _, a := range fn.alts {
		switch a.kind {
		case fnTop:
			c.checkDemands(a, args, at, via)
			if s := c.summaries[a.name]; s != nil {
				c.replay(s.key, s.deferred, args, at, append(append([]string(nil), via...), a.name), depth)
				result = joinCallable(result, c.substCallable(s.result, s.key, args))
			}
		case fnLambda:
			c.checkDemands(a, args, at, via)
			c.replay(a.owner, a.deferred, args, at, via, depth)
			result = joinCallable(result, c.substCallable(a.result, a.owner, args))
		case fnDeclared:
			c.checkDemands(a, args, at, via)
			result = joinCallable(result, a.result)
		case fnProbe:
			if c.isActive(a.owner) {
				c.recordDeferred(single(a), args, via, true)
			}
			result = joinCallable(result, probe(a.owner, a.idx, a.res+1))
		case fnOpaque:
			result = joinCallable(result, c.applyOpaque(args, at, calleeLabel, via, depth))
		case fnCtor:
			for _, arg := range args {
				result = joinCallable(result, arg.fn)
			}
		}
	}
	return result
}

// checkDemands runs Checks A/C for one alternative and, when an argument's
// label is symbolic, records the demand on the innermost summary (inference).
func (c *ifcChecker) checkDemands(a *ifcFn, args []ifcVal, at *ast.FuncCall, via []string) {
	symbolic := false
	for i, d := range a.params {
		if i >= len(args) || d.empty() {
			continue
		}
		if !c.silent {
			c.addErrs(demandErrors(at, a.name, d, eraseRho(args[i].label), via))
		}
		if !LabelEqual(c.activeRho(args[i].label), LabelBottom()) {
			symbolic = true
		}
	}
	if symbolic {
		c.recordDeferred(single(&ifcFn{kind: fnDeclared, name: a.name, params: a.params}), args, via, false)
	}
}

// applyOpaque is the conservative fallback for a callee with no summary
// (imported, unknown): every callable argument is assumed applied to the join
// of the other arguments' labels and the callee's own label (design F2).
func (c *ifcChecker) applyOpaque(args []ifcVal, at *ast.FuncCall, calleeLabel Label, via []string, depth int) *ifcCallable {
	result := single(&ifcFn{kind: fnOpaque})
	for j, a := range args {
		if a.fn == nil {
			continue
		}
		result = joinCallable(result, a.fn)
		other := calleeLabel
		for k, b := range args {
			if k != j {
				other = LabelJoin(other, b.label)
			}
		}
		n := 1
		for _, alt := range a.fn.alts {
			if len(alt.params) > n {
				n = len(alt.params)
			}
			if alt.kind == fnTop && c.sigs[alt.name] != nil && len(c.sigs[alt.name].params) > n {
				n = len(c.sigs[alt.name].params)
			}
		}
		fargs := make([]ifcVal, n)
		for i := range fargs {
			fargs[i] = ifcVal{label: other}
		}
		c.applyCallable(a.fn, fargs, at, LabelBottom(), via, depth+1)
	}
	return result
}

// replay re-applies a summary's deferred applications with the owner's ρ and
// probes substituted by the actual arguments.
func (c *ifcChecker) replay(owner string, deferred []ifcDeferred, args []ifcVal, at *ast.FuncCall, via []string, depth int) {
	key := owner + deferredKey(ifcDeferred{args: args})
	if c.replaying[key] {
		return
	}
	c.replaying[key] = true
	defer delete(c.replaying, key)
	for _, d := range deferred {
		target := c.substCallable(d.target, owner, args)
		dargs := make([]ifcVal, len(d.args))
		for i, a := range d.args {
			dargs[i] = ifcVal{label: c.substLabel(a.label, owner, args), fn: c.substCallable(a.fn, owner, args)}
		}
		if target == nil {
			continue // a probe bound to a function with no requirements
		}
		c.applyCallable(target, dargs, at, LabelBottom(), append(append([]string(nil), via...), d.via...), depth+1)
	}
}

// substCallable substitutes owner's probes by the actual argument callables
// and owner's ρ by the actual argument labels, through lambda summaries.
func (c *ifcChecker) substCallable(fn *ifcCallable, owner string, args []ifcVal) *ifcCallable {
	if fn == nil {
		return nil
	}
	var out *ifcCallable
	for _, a := range fn.alts {
		switch {
		case a.kind == fnProbe && a.owner == owner:
			if a.idx < len(args) {
				out = joinCallable(out, c.resultN(args[a.idx].fn, a.res))
			}
		case a.kind == fnLambda:
			n := &ifcFn{kind: fnLambda, name: a.name, params: a.params, owner: a.owner,
				result: c.substCallable(a.result, owner, args)}
			for _, d := range a.deferred {
				nd := ifcDeferred{target: c.substCallable(d.target, owner, args), via: d.via}
				if nd.target == nil {
					continue
				}
				for _, x := range d.args {
					nd.args = append(nd.args, ifcVal{label: c.substLabel(x.label, owner, args), fn: c.substCallable(x.fn, owner, args)})
				}
				n.deferred = append(n.deferred, nd)
			}
			out = joinCallable(out, single(n))
		default:
			out = joinCallable(out, single(a))
		}
	}
	return out
}

// resultN is the callable obtained by applying fn n times to unknown args.
func (c *ifcChecker) resultN(fn *ifcCallable, n int) *ifcCallable {
	if n == 0 || fn == nil {
		return fn
	}
	var out *ifcCallable
	for _, a := range fn.alts {
		switch a.kind {
		case fnTop:
			if s := c.summaries[a.name]; s != nil {
				out = joinCallable(out, s.result)
			}
		case fnLambda, fnDeclared:
			out = joinCallable(out, a.result)
		case fnProbe:
			out = joinCallable(out, probe(a.owner, a.idx, a.res+1))
		case fnOpaque:
			out = joinCallable(out, single(a))
		}
	}
	return c.resultN(out, n-1)
}

// --- module fixed point ------------------------------------------------------

// summariseModule iterates the summaries of every module function to a fixed
// point. Each round re-walks every body (silently) against the previous
// round's summaries; summaries only grow and are drawn from a finite set
// (labels and callables written in the module), so the loop terminates. A defensive work budget produces a diagnostic, never a partial summary.
func (c *ifcChecker) summariseModule(file *ast.File) {
	for {
		c.analysisStep()
		changed := false
		for _, fn := range file.Funcs {
			if fn == nil || fn.Body == nil {
				continue
			}
			s, _ := c.walkFunc(c.sigs[fn.Name], true)
			if s.fingerprint() != c.summaries[fn.Name].fingerprint() {
				changed = true
			}
			c.summaries[fn.Name] = s
		}
		if !changed {
			return
		}
	}
}

// walkFunc walks one module function body with its parameters seeded
// symbolically, returning the summary recorded and the body's label.
// Parameters with a positive declared label keep it instead of ρ: Check C at
// every call already bounds their actual arguments by that label.
func (c *ifcChecker) walkFunc(sig *ifcSig, silent bool) (*ifcSummary, Label) {
	key := "f:" + sig.decl.Name
	s := newSummary(key)
	env := make(ifcEnv, len(sig.params))
	for i, p := range sig.params {
		label := p.label
		if LabelEqual(label, LabelBottom()) {
			label = c.rho(key, i)
		}
		env[p.name] = ifcVar{label: label, typ: p.typ,
			fn: joinCallable(probe(key, i, 0), c.types.fromFuncType(p.typ, p.name))}
	}
	prevSilent := c.silent
	c.silent = silent
	c.push(s)
	v := c.valueOf(sig.decl.Body, env)
	c.pop()
	c.silent = prevSilent
	s.result = v.fn
	return s, eraseRho(v.label)
}

// lambdaVal walks a lambda / function literal body once, producing both its
// label (what it returns, its own ρ erased) and its callable (typed-param
// demands plus the summary recorded during the walk).
func (c *ifcChecker) lambdaVal(node ast.Expr, params []*ast.Param, body ast.Expr, env ifcEnv) ifcVal {
	key, ok := c.lamIDs[node]
	if !ok {
		key = fmt.Sprintf("λ%d", len(c.lamIDs)+1)
		c.lamIDs[node] = key
	}
	s := newSummary(key)
	inner := env
	fn := &ifcFn{kind: fnLambda, name: "lambda", owner: key}
	for i, p := range params {
		if p == nil {
			continue
		}
		prev := env[p.Name]
		if prev.label == nil {
			prev.label = LabelBottom()
		}
		rho := c.rho(key, i)
		v := ifcVar{label: LabelJoin(prev.label, rho), typ: prev.typ, fn: joinCallable(prev.fn, probe(key, i, 0))}
		d := ifcDemand{param: p.Name}
		if p.Type != nil {
			// Typed params keep the outer flow label (today's over-approximation)
			// but take their declared type; a positive declared label bounds the
			// argument, so it replaces ρ.
			v.typ = p.Type
			d = demandOf(p.Name, p.Type)
			if len(d.covers) > 0 {
				v.label = prev.label
			}
			v.fn = joinCallable(v.fn, c.types.fromFuncType(p.Type, p.Name))
		}
		fn.params = append(fn.params, d)
		inner = extendEnv(inner, p.Name, v)
	}
	c.push(s)
	v := c.valueOf(body, inner)
	c.pop()
	fn.deferred = s.deferred
	fn.result = v.fn
	return ifcVal{label: c.eraseOwner(v.label, key), fn: single(fn)}
}

func (c *ifcChecker) addErrs(errs []*TypeCheckError) {
	for _, e := range errs {
		k := e.Position + "\x00" + e.Message
		if c.errSeen[k] {
			continue
		}
		c.errSeen[k] = true
		c.errs = append(c.errs, e)
	}
}
