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
	label    Label  // source label from `x: T<label>` (⊥ if none)
	notLabel string // sink refinement label from `x: T{not ℓ}` ("" if none)
	hasNot   bool
}

type ifcChecker struct {
	sigs      map[string]*ifcSig
	errs      []*TypeCheckError
	effLabel  map[string]Label // memoised effective body label per local func
	computing map[string]bool  // cycle guard for the effLabel fixpoint
	silent    bool             // suppress sink diagnostics during effLabel walks
}

func buildIFCSig(fn *ast.FuncDecl) *ifcSig {
	sig := &ifcSig{decl: fn, returnLabel: LabelBottom()}
	for _, p := range fn.Params {
		ip := ifcParam{name: p.Name, label: LabelBottom()}
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
	env := make(map[string]Label, len(sig.params))
	for _, p := range sig.params {
		env[p.name] = p.label
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

// labelOf computes the IFC label of expr under env (var name -> label). As a
// side effect it performs Check A on every call it encounters (unless silent).
func (c *ifcChecker) labelOf(expr ast.Expr, env map[string]Label) Label {
	switch e := expr.(type) {
	case nil:
		return LabelBottom()
	case *ast.Identifier:
		if l, ok := env[e.Name]; ok {
			return l
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
		return c.labelOf(e.Body, extendEnv(env, e.Name, c.labelOf(e.Value, env)))
	case *ast.LetRec:
		return c.labelOf(e.Body, extendEnv(env, e.Name, c.labelOf(e.Value, env)))
	case *ast.Block:
		// A block { s1; s2; ...; result } scopes a block-statement let (a Let or
		// LetRec parsed without `in`, so its Body is nil) over the remainder of
		// the block. Elaboration turns this into nested Core lets, so the surface
		// walk must thread the binding into the environment for later statements.
		last := LabelBottom()
		cur := env
		for _, sub := range e.Exprs {
			switch s := sub.(type) {
			case *ast.Let:
				if s.Body == nil {
					cur = extendEnv(cur, s.Name, c.labelOf(s.Value, cur))
					last = LabelBottom()
					continue
				}
			case *ast.LetRec:
				if s.Body == nil {
					cur = extendEnv(cur, s.Name, c.labelOf(s.Value, cur))
					last = LabelBottom()
					continue
				}
			}
			last = c.labelOf(sub, cur)
		}
		return last
	case *ast.If:
		c.labelOf(e.Condition, env)
		return LabelJoin(c.labelOf(e.Then, env), c.labelOf(e.Else, env))
	case *ast.Match:
		scrut := c.labelOf(e.Expr, env)
		result := LabelBottom()
		for _, cs := range e.Cases {
			armEnv := env
			for _, name := range patternVars(cs.Pattern) {
				armEnv = extendEnv(armEnv, name, scrut)
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
		// Conservative: a field inherits the record's join label.
		return c.labelOf(e.Record, env)
	case *ast.List:
		return c.joinElems(e.Elements, env)
	case *ast.Array:
		return c.joinElems(e.Elements, env)
	case *ast.Tuple:
		return c.joinElems(e.Elements, env)
	case *ast.Lambda:
		// A closure value carries no label; still walk the body so sink
		// violations fed by captured labelled variables are reported.
		c.labelOf(e.Body, env)
		return LabelBottom()
	case *ast.FuncLit:
		c.labelOf(e.Body, env)
		return LabelBottom()
	default:
		return LabelBottom()
	}
}

func (c *ifcChecker) joinElems(elems []ast.Expr, env map[string]Label) Label {
	result := LabelBottom()
	for _, el := range elems {
		result = LabelJoin(result, c.labelOf(el, env))
	}
	return result
}

// labelOfCall computes a call's result label and runs Check A on the callee's
// sink parameters.
func (c *ifcChecker) labelOfCall(call *ast.FuncCall, env map[string]Label) Label {
	argLabels := make([]Label, len(call.Args))
	for i, a := range call.Args {
		argLabels[i] = c.labelOf(a, env)
	}
	name := calleeName(call.Func)

	if sig, ok := c.sigs[name]; ok {
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
		return c.calleeResultLabel(sig, argLabels)
	}
	if src, ok := ifcBuiltinSourceLabels[name]; ok {
		return LabelConst(src)
	}
	// Unknown or imported callee: transparent — propagate the join of args.
	return joinLabels(argLabels)
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
// local function's body produces with parameters seeded at ⊥ — capturing taint
// introduced inside the body (e.g. a secret() call) independent of arguments.
// A cycle guard makes mutual recursion converge conservatively.
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
	env := make(map[string]Label, len(sig.params))
	for _, p := range sig.params {
		env[p.name] = LabelBottom()
	}
	l := c.labelOf(sig.decl.Body, env)
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

func extendEnv(env map[string]Label, name string, l Label) map[string]Label {
	next := make(map[string]Label, len(env)+1)
	for k, v := range env {
		next[k] = v
	}
	next[name] = l
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
