package types

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
)

// ifc_check.go — static information-flow-control (IFC) enforcement.
//
// M-SECRET-EFFECT (M5) wires the M-TAINT-TYPES label lattice (labels.go) into a
// real compile-time check. M-TAINT-TYPES (v0.16.0) shipped the label algebra and
// parsing but never enforced it: CheckSinkRefinement / CheckDeclassify had zero
// callers, so a <secret> value could reach a {not secret} sink with no error.
//
// This pass is intentionally self-contained. It reads only the surface AST and
// emits TypeCheckErrors; it never injects TLabelled types into CoreTI or codegen
// (TLabelled is confined to internal/types + internal/iface, and the backend has
// no case for it). Keeping IFC as a separate post-typecheck analysis therefore
// adds the enforcement with no regression surface in the HM core or the backend.
//
// Scope (v1): intraprocedural sink enforcement with source-label propagation
// through let/match/record/call within a module. A value's label is tracked
// from its source (an explicit <label> annotation, or the secret() builtin) to
// any {not ℓ} sink it reaches in the same call graph. Cross-module sink
// enforcement and context-sensitive interprocedural flow through unannotated
// parameters require label-polymorphic inference, deferred to M-TAINT-TYPES
// Phase 2 (see design_docs .../m-secret-effect-remote-approval.md).

// ifcBuiltinSourceLabels maps builtin/stdlib functions that introduce an IFC
// source label to that label. The canonical source is secret() (M-SECRET-EFFECT):
// its resolved value is labelled <secret>.
//
// This seed is required because the std/secret wrapper's interface type is plain
// `string` — the <secret> label is attached by the _secret_read builtin's return
// type, not the stdlib signature — and this single-module pass does not read
// imported interfaces. A call to secret(...) is therefore treated as producing a
// <secret> value at the call site.
var ifcBuiltinSourceLabels = map[string]string{
	"secret":       "secret",
	"_secret_read": "secret",
}

// CheckModuleIFC runs the static IFC check over a parsed module, returning any
// violations as TypeCheckErrors (empty when the module is clean).
//
// It enforces two rules over the label lattice:
//
//	(A) Sink refinement — a value whose label subsumes ℓ must not be passed to a
//	    parameter declared T{not ℓ}. Checked at every call site.
//	(B) Declassification — a function declaring an *explicit* return label must
//	    not return a value carrying a label that its declared output drops, unless
//	    it declares ! {Declassify}. Checked at every function definition.
//
// Functions without an explicit return label are label-transparent: their result
// label is the join of their body's intrinsic label and their actual argument
// labels (APP-PURE), so ordinary forwarding code needs no annotations and raises
// no violations.
func CheckModuleIFC(file *ast.File) []*TypeCheckError {
	if file == nil {
		return nil
	}
	c := &ifcChecker{
		types:     newIFCTypes(file),
		sigs:      make(map[string]*ifcSig, len(file.Funcs)),
		effLabel:  make(map[string]Label),
		computing: make(map[string]bool),
	}
	for _, fn := range file.Funcs {
		if fn != nil {
			c.sigs[fn.Name] = buildIFCSig(fn)
		}
	}
	// Walk function bodies in source order for deterministic diagnostics.
	for _, fn := range file.Funcs {
		if fn != nil && fn.Body != nil {
			c.checkFunc(c.sigs[fn.Name])
		}
	}
	return c.errs
}

// ifcSig is the IFC-relevant projection of a local function's signature.
type ifcSig struct {
	decl           *ast.FuncDecl
	params         []ifcParam
	returnLabel    Label // declared return label; ⊥ when unannotated
	hasReturnLabel bool  // return type carried an explicit <label>
	declassify     bool  // ! {Declassify} present in the effect row
}

type ifcParam struct {
	name     string
	label    Label    // source label from `x: T<label>` (⊥ if none)
	notLabel string   // sink refinement label from `x: T{not ℓ}` ("" if none)
	typ      ast.Type // the declared parameter type (nil if unannotated)
	hasNot   bool
}

// ifcVar is what the IFC environment knows about a bound name: its flow label
// (labels carried in by the value) and, when the surface AST names it, its
// static type (whose declared labels labelOf adds on every use).
type ifcVar struct {
	label Label
	typ   ast.Type
}

// ifcEnv maps a bound name to its flow label and static type.
type ifcEnv map[string]ifcVar

type ifcChecker struct {
	types     *ifcTypes // module-local type declarations (ifc_static_type.go)
	sigs      map[string]*ifcSig
	errs      []*TypeCheckError
	effLabel  map[string]Label // memoised effective body label per local func
	computing map[string]bool  // cycle guard for the effLabel fixpoint
	silent    bool             // suppress sink diagnostics during effLabel walks
}

func buildIFCSig(fn *ast.FuncDecl) *ifcSig {
	sig := &ifcSig{decl: fn, returnLabel: LabelBottom()}
	for _, p := range fn.Params {
		ip := ifcParam{name: p.Name, label: LabelBottom(), typ: p.Type}
		if lt, ok := p.Type.(*ast.LabelledType); ok {
			if lt.Label != nil {
				ip.label = LabelConst(lt.Label.Name)
			}
			if lt.Refinement != nil {
				ip.notLabel = lt.Refinement.NotLabel
				ip.hasNot = true
			}
		}
		sig.params = append(sig.params, ip)
	}
	if lt, ok := fn.ReturnType.(*ast.LabelledType); ok && lt.Label != nil {
		sig.returnLabel = LabelConst(lt.Label.Name)
		sig.hasReturnLabel = true
	}
	for _, eff := range fn.Effects {
		if eff.Name == "Declassify" {
			sig.declassify = true
		}
	}
	return sig
}

// checkFunc runs Check A (via the body walk) and Check B over one function.
func (c *ifcChecker) checkFunc(sig *ifcSig) {
	env := make(ifcEnv, len(sig.params))
	for _, p := range sig.params {
		env[p.name] = ifcVar{label: p.label, typ: p.typ}
	}
	bodyLabel := c.labelOf(sig.decl.Body, env)

	// Check B: an explicit return label must cover the body's actual label,
	// unless the function is authorised to declassify.
	if sig.hasReturnLabel && !sig.declassify {
		if leaked := labelNotCoveredBy(bodyLabel, sig.returnLabel); leaked != "" {
			c.errs = append(c.errs, newDeclassError(sig.decl, bodyLabel, sig.returnLabel, leaked))
		}
	}
}

// labelOf computes the full IFC label of expr under env: the labels that flowed
// in with the value (flowOf) joined with the labels its static type declares
// (M-IFC-DECLARED-RECORD-LABELS, #1523). As a side effect it performs Check A on
// every call it encounters (unless silent). Each sub-expression is walked once.
func (c *ifcChecker) labelOf(expr ast.Expr, env ifcEnv) Label {
	return LabelJoin(c.flowOf(expr, env), c.types.deepLabel(c.staticType(expr, env)))
}

// flowOf computes the label carried in by the VALUE of expr — the pre-#1523
// walk. labelOf adds the declared part; keeping the two apart is what makes
// field projection precise: s.user does not inherit s.code's declared label.
func (c *ifcChecker) flowOf(expr ast.Expr, env ifcEnv) Label {
	switch e := expr.(type) {
	case nil:
		return LabelBottom()
	case *ast.Identifier:
		if v, ok := env[e.Name]; ok {
			return v.label
		}
		return LabelBottom()
	case *ast.Literal:
		return LabelBottom()
	case *ast.BinaryOp:
		return LabelJoin(c.labelOf(e.Left, env), c.labelOf(e.Right, env))
	case *ast.UnaryOp:
		return c.labelOf(e.Expr, env)
	case *ast.FuncCall:
		return c.labelOfCall(e, env)
	case *ast.Let:
		return c.labelOf(e.Body, extendEnv(env, e.Name, c.bindValue(e.Value, e.Type, env)))
	case *ast.LetRec:
		return c.labelOf(e.Body, extendEnv(env, e.Name, c.bindValue(e.Value, e.Type, env)))
	case *ast.Block:
		return c.walkBlock(e, env, c.labelOf)
	case *ast.If:
		c.labelOf(e.Condition, env)
		return LabelJoin(c.labelOf(e.Then, env), c.labelOf(e.Else, env))
	case *ast.Match:
		// Every pattern variable gets the scrutinee's FULL label (flow and
		// declared): destructuring is conservative, not field-precise.
		scrut := c.labelOf(e.Expr, env)
		result := LabelBottom()
		for _, cs := range e.Cases {
			armEnv := env
			for _, name := range patternVars(cs.Pattern) {
				armEnv = extendEnv(armEnv, name, ifcVar{label: scrut})
			}
			if cs.Guard != nil {
				c.labelOf(cs.Guard, armEnv)
			}
			result = LabelJoin(result, c.labelOf(cs.Body, armEnv))
		}
		return result
	case *ast.Record:
		result := LabelBottom()
		for _, f := range e.Fields {
			result = LabelJoin(result, c.labelOf(f.Value, env))
		}
		return result
	case *ast.RecordUpdate:
		result := c.labelOf(e.Base, env)
		for _, f := range e.Fields {
			result = LabelJoin(result, c.labelOf(f.Value, env))
		}
		return result
	case *ast.RecordAccess:
		// Field-precise when the field's declared type is known: the field gets
		// the record's FLOW label here, and labelOf adds deepLabel(field type)
		// — not the declared labels of the record's other fields. Otherwise a
		// field inherits the record's full join label (conservative).
		if c.staticType(e, env) != nil {
			return c.flowOf(e.Record, env)
		}
		return c.labelOf(e.Record, env)
	case *ast.List:
		return c.joinElems(e.Elements, env)
	case *ast.Array:
		return c.joinElems(e.Elements, env)
	case *ast.Tuple:
		return c.joinElems(e.Elements, env)
	case *ast.Lambda:
		// A closure carries the label of what it RETURNS.
		//
		// This used to return LabelBottom() on the reasoning that "a closure
		// value carries no label". True of the function value itself, but it
		// made the guarantee bypassable in three tokens, because applying the
		// closure then yielded an unlabelled result:
		//
		//   let f = \u. s in sink(f(0))   -- s : string<secret>, sink : {not secret}
		//
		// Propagating the body's label is a conservative over-approximation: it
		// can label a closure whose result a caller never uses, which blocks more
		// than strictly necessary. For a security control that is the correct
		// direction to err — over-approximating rejects safe programs, while
		// under-approximating admits leaks silently.
		return c.labelOf(e.Body, typedParams(env, e.Params))
	case *ast.FuncLit:
		return c.labelOf(e.Body, typedParams(env, e.Params))
	default:
		return LabelBottom()
	}
}

// walkBlock threads block-statement lets through a block { s1; s2; ...; result }
// and returns `last` applied to the result expression. A Let or LetRec parsed
// without `in` (Body == nil) scopes over the remainder of the block, because
// elaboration turns it into nested Core lets.
func (c *ifcChecker) walkBlock(b *ast.Block, env ifcEnv, last func(ast.Expr, ifcEnv) Label) Label {
	result := LabelBottom()
	cur := env
	for _, sub := range b.Exprs {
		switch s := sub.(type) {
		case *ast.Let:
			if s.Body == nil {
				cur = extendEnv(cur, s.Name, c.bindValue(s.Value, s.Type, cur))
				result = LabelBottom()
				continue
			}
		case *ast.LetRec:
			if s.Body == nil {
				cur = extendEnv(cur, s.Name, c.bindValue(s.Value, s.Type, cur))
				result = LabelBottom()
				continue
			}
		}
		result = last(sub, cur)
	}
	return result
}

// typedParams binds the ANNOTATED parameters of a lambda / function literal to
// their declared type, keeping whatever flow label an outer binding of the same
// name had (unannotated parameters are not rebound — today's over-approximation).
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

// bindValue computes what a let binding records: the annotation (or, when
// unannotated, the value's own static type) and the hand-off label of the value
// into that type.
func (c *ifcChecker) bindValue(value ast.Expr, annot ast.Type, env ifcEnv) ifcVar {
	t := annot
	if t == nil {
		t = c.staticType(value, env)
	}
	return ifcVar{label: c.handOff(value, t, env), typ: t}
}

// handOff is the label a value keeps when it moves into a position declared as
// type t (a let, a function body against its return type). When the value's
// static type is written exactly as t, only its FLOW label is kept: the
// receiving position re-derives the declared part from t itself. A record
// literal is handed off field by field. In every other case — including t
// unknown, or an annotation whose labels sit elsewhere than the value's (HM
// strips labels, so that type-checks) — the value keeps its FULL label, so an
// annotation can never lower a label.
func (c *ifcChecker) handOff(e ast.Expr, t ast.Type, env ifcEnv) Label {
	if t == nil {
		return c.labelOf(e, env)
	}
	switch x := e.(type) {
	case *ast.Record:
		if c.types.recordTypeOf(t) != nil {
			result := LabelBottom()
			for _, f := range x.Fields {
				result = LabelJoin(result, c.handOff(f.Value, c.types.fieldType(t, f.Name), env))
			}
			return result
		}
	case *ast.Block:
		return c.walkBlock(x, env, func(sub ast.Expr, cur ifcEnv) Label { return c.handOff(sub, t, cur) })
	case *ast.Let:
		if x.Body != nil {
			return c.handOff(x.Body, t, extendEnv(env, x.Name, c.bindValue(x.Value, x.Type, env)))
		}
	case *ast.If:
		c.labelOf(x.Condition, env)
		return LabelJoin(c.handOff(x.Then, t, env), c.handOff(x.Else, t, env))
	}
	if sameType(c.staticType(e, env), t) {
		return c.flowOf(e, env)
	}
	return c.labelOf(e, env)
}

func (c *ifcChecker) joinElems(elems []ast.Expr, env ifcEnv) Label {
	result := LabelBottom()
	for _, el := range elems {
		result = LabelJoin(result, c.labelOf(el, env))
	}
	return result
}

// labelOfCall computes a call's result label and runs Check A on the callee's
// sink parameters.
func (c *ifcChecker) labelOfCall(call *ast.FuncCall, env ifcEnv) Label {
	name := calleeName(call.Func)
	sig, isLocal := c.sigs[name]

	// Each argument is walked ONCE. argLabels are full labels (used by Check A
	// and by unknown callees); passed are the hand-off labels into a local
	// callee's declared parameter types (the summary re-derives the declared
	// part from the parameter type, so a type-matched argument passes only its
	// flow label — see handOff).
	argLabels := make([]Label, len(call.Args))
	passed := make([]Label, len(call.Args))
	for i, a := range call.Args {
		flow := c.flowOf(a, env)
		st := c.staticType(a, env)
		argLabels[i] = LabelJoin(flow, c.types.deepLabel(st))
		passed[i] = argLabels[i]
		if isLocal && i < len(sig.params) && sameType(st, sig.params[i].typ) {
			passed[i] = flow
		}
	}

	if isLocal {
		// Check A: sink refinements on the callee's parameters.
		if !c.silent {
			for i, p := range sig.params {
				if p.hasNot && i < len(argLabels) {
					if CheckSinkLabel(argLabels[i], p.notLabel) != nil {
						c.errs = append(c.errs, newSinkError(call, p.name, p.notLabel, argLabels[i]))
					}
				}
			}
		}
		return c.calleeResultLabel(sig, passed)
	}
	if src, ok := ifcBuiltinSourceLabels[name]; ok {
		return LabelConst(src)
	}
	// Unknown or imported callee: transparent — propagate the join of the args
	// AND of the callee expression itself.
	//
	// The callee's own label is what makes a local closure safe. `f` in
	// `let f = \u. s in f(0)` has no entry in c.sigs (it is a binding, not a
	// declaration), so this fallback is the path it takes; joining only the
	// argument labels discarded the captured <secret> entirely and made the
	// guarantee bypassable in three tokens. c.labelOf on the callee reads the
	// binding out of env, where the Let case put it.
	return LabelJoin(c.labelOf(call.Func, env), joinLabels(argLabels))
}

// calleeResultLabel computes the label produced by calling a known local function.
func (c *ifcChecker) calleeResultLabel(sig *ifcSig, argLabels []Label) Label {
	if sig.declassify {
		// Declassification authoritatively relabels the result to the declared
		// output (typically ⊥/clean), breaking the taint chain.
		return sig.returnLabel
	}
	if sig.hasReturnLabel {
		return sig.returnLabel
	}
	// Transparent: the body's intrinsic label joined with the labels flowing in
	// through the actual arguments.
	return LabelJoin(c.effectiveBodyLabel(sig.decl.Name), joinLabels(argLabels))
}

// effectiveBodyLabel computes (and memoises) the intrinsic label a transparent
// local function's body produces with parameters' flow labels seeded at ⊥ —
// capturing taint introduced inside the body (e.g. a secret() call) independent
// of arguments. Parameters keep their declared TYPE, so a getter that projects
// a declared-labelled field (`getRaw(c: RawCode) -> string = c.raw`) yields that
// field's label, while one projecting an unlabelled sibling stays clean. The
// body is handed off against the declared return type; the caller's labelOf
// adds that type's declared labels. A cycle guard makes mutual recursion
// converge conservatively.
func (c *ifcChecker) effectiveBodyLabel(name string) Label {
	if l, ok := c.effLabel[name]; ok {
		return l
	}
	sig, ok := c.sigs[name]
	if !ok || sig.decl.Body == nil {
		return LabelBottom()
	}
	if c.computing[name] {
		return LabelBottom() // break recursion conservatively
	}
	c.computing[name] = true
	prevSilent := c.silent
	c.silent = true // label-only walk; sinks are checked when this func is checkFunc'd
	env := make(ifcEnv, len(sig.params))
	for _, p := range sig.params {
		env[p.name] = ifcVar{label: LabelBottom(), typ: p.typ}
	}
	l := c.handOff(sig.decl.Body, sig.decl.ReturnType, env)
	c.silent = prevSilent
	c.computing[name] = false
	c.effLabel[name] = l
	return l
}

// --- helpers ---

func calleeName(fn ast.Expr) string {
	if id, ok := fn.(*ast.Identifier); ok {
		return id.Name
	}
	return ""
}

func extendEnv(env ifcEnv, name string, v ifcVar) ifcEnv {
	next := make(ifcEnv, len(env)+1)
	for k, val := range env {
		next[k] = val
	}
	next[name] = v
	return next
}

func joinLabels(ls []Label) Label {
	result := LabelBottom()
	for _, l := range ls {
		result = LabelJoin(result, l)
	}
	return result
}

// patternVars collects every variable name a pattern binds (an Identifier in
// pattern position). Used to seed match-arm environments with the scrutinee's
// label so destructuring does not silently erase taint.
func patternVars(p ast.Pattern) []string {
	switch pat := p.(type) {
	case *ast.Identifier:
		return []string{pat.Name}
	case *ast.ConsPattern:
		return append(patternVars(pat.Head), patternVars(pat.Tail)...)
	case *ast.ListPattern:
		var vs []string
		for _, e := range pat.Elements {
			vs = append(vs, patternVars(e)...)
		}
		if pat.Rest != nil {
			vs = append(vs, patternVars(pat.Rest)...)
		}
		return vs
	case *ast.TuplePattern:
		var vs []string
		for _, e := range pat.Elements {
			vs = append(vs, patternVars(e)...)
		}
		return vs
	case *ast.RecordPattern:
		var vs []string
		for _, f := range pat.Fields {
			vs = append(vs, patternVars(f.Pattern)...)
		}
		return vs
	case *ast.ConstructorPattern:
		var vs []string
		for _, sub := range pat.Patterns {
			vs = append(vs, patternVars(sub)...)
		}
		return vs
	default: // WildcardPattern, Literal — no bindings
		return nil
	}
}

// labelNotCoveredBy returns the name of a constituent label present in `have`
// but absent from `declared` (i.e. taint the declaration hides), or "" when
// `declared` covers `have`.
func labelNotCoveredBy(have, declared Label) string {
	for _, part := range normalisedParts(have) {
		if _, isBot := part.(labelBottom); isBot {
			continue
		}
		if !LabelSubsumes(declared, part) {
			return labelDisplayName(part)
		}
	}
	return ""
}

func labelDisplayName(l Label) string {
	if lc, ok := l.(labelConst); ok {
		return lc.name
	}
	return l.String()
}

func newSinkError(call *ast.FuncCall, paramName, notLabel string, argLabel Label) *TypeCheckError {
	return &TypeCheckError{
		Kind:     SinkRefinementError,
		Position: call.Pos.String(),
		Message: fmt.Sprintf(
			"information-flow violation: value labelled %s reaches parameter %q which forbids it (string{not %s})",
			argLabel, paramName, notLabel),
		Suggestion: fmt.Sprintf(
			"declassify the value through a function declaring ! {Declassify} before passing it to a {not %s} sink",
			notLabel),
	}
}

func newDeclassError(fn *ast.FuncDecl, bodyLabel, declared Label, leaked string) *TypeCheckError {
	return &TypeCheckError{
		Kind:     DeclassifyRequiredError,
		Position: fn.Pos.String(),
		Message: fmt.Sprintf(
			"information-flow violation: function %q returns a value labelled %s but declares return label %s, hiding <%s>",
			fn.Name, bodyLabel, declared, leaked),
		Suggestion: "add ! {Declassify} to the effect row to authorise lowering this label, or widen the declared return label",
	}
}
