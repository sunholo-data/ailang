package types

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
)

// ifc_summary_local.go — per-module function summaries for the callable
// domain (M-IFC-FUNCTION-VALUE-SINKS M2, #1719).
//
// Every function-like entity (module function, lambda, function literal,
// written function type) has an ifcFun summary:
//
//   - dem[i]: what the entity requires of argument i. Declared demands come
//     from `{not ℓ}` / `<ℓ>` on the parameter; INFERRED demands (M2b) come from
//     walking the body with parameter i seeded as a label variable ρ_i: when
//     an argument labelled ρ_i reaches a demand, parameter i inherits it
//     (`wrap(y) = gate(y)` forbids <email> on y).
//   - symbolic demands (M2a, applied-at): when the body applies a function
//     PARAMETER (a probe) to an argument labelled ρ_i, parameter i gets "what
//     probe p requires at position k". At each call the probe is bound to the
//     actual function argument, so `apply1(gate, x)` checks x against gate and
//     `apply1(upper, x)` checks nothing.
//   - intrinsic: labels the body applies a probe to on its own
//     (`h(f, x) = f(fromInbox(..))`), checked against the bound function at
//     each call.
//   - ret: the function atoms the body may return (R8, R9).
//
// The module flow table records, per probe, every function atom passed to it
// anywhere. It resolves a probe that is out of scope — inside a closure that
// escaped its defining call — context-insensitively (an over-approximation).
//
// All tables are finite (labels, demands and atoms written in the module) and
// only grow, so the fixed point over the module terminates; each round is one
// walk of every body. ρ never leaves this machinery: it is erased from every
// result label, Check B input and diagnostic.

// maxResolveDepth bounds resolution through chains of probes and summaries.
const maxResolveDepth = 16

// fallbackArity is how many argument positions a function of unknown arity
// (a probe handed to an imported function) is assumed to take.
const fallbackArity = 4

type ifcFun struct {
	key       string
	name      string
	params    []string
	dem       []map[string]*ifcDemand
	intrinsic map[string]*ifcIntrinsic
	ret       *ifcCallable
}

// ifcIntrinsic: the body applies its own parameter idx, at position pos, to a
// value carrying the concrete label.
type ifcIntrinsic struct {
	idx, pos int
	label    Label
	via      []string
}

func newFun(key, name string, params []string) *ifcFun {
	f := &ifcFun{key: key, name: name, params: params, intrinsic: map[string]*ifcIntrinsic{}}
	f.dem = make([]map[string]*ifcDemand, len(params))
	for i := range f.dem {
		f.dem[i] = map[string]*ifcDemand{}
	}
	return f
}

func (c *ifcChecker) addDemand(f *ifcFun, pos int, d *ifcDemand) {
	if f == nil || pos >= len(f.dem) {
		return
	}
	k := d.key()
	if _, ok := f.dem[pos][k]; ok {
		return
	}
	f.dem[pos][k] = d
	c.version++
}

func sortedDemands(m map[string]*ifcDemand) []*ifcDemand {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]*ifcDemand, len(keys))
	for i, k := range keys {
		out[i] = m[k]
	}
	return out
}

func (c *ifcChecker) addIntrinsic(f *ifcFun, idx, pos int, l Label, via []string) {
	if f == nil || LabelEqual(l, LabelBottom()) {
		return
	}
	k := fmt.Sprintf("%d/%d", idx, pos)
	in, ok := f.intrinsic[k]
	if !ok {
		f.intrinsic[k] = &ifcIntrinsic{idx: idx, pos: pos, label: l, via: append([]string(nil), via...)}
		c.version++
		return
	}
	if j := LabelJoin(in.label, l); !LabelEqual(j, in.label) {
		in.label = j
		c.version++
	}
}

func (c *ifcChecker) addFlow(key string, fn *ifcCallable) {
	fn = relevant(fn)
	if fn == nil {
		return
	}
	j := joinCallable(c.flow[key], fn)
	if j != c.flow[key] {
		c.flow[key] = j
		c.version++
	}
}

func (c *ifcChecker) addRet(f *ifcFun, fn *ifcCallable) {
	fn = relevant(fn)
	if f == nil || fn == nil {
		return
	}
	if j := joinCallable(f.ret, fn); j != f.ret {
		f.ret = j
		c.version++
	}
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

// eraseRho drops every label variable: the concrete part of a label.
func eraseRho(l Label) Label {
	if !hasLabelVar(l) {
		return l
	}
	out := LabelBottom()
	for _, p := range normalisedParts(l) {
		if _, ok := p.(labelVar); !ok {
			out = LabelJoin(out, p)
		}
	}
	return out
}

// eraseOwner drops the label variables of one entity (a lambda's own params
// when its label leaves the lambda).
func (c *ifcChecker) eraseOwner(l Label, owner string) Label {
	if !hasLabelVar(l) {
		return l
	}
	out := LabelBottom()
	for _, p := range normalisedParts(l) {
		if v, ok := p.(labelVar); ok && c.rhos[v.name].owner == owner {
			continue
		}
		out = LabelJoin(out, p)
	}
	return out
}

// activeRhos lists the label variables in l that belong to entities whose
// bodies are being walked.
func (c *ifcChecker) activeRhos(l Label) []rhoRef {
	if !hasLabelVar(l) {
		return nil
	}
	var out []rhoRef
	for _, p := range normalisedParts(l) {
		if v, ok := p.(labelVar); ok {
			if r := c.rhos[v.name]; c.isActive(r.owner) {
				out = append(out, r)
			}
		}
	}
	return out
}

func (c *ifcChecker) isActive(owner string) bool {
	for _, k := range c.owners {
		if k == owner {
			return true
		}
	}
	return false
}

// --- application -------------------------------------------------------------

// applyCallable checks an application of the callee's atoms to args (Checks
// A/C against every atom's demands, declared, inferred and symbolic) and
// returns the atoms of the result. A nil callee is unknown: fallback.
func (c *ifcChecker) applyCallable(callee *ifcCallable, args []ifcVal, at *ast.FuncCall, calleeLabel Label, via []string, depth int) *ifcCallable {
	if callee == nil {
		return c.applyOpaque(args, at, calleeLabel, via, depth)
	}
	var result *ifcCallable
	for _, a := range callee.sorted() {
		switch a.kind {
		case atomFun:
			result = joinCallable(result, c.applyFun(c.funs[a.fun], args, at, via, depth))
		case atomProbe:
			result = joinCallable(result, c.applyProbe(a, args, at, calleeLabel, via, depth))
		case atomOpaque:
			result = joinCallable(result, c.applyOpaque(args, at, calleeLabel, via, depth))
		case atomCtor:
			for _, arg := range args {
				result = joinCallable(result, relevant(arg.fn))
			}
		}
	}
	return result
}

func (c *ifcChecker) applyFun(f *ifcFun, args []ifcVal, at *ast.FuncCall, via []string, depth int) *ifcCallable {
	if f == nil || depth > maxResolveDepth {
		return nil
	}
	for k := 0; k < len(f.params) && k < len(args); k++ {
		for _, d := range sortedDemands(f.dem[k]) {
			c.checkDemand(f, d, args[k], args, at, via, depth)
		}
		c.addFlow(probeKey(f.key, k), args[k].fn)
	}
	for _, k := range sortedIntrinsicKeys(f) {
		in := f.intrinsic[k]
		if in.idx < len(args) {
			hops := append(append(append([]string(nil), via...), f.name), in.via...)
			c.applyAtPos(args[in.idx].fn, in.pos, ifcVal{label: in.label}, at, hops, depth+1)
		}
	}
	return c.substRet(f, args)
}

func sortedIntrinsicKeys(f *ifcFun) []string {
	keys := make([]string, 0, len(f.intrinsic))
	for k := range f.intrinsic {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// substRet is f's result atoms with f's own probes bound to the actual args.
func (c *ifcChecker) substRet(f *ifcFun, args []ifcVal) *ifcCallable {
	if f.ret == nil {
		return nil
	}
	var out *ifcCallable
	for k, a := range f.ret.atoms {
		if a.kind == atomProbe && a.owner == f.key {
			if a.idx < len(args) {
				out = joinCallable(out, relevant(args[a.idx].fn))
			}
			continue
		}
		out = joinCallable(out, &ifcCallable{atoms: map[string]ifcAtom{k: a}})
	}
	return out
}

// checkDemand checks one demand of entity f against the argument at its
// position. args binds f's own probes when f is the function being applied
// (nil when f is applied by an unknown caller).
func (c *ifcChecker) checkDemand(f *ifcFun, d *ifcDemand, arg ifcVal, args []ifcVal, at *ast.FuncCall, via []string, depth int) {
	full := via
	if d.symbolic() || d.leaf != f.name || len(d.via) > 0 {
		full = append(append(append([]string(nil), via...), f.name), d.via...)
	}
	if d.symbolic() {
		if d.owner == f.key && args != nil {
			if d.idx < len(args) {
				c.applyAtPos(args[d.idx].fn, d.pos, arg, at, full, depth+1)
			}
			return
		}
		c.probeApplied(d.owner, d.idx, d.pos, arg, at, full, depth+1)
		return
	}
	concrete := eraseRho(arg.label)
	if !c.silent && d.violation(concrete) != "" {
		c.addErrs([]*TypeCheckError{demandError(at, d, concrete, full)})
	}
	for _, r := range c.activeRhos(arg.label) {
		cp := *d
		cp.via = full
		c.addDemand(c.funs[r.owner], r.idx, &cp)
	}
}

// applyAtPos: the functions in fn are applied with arg at position pos (other
// arguments unknown).
func (c *ifcChecker) applyAtPos(fn *ifcCallable, pos int, arg ifcVal, at *ast.FuncCall, via []string, depth int) {
	if fn == nil || depth > maxResolveDepth {
		return
	}
	for _, a := range fn.sorted() {
		switch a.kind {
		case atomFun:
			g := c.funs[a.fun]
			if g == nil || pos >= len(g.params) {
				continue
			}
			for _, d := range sortedDemands(g.dem[pos]) {
				c.checkDemand(g, d, arg, nil, at, via, depth)
			}
			c.addFlow(probeKey(g.key, pos), arg.fn)
			for _, k := range sortedIntrinsicKeys(g) {
				in := g.intrinsic[k]
				bound := c.flowStar(probeKey(g.key, in.idx))
				if in.idx == pos {
					bound = arg.fn
				}
				hops := append(append(append([]string(nil), via...), g.name), in.via...)
				c.applyAtPos(bound, in.pos, ifcVal{label: in.label}, at, hops, depth+1)
			}
		case atomProbe:
			c.probeApplied(a.owner, a.idx, pos, arg, at, via, depth)
		}
	}
}

// probeApplied: parameter idx of entity owner is applied with arg at pos. While
// the owner's body is being walked this is recorded on the owner's summary
// (symbolic demands for ρ, intrinsic labels for concrete data); otherwise the
// probe is resolved through the module flow table.
func (c *ifcChecker) probeApplied(owner string, idx, pos int, arg ifcVal, at *ast.FuncCall, via []string, depth int) {
	if depth > maxResolveDepth {
		return
	}
	if !c.isActive(owner) {
		c.applyAtPos(c.flowStar(probeKey(owner, idx)), pos, arg, at, via, depth+1)
		return
	}
	for _, r := range c.activeRhos(arg.label) {
		c.addDemand(c.funs[r.owner], r.idx, &ifcDemand{probe: probeKey(owner, idx), owner: owner, idx: idx, pos: pos, via: via})
	}
	c.addIntrinsic(c.funs[owner], idx, pos, eraseRho(arg.label), via)
}

// applyProbe applies a function parameter to actual arguments.
func (c *ifcChecker) applyProbe(a ifcAtom, args []ifcVal, at *ast.FuncCall, calleeLabel Label, via []string, depth int) *ifcCallable {
	for k, arg := range args {
		c.probeApplied(a.owner, a.idx, k, arg, at, via, depth)
	}
	// Function arguments handed to a function we cannot see yet: assume it
	// applies them to everything else it was given (as for imported callees).
	c.fallback(args, at, calleeLabel, via, depth)
	var out *ifcCallable
	for _, b := range c.flowStar(probeKey(a.owner, a.idx)).sorted() {
		if g := c.funs[b.fun]; g != nil {
			out = joinCallable(out, g.ret)
		}
	}
	return out
}

// applyOpaque is the conservative fallback for an imported or unknown callee
// (design F2): every function argument is assumed applied to the join of the
// other arguments' labels and the callee's own label.
func (c *ifcChecker) applyOpaque(args []ifcVal, at *ast.FuncCall, calleeLabel Label, via []string, depth int) *ifcCallable {
	result := opaqueCallable
	for _, arg := range args {
		result = joinCallable(result, relevant(arg.fn))
	}
	c.fallback(args, at, calleeLabel, via, depth)
	return result
}

func (c *ifcChecker) fallback(args []ifcVal, at *ast.FuncCall, calleeLabel Label, via []string, depth int) {
	for j, arg := range args {
		fn := relevant(arg.fn)
		if fn == nil {
			continue
		}
		other := calleeLabel
		for k, b := range args {
			if k != j {
				other = LabelJoin(other, b.label)
			}
		}
		for _, a := range fn.sorted() {
			n := fallbackArity
			if a.kind == atomFun && c.funs[a.fun] != nil {
				n = len(c.funs[a.fun].params)
			}
			for pos := 0; pos < n; pos++ {
				c.applyAtPos(single(a), pos, ifcVal{label: other}, at, via, depth+1)
			}
		}
	}
}

// flowStar resolves a probe to the function atoms passed to it anywhere in the
// module, following probes passed on to probes.
func (c *ifcChecker) flowStar(key string) *ifcCallable {
	var out *ifcCallable
	seen := map[string]bool{}
	stack := []string{key}
	for len(stack) > 0 {
		k := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[k] {
			continue
		}
		seen[k] = true
		for _, a := range c.flow[k].sorted() {
			if a.kind == atomProbe {
				stack = append(stack, probeKey(a.owner, a.idx))
				continue
			}
			out = joinCallable(out, single(a))
		}
	}
	return out
}

// --- entities ----------------------------------------------------------------

// fromSig is the callable of a module function referenced as a value.
func fromSig(sig *ifcSig) *ifcCallable { return funAtom("f:" + sig.decl.Name) }

func (c *ifcChecker) registerTops(file *ast.File) {
	for _, fn := range file.Funcs {
		if fn == nil {
			continue
		}
		sig := c.sigs[fn.Name]
		names := make([]string, len(sig.params))
		for i, p := range sig.params {
			names[i] = p.name
		}
		f := newFun("f:"+fn.Name, fn.Name, names)
		for i, p := range sig.params {
			if !LabelEqual(p.label, LabelBottom()) {
				f.dem[i]["decl-cover"] = &ifcDemand{cover: p.label, leaf: fn.Name, param: p.name, typ: p.typ}
			}
			if p.hasNot {
				f.dem[i]["decl-not"] = &ifcDemand{forbid: p.notLabel, leaf: fn.Name, param: p.name, typ: p.typ}
			}
		}
		c.funs[f.key] = f
	}
}

// fromFuncType is the callable a written function type promises: an entity
// carrying the demands of its labelled or refined parameters (and of its
// result, if that is a function type). nil when the type declares none.
func (c *ifcChecker) fromFuncType(t ast.Type, name string) *ifcCallable {
	ft := c.types.funcTypeOf(t)
	if ft == nil || !c.typeHasDemands(ft) {
		return nil
	}
	key, ok := c.declIDs[ft]
	if !ok {
		key = fmt.Sprintf("d%d", len(c.declIDs)+1)
		c.declIDs[ft] = key
		names := make([]string, len(ft.Params))
		for i := range names {
			names[i] = fmt.Sprintf("#%d", i+1)
		}
		f := newFun(key, name, names)
		for i, p := range ft.Params {
			for _, d := range demandsOfType(name, names[i], p) {
				f.dem[i][d.key()] = d
			}
		}
		f.ret = c.fromFuncType(ft.Return, name+"(..)")
		c.funs[key] = f
	}
	return funAtom(key)
}

// --- module fixed point ------------------------------------------------------

// summariseModule walks every module function body (silently) until no
// summary, flow or result table grows. Every table is drawn from a finite set
// and only grows, so this terminates; the bound is a backstop.
func (c *ifcChecker) summariseModule(file *ast.File) {
	for round := 0; round < 64; round++ {
		before := c.version
		for _, fn := range file.Funcs {
			if fn != nil && fn.Body != nil {
				c.walkFunc(c.sigs[fn.Name], true)
			}
		}
		if c.version == before {
			return
		}
	}
}

// walkFunc walks one module function body, its parameters seeded as their
// declared label — or ρ when unlabelled (a positive declared label already
// bounds every actual argument through Check C) — and as probes.
func (c *ifcChecker) walkFunc(sig *ifcSig, silent bool) Label {
	key := "f:" + sig.decl.Name
	env := make(ifcEnv, len(sig.params))
	for i, p := range sig.params {
		v := ifcVar{label: p.label, typ: p.typ}
		if c.callables {
			if LabelEqual(v.label, LabelBottom()) {
				v.label = c.rho(key, i)
			}
			v.fn = joinCallable(probeAtom(key, i), c.fromFuncType(p.typ, p.name))
		}
		env[p.name] = v
	}
	prevSilent := c.silent
	c.silent = silent
	c.owners = append(c.owners, key)
	v := c.valueOf(sig.decl.Body, env)
	c.owners = c.owners[:len(c.owners)-1]
	c.silent = prevSilent
	c.addRet(c.funs[key], v.fn)
	return eraseRho(v.label)
}

// lambdaVal walks a lambda / function literal body once, producing both its
// label (what it returns, its own ρ erased) and its callable (the entity for
// this AST node, whose summary the walk updates).
func (c *ifcChecker) lambdaVal(node ast.Expr, params []*ast.Param, body ast.Expr, env ifcEnv) ifcVal {
	if !c.callables {
		return ifcVal{label: c.labelOf(body, typedParams(env, params))}
	}
	key, ok := c.lamIDs[node]
	if !ok {
		key = fmt.Sprintf("λ%d", len(c.lamIDs)+1)
		c.lamIDs[node] = key
		names := make([]string, len(params))
		for i, p := range params {
			if p != nil {
				names[i] = p.Name
			}
		}
		f := newFun(key, "lambda", names)
		for i, p := range params {
			if p != nil && p.Type != nil {
				for _, d := range demandsOfType("lambda", p.Name, p.Type) {
					f.dem[i][d.key()] = d
				}
			}
		}
		c.funs[key] = f
	}
	inner := env
	for i, p := range params {
		if p == nil {
			continue
		}
		prev := env[p.Name]
		if prev.label == nil {
			prev.label = LabelBottom()
		}
		v := ifcVar{label: LabelJoin(prev.label, c.rho(key, i)), typ: prev.typ, fn: joinCallable(prev.fn, probeAtom(key, i))}
		if p.Type != nil {
			// Typed params keep the outer flow label (the label walk's long-
			// standing over-approximation) but take their declared type; a
			// positive declared label bounds the argument, so it replaces ρ.
			v.typ = p.Type
			if lt, ok := p.Type.(*ast.LabelledType); ok && lt.Label != nil {
				v.label = prev.label
			}
			v.fn = joinCallable(v.fn, c.fromFuncType(p.Type, p.Name))
		}
		inner = extendEnv(inner, p.Name, v)
	}
	c.owners = append(c.owners, key)
	v := c.valueOf(body, inner)
	c.owners = c.owners[:len(c.owners)-1]
	c.addRet(c.funs[key], v.fn)
	return ifcVal{label: c.eraseOwner(v.label, key), fn: funAtom(key)}
}

// typedParams binds the ANNOTATED parameters of a lambda / function literal to
// their declared type, keeping whatever flow label an outer binding of the same
// name had (unannotated parameters are not rebound — the over-approximation the
// label walk has always made). Used when the module has no demand sources.
func typedParams(env ifcEnv, params []*ast.Param) ifcEnv {
	out := env
	for _, p := range params {
		if p == nil || p.Type == nil {
			continue
		}
		prev := LabelBottom()
		if v, ok := env[p.Name]; ok {
			prev = v.label
		}
		out = extendEnv(out, p.Name, ifcVar{label: prev, typ: p.Type})
	}
	return out
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
