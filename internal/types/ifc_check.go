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
// Scope: per-module sink enforcement with source-label propagation
// through let/match/record/call within a module. A value's label is tracked
// from its source (an explicit <label> annotation, or the secret() builtin) to
// any {not ℓ} sink it reaches in the same call graph. Since #1719
// (M-IFC-FUNCTION-VALUE-SINKS) a sink's requirements also travel with the
// function VALUE, and flow through unannotated parameters into a sink is
// inferred per module function (ρ-seeded summaries, ifc_summary_local.go).
// Cross-module sink enforcement is still open (#1134, M-IFC-CROSS-MODULE).

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
// It enforces three rules over the label lattice:
//
//	(A) Sink refinement — a value whose label subsumes ℓ must not be passed to a
//	    parameter declared T{not ℓ}. Checked at every call site.
//	(B) Declassification — a function declaring an *explicit* return label must
//	    not return a value carrying a label that its declared output drops, unless
//	    its Declassify authority covers every hidden label. Checked at every function definition.
//
// (C) Positive parameter labels must cover every argument constituent at local calls.
//
// Functions without an explicit return label are label-transparent: their result
// label is the join of their body's intrinsic label and their actual argument
// labels (APP-PURE), so ordinary forwarding code needs no annotations and raises
// no violations.
//
// Checks A and C run at EVERY application, against the callable the callee
// VALUE carries — however that value was reached (M-IFC-FUNCTION-VALUE-SINKS,
// #1719; ifc_callable.go, ifc_summary_local.go).
func CheckModuleIFC(file *ast.File) []*TypeCheckError {
	if file == nil {
		return nil
	}
	return guardIFCAnalysis(func() []*TypeCheckError { return newIFCChecker(file).run(file) })
}

func newIFCChecker(file *ast.File) *ifcChecker {
	c := &ifcChecker{
		types:             newIFCTypes(file),
		sigs:              make(map[string]*ifcSig, len(file.Funcs)),
		effLabel:          make(map[string]Label),
		computing:         make(map[string]bool),
		summaries:         make(map[string]*ifcSummary, len(file.Funcs)),
		rhos:              make(map[string]rhoRef),
		lamIDs:            make(map[ast.Expr]string),
		errSeen:           make(map[string]bool),
		replaying:         make(map[string]bool),
		analysisRemaining: ifcAnalysisWork,
	}
	for _, fn := range file.Funcs {
		if fn != nil {
			c.sigs[fn.Name] = buildIFCSig(fn)
		}
	}
	return c
}

func (c *ifcChecker) run(file *ast.File) []*TypeCheckError {
	// Summaries first (silent), so every application below sees what each
	// module function does with the function values and labels it receives.
	if moduleHasIFCRequirements(file) {
		c.summariseModule(file)
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
	decl             *ast.FuncDecl
	params           []ifcParam
	returnLabel      Label    // declared return label; ⊥ when unannotated
	hasReturnLabel   bool     // return type carried an explicit <label>
	authorizedLabels []string // none=[], bare=["*"], scoped=[label] (single label in v1)
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
	fn    *ifcCallable // what applying this value requires (ifc_callable.go)
}

// ifcEnv maps a bound name to its flow label, static type and callable.
type ifcEnv map[string]ifcVar

type ifcChecker struct {
	exportSummaries bool // M1 producer/closure label facts; ordinary caller policy stays unchanged
	imports         map[string]*ImportedIFCSig
	types           *ifcTypes // module-local type declarations (ifc_static_type.go)
	sigs            map[string]*ifcSig
	errs            []*TypeCheckError
	errSeen         map[string]bool  // position+message, so a replay cannot repeat a diagnostic
	effLabel        map[string]Label // memoised effective body label per local func
	computing       map[string]bool  // cycle guard for the effLabel fixpoint
	silent          bool             // suppress sink diagnostics during effLabel and summary walks

	summaries         map[string]*ifcSummary // module fixed point (ifc_summary_local.go)
	owners            []*ifcSummary          // active summaries, innermost last
	rhos              map[string]rhoRef      // label variable name -> owner and param index
	lamIDs            map[ast.Expr]string    // stable summary key per lambda / function literal
	replaying         map[string]bool        // active application states, for cycle-safe replay
	analysisRemaining int                    // fail closed if defensive analysis budget is exhausted
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
			sig.authorizedLabels = []string{"*"}
			for _, p := range eff.Params {
				if p.Key == "label" {
					sig.authorizedLabels = []string{p.Value}
				}
			}
		}
	}
	return sig
}

// checkFunc runs Checks A/C (via the body walk) and Check B over one function.
func (c *ifcChecker) checkFunc(sig *ifcSig) {
	// Keep the existing concrete diagnostic walk: summary variables belong only
	// to silent analysis, never to an ordinary function's environment.
	env := make(ifcEnv, len(sig.params))
	for _, p := range sig.params {
		env[p.name] = ifcVar{label: p.label, typ: p.typ, fn: c.types.fromFuncType(p.typ, p.name)}
	}
	bodyLabel := c.labelOf(sig.decl.Body, env)

	// Check B: the return label and declared authority jointly cover the body.
	// Declassifiers with unlabelled returns lower to bottom too; check them.
	if sig.hasReturnLabel || len(sig.authorizedLabels) > 0 {
		if leaked := unauthorizedLabel(bodyLabel, sig.returnLabel, sig.authorizedLabels); leaked != "" {
			c.addErrs([]*TypeCheckError{newDeclassError(sig.decl, bodyLabel, sig.returnLabel, leaked, sig.authorizedLabels)})
		}
	}
}

// labelOf computes the full IFC label of expr under env: the labels that flowed
// in with the value (flowOf) joined with the labels its static type declares
// (M-IFC-DECLARED-RECORD-LABELS, #1523). As a side effect it performs Check A on
// every call it encounters (unless silent). Each sub-expression is walked once.
func (c *ifcChecker) labelOf(expr ast.Expr, env ifcEnv) Label {
	return c.valueOf(expr, env).label
}

// valueOf is labelOf together with the callable of expr's value.
func (c *ifcChecker) valueOf(expr ast.Expr, env ifcEnv) ifcVal {
	v := c.flowVal(expr, env)
	v.label = LabelJoin(v.label, c.types.deepLabel(c.staticType(expr, env)))
	return v
}

// flowVal computes the label carried in by the VALUE of expr (the flow label —
// the pre-#1523 walk) together with the value's callable. valueOf adds the
// declared part; keeping the two apart is what makes field projection precise:
// s.user does not inherit s.code's declared label. The callable travels
// through every form the label travels through.
func (c *ifcChecker) flowVal(expr ast.Expr, env ifcEnv) ifcVal {
	bottom := ifcVal{label: LabelBottom()}
	switch e := expr.(type) {
	case nil:
		return bottom
	case *ast.Identifier:
		return c.identVal(e.Name, env)
	case *ast.Literal:
		return bottom
	case *ast.BinaryOp:
		return ifcVal{label: LabelJoin(c.labelOf(e.Left, env), c.labelOf(e.Right, env))}
	case *ast.UnaryOp:
		return ifcVal{label: c.labelOf(e.Expr, env)}
	case *ast.FuncCall:
		return c.callVal(e, env)
	case *ast.Let:
		return c.valueOf(e.Body, extendEnv(env, e.Name, c.bindValue(e.Value, e.Type, env)))
	case *ast.LetRec:
		return c.valueOf(e.Body, extendEnv(env, e.Name, c.bindValue(e.Value, e.Type, env)))
	case *ast.Block:
		return c.walkBlock(e, env, c.valueOf)
	case *ast.If:
		c.labelOf(e.Condition, env)
		return joinVals(c.valueOf(e.Then, env), c.valueOf(e.Else, env))
	case *ast.Match:
		// Every pattern variable gets the scrutinee's FULL label (flow and
		// declared) and its callable: destructuring is conservative, not
		// field-precise.
		scrut := c.valueOf(e.Expr, env)
		result := bottom
		for _, cs := range e.Cases {
			armEnv := env
			for _, name := range patternVars(cs.Pattern) {
				armEnv = extendEnv(armEnv, name, ifcVar{label: scrut.label, fn: scrut.fn})
			}
			if cs.Guard != nil {
				c.labelOf(cs.Guard, armEnv)
			}
			result = joinVals(result, c.valueOf(cs.Body, armEnv))
		}
		return result
	case *ast.Record:
		result := bottom
		for _, f := range e.Fields {
			result = joinVals(result, c.valueOf(f.Value, env))
		}
		return result
	case *ast.RecordUpdate:
		result := c.valueOf(e.Base, env)
		for _, f := range e.Fields {
			result = joinVals(result, c.valueOf(f.Value, env))
		}
		return result
	case *ast.RecordAccess:
		// Field-precise when the field's declared type is known: the field gets
		// the record's FLOW label here, and labelOf adds deepLabel(field type)
		// — not the declared labels of the record's other fields. Otherwise a
		// field inherits the record's full join label (conservative). The
		// callable is the whole record's (conservative): any function stored in
		// the record may be the one projected.
		if ft := c.staticType(e, env); ft != nil {
			v := c.flowVal(e.Record, env)
			v.fn = joinCallable(v.fn, c.types.fromFuncType(ft, e.Field))
			return v
		}
		return c.valueOf(e.Record, env)
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
		//
		// The same single walk also yields the closure's callable (#1719): the
		// demands of its typed params and those it infers from its body.
		return c.lambdaVal(e, e.Params, e.Body, env)
	case *ast.FuncLit:
		return c.lambdaVal(e, e.Params, e.Body, env)
	default:
		return bottom
	}
}

// identVal resolves a name in value position: a lexical binding first (it
// shadows a module declaration), then a module function referenced as a value
// (it carries that function's requirements, #1719), then a constructor, and
// otherwise an imported or unknown function (opaque).
func (c *ifcChecker) identVal(name string, env ifcEnv) ifcVal {
	if v, ok := env[name]; ok {
		return ifcVal{label: v.label, fn: v.fn}
	}
	if sig, ok := c.sigs[name]; ok {
		label := LabelBottom()
		if c.exportSummaries {
			// Match an eta closure's existing conservative result label.
			label = LabelJoin(c.calleeResultLabel(sig, nil, nil), c.types.deepLabel(sig.decl.ReturnType))
		}
		return ifcVal{label: label, fn: fromSig(sig)}
	}
	if _, ok := c.types.ctors[name]; ok {
		return ifcVal{label: LabelBottom(), fn: single(&ifcFn{kind: fnCtor})}
	}
	if imported := c.imports[name]; imported != nil {
		return ifcVal{label: importedIFCResult(imported, nil), fn: single(&ifcFn{kind: fnOpaque, name: name, imported: imported})}
	}
	if c.exportSummaries {
		if source, ok := ifcBuiltinSourceLabels[name]; ok {
			// The same trusted string-source facts used by direct calls. Reuse
			// synthetic declared-return metadata; no new producer kind or codec.
			fact := IFCSummaryFromScheme(&Scheme{Type: &TFunc2{Params: []Type{TString}, Return: WithLabel(TString, LabelConst(source))}})
			return ifcVal{label: fact.Return.Decode(), fn: single(&ifcFn{kind: fnOpaque, name: name, imported: fact})}
		}
	}
	// Imported names have no constructor evidence in this surface-only pass.
	// Capitalized functions are legal, so every unknown name takes F2 fallback.
	return ifcVal{label: LabelBottom(), fn: single(&ifcFn{kind: fnOpaque, name: name})}
}

// walkBlock threads block-statement lets through a block { s1; s2; ...; result }
// and returns `last` applied to the result expression. A Let or LetRec parsed
// without `in` (Body == nil) scopes over the remainder of the block, because
// elaboration turns it into nested Core lets.
func (c *ifcChecker) walkBlock(b *ast.Block, env ifcEnv, last func(ast.Expr, ifcEnv) ifcVal) ifcVal {
	result := ifcVal{label: LabelBottom()}
	cur := env
	for _, sub := range b.Exprs {
		switch s := sub.(type) {
		case *ast.Let:
			if s.Body == nil {
				cur = extendEnv(cur, s.Name, c.bindValue(s.Value, s.Type, cur))
				result = ifcVal{label: LabelBottom()}
				continue
			}
		case *ast.LetRec:
			if s.Body == nil {
				cur = extendEnv(cur, s.Name, c.bindValue(s.Value, s.Type, cur))
				result = ifcVal{label: LabelBottom()}
				continue
			}
		}
		result = last(sub, cur)
	}
	return result
}

// bindValue computes what a let binding records: the annotation (or, when
// unannotated, the value's own static type), the hand-off label of the value
// into that type, and its callable — the value's own, joined with whatever a
// written function type promises (`let g: string{not email} -> string = …`).
func (c *ifcChecker) bindValue(value ast.Expr, annot ast.Type, env ifcEnv) ifcVar {
	t := annot
	if t == nil {
		t = c.staticType(value, env)
	}
	v := c.handOffVal(value, t, env)
	return ifcVar{label: v.label, typ: t, fn: joinCallable(v.fn, c.types.fromFuncType(t, describeExpr(value)))}
}

// describeExpr names an expression for a diagnostic's via trail.
func describeExpr(e ast.Expr) string {
	switch x := e.(type) {
	case nil:
		return "value"
	case *ast.Identifier:
		return x.Name
	case *ast.Lambda, *ast.FuncLit:
		return "lambda"
	}
	s := e.String()
	if len(s) > 40 {
		s = s[:37] + "..."
	}
	return s
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
	return c.handOffVal(e, t, env).label
}

// handOffVal is handOff together with the value's callable.
func (c *ifcChecker) handOffVal(e ast.Expr, t ast.Type, env ifcEnv) ifcVal {
	if t == nil {
		return c.valueOf(e, env)
	}
	switch x := e.(type) {
	case *ast.Record:
		if c.types.recordTypeOf(t) != nil {
			result := ifcVal{label: LabelBottom()}
			for _, f := range x.Fields {
				result = joinVals(result, c.handOffVal(f.Value, c.types.fieldType(t, f.Name), env))
			}
			return result
		}
	case *ast.Block:
		return c.walkBlock(x, env, func(sub ast.Expr, cur ifcEnv) ifcVal { return c.handOffVal(sub, t, cur) })
	case *ast.Let:
		if x.Body != nil {
			return c.handOffVal(x.Body, t, extendEnv(env, x.Name, c.bindValue(x.Value, x.Type, env)))
		}
	case *ast.If:
		c.labelOf(x.Condition, env)
		return joinVals(c.handOffVal(x.Then, t, env), c.handOffVal(x.Else, t, env))
	}
	if sameType(c.staticType(e, env), t) {
		return c.flowVal(e, env)
	}
	return c.valueOf(e, env)
}

func (c *ifcChecker) joinElems(elems []ast.Expr, env ifcEnv) ifcVal {
	result := ifcVal{label: LabelBottom()}
	for _, el := range elems {
		result = joinVals(result, c.valueOf(el, env))
	}
	return result
}

// callVal computes a call's result label and callable, and runs Checks A/C
// against the callable of the callee VALUE — whatever the callee expression is
// (#1719). Ordinary checking keeps its existing result policy; M1 export
// analysis additionally follows known producer facts.
func (c *ifcChecker) callVal(call *ast.FuncCall, env ifcEnv) ifcVal {
	name := calleeName(call.Func)
	sig, isLocal := c.sigs[name]
	// A lexical binding shadows a module declaration; use the closure fallback.
	if _, bound := env[name]; bound {
		isLocal = false
	}
	callee := c.valueOf(call.Func, env)

	// Each argument is walked ONCE. argLabels are full labels (used by Check A
	// and by unknown callees); passed are the hand-off labels into a local
	// callee's declared parameter types (the summary re-derives the declared
	// part from the parameter type, so a type-matched argument passes only its
	// flow label — see handOff).
	argLabels := make([]Label, len(call.Args))
	var flowLabels []Label
	if c.exportSummaries {
		flowLabels = make([]Label, len(call.Args))
	}
	passed := make([]Label, len(call.Args))
	args := make([]ifcVal, len(call.Args))
	for i, a := range call.Args {
		v := c.flowVal(a, env)
		st := c.staticType(a, env)
		argLabels[i] = LabelJoin(v.label, c.types.deepLabel(st))
		if flowLabels != nil {
			flowLabels[i] = v.label
		}
		passed[i] = argLabels[i]
		if isLocal && i < len(sig.params) && sameType(st, sig.params[i].typ) {
			passed[i] = v.label
		}
		args[i] = ifcVal{label: argLabels[i], fn: v.fn}
	}

	// Checks A/C on every application. A direct call to a module function
	// keeps the pre-#1719 wording; any other route names the path taken.
	var via []string
	if !isLocal {
		via = []string{describeExpr(call.Func)}
	}
	fn := c.applyCallable(callee.fn, args, call, callee.label, via, 0)

	if isLocal {
		return ifcVal{label: c.calleeResultLabel(sig, passed, argLabels), fn: fn}
	}
	if label, known := c.producerCallableResult(callee.fn, argLabels, flowLabels, callee.label, call, env); known {
		return ifcVal{label: label, fn: fn}
	}
	_, bound := env[name]
	_, constructor := c.types.ctors[name]
	if src, ok := ifcBuiltinSourceLabels[name]; ok && (!c.exportSummaries || (!bound && !constructor)) {
		return ifcVal{label: LabelConst(src), fn: fn}
	}
	// Unknown or imported callee: transparent — propagate the join of the args
	// AND of the callee expression itself.
	//
	// The callee's own label is what makes a local closure safe. `f` in
	// `let f = \u. s in f(0)` has no entry in c.sigs (it is a binding, not a
	// declaration), so this fallback is the path it takes; joining only the
	// argument labels discarded the captured <secret> entirely and made the
	// guarantee bypassable in three tokens. The callee's label was read out of
	// env, where the Let case put it.
	return ifcVal{label: LabelJoin(callee.label, joinLabels(argLabels)), fn: fn}
}

// calleeResultLabel computes the label produced by calling a known local function.
func (c *ifcChecker) calleeResultLabel(sig *ifcSig, argLabels, fullArgLabels []Label) Label {
	if len(sig.authorizedLabels) > 0 {
		// Bare authority retains legacy relabelling. Scoped authority must also
		// account for actual arguments: unlabelled formals seed bottom during
		// Check B, and cannot prove that secret-carrying actuals are covered.
		// Use FULL labels, before hand-off strips type-matched declared labels.
		result := sig.returnLabel
		for _, part := range normalisedParts(joinLabels(fullArgLabels)) {
			if unauthorizedLabel(part, sig.returnLabel, sig.authorizedLabels) != "" {
				result = LabelJoin(result, part)
			}
		}
		return result
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
	// No enclosing summary may record from this walk: its parameters are ⊥,
	// not the caller's ρ, so anything it recorded would be about the wrong body.
	prevOwners := c.owners
	c.owners = nil
	env := make(ifcEnv, len(sig.params))
	for _, p := range sig.params {
		env[p.name] = ifcVar{label: LabelBottom(), typ: p.typ}
	}
	l := c.handOff(sig.decl.Body, sig.decl.ReturnType, env)
	c.owners = prevOwners
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

func newDeclassError(fn *ast.FuncDecl, bodyLabel, declared Label, leaked string, authority []string) *TypeCheckError {
	return &TypeCheckError{
		Kind:     DeclassifyRequiredError,
		Position: fn.Pos.String(),
		Message: fmt.Sprintf(
			"information-flow violation: function %q returns a value labelled %s but declares return label %s, hiding <%s>; declared authority %s",
			fn.Name, bodyLabel, declared, leaked, formatAuthority(authority)),
		Suggestion: "widen the declared return label, authorize the hidden label with Declassify[label=...], or explicitly use bare Declassify for blanket authority",
	}
}

// unauthorizedLabel checks return coverage plus the stable authority summary.
func unauthorizedLabel(have, declared Label, labels []string) string {
	result := LabelBottom()
	for _, label := range labels {
		if label == "*" {
			return ""
		}
		result = LabelJoin(result, LabelConst(label))
	}
	return labelNotCoveredBy(have, LabelJoin(declared, result))
}

func formatAuthority(labels []string) string {
	if len(labels) == 0 {
		return "none"
	}
	if labels[0] == "*" {
		return "Declassify"
	}
	return fmt.Sprintf("Declassify[label=%s]", labels[0])
}

func newParamLabelCoverError(call *ast.FuncCall, param ifcParam, label Label) *TypeCheckError {
	return &TypeCheckError{
		Kind:       ParamLabelCoverError,
		Position:   call.Pos.String(),
		Message:    fmt.Sprintf("information-flow violation: value labelled %s reaches parameter %q declared %s, which does not cover it", label, param.name, param.typ),
		Suggestion: "relabel explicitly through a scoped declassifier, or widen the parameter's declared label",
	}
}
