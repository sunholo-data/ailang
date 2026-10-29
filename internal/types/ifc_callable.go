package types

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// ifc_callable.go — the CALLABLE half of the IFC abstract value
// (M-IFC-FUNCTION-VALUE-SINKS, #1719).
//
// The IFC pass used to model every value as a Label and nothing else, so a
// sink's argument requirements ({not ℓ}, positive <ℓ>) lived only in a
// name-keyed table read at a call written `name(args)`. Any other route to the
// same function value — an alias, a higher-order parameter, a record field, a
// list element, a returned function, a wrapper lambda — dropped them.
//
// An ifcVal now carries, beside its label, an *ifcCallable: what this value
// requires of its arguments if it is applied. It travels through every binding
// form the label travels through, and Checks A/C run at EVERY application
// against the callee value's callable, whatever the callee expression is.
// Requirements stay in the IFC domain: nothing here touches HM types, CoreTI,
// codegen or the iface (design F1).

// ifcVal is the IFC abstract value of an expression.
type ifcVal struct {
	label Label
	fn    *ifcCallable // nil: no requirement known for applying this value
}

// ifcCallable is a set of alternatives: the value may be any one of them. The
// join of two callables is the union of their alternatives, so a join is
// checked against every demand of either side — at least as strict as each.
type ifcCallable struct {
	alts []*ifcFn
}

type fnKind int

const (
	// fnTop is a module function referenced by name. Its declared demands are
	// in params; its summary (deferred applications, result callable) is read
	// from the module fixed point when it is applied.
	fnTop fnKind = iota
	// fnLambda is a lambda or function literal: demands from its typed params,
	// plus the summary recorded while its body was walked.
	fnLambda
	// fnDeclared carries demands written in a function-type annotation
	// (`f: string{not email} -> string`), or the demand-only leaf of a
	// deferred application.
	fnDeclared
	// fnProbe is "parameter idx of the body being summarised (owner)": a
	// symbolic function argument. Applying it records a deferred application
	// that is replayed when the owner is applied to a concrete function.
	fnProbe
	// fnOpaque is an imported or otherwise unknown function. Passing a
	// callable to it takes the conservative arg-join fallback (design F2).
	fnOpaque
	// fnCtor is a data constructor: it stores its arguments, it never applies
	// them, so no fallback check is needed.
	fnCtor
)

type ifcFn struct {
	kind     fnKind
	name     string      // display name for diagnostics
	params   []ifcDemand // positional demands (fnTop, fnLambda, fnDeclared)
	owner    string      // summary key: fnLambda's own, fnProbe's owner
	idx      int         // fnProbe: parameter index
	res      int         // fnProbe: applications already taken
	deferred []ifcDeferred
	result   *ifcCallable // fnLambda, fnDeclared: callable of the result
	key      string       // memoised identity (see fnKey)
}

// ifcDemand is what one parameter position requires of its argument.
type ifcDemand struct {
	param  string
	typ    ast.Type // declared parameter type, for messages (may be nil)
	forbid []string // {not ℓ}
	covers []Label  // positive <ℓ>: the argument must be covered by each
}

// canonicalDemand keeps forbidden labels as a sorted set. Positive covers are
// a conjunction: nil means no cover, while []Label{bottom} permits only bottom.
// This is the meet representation, preserving absent versus explicit bottom.
func canonicalDemand(d ifcDemand) ifcDemand {
	set := map[string]bool{}
	for _, name := range d.forbid {
		set[name] = true
	}
	d.forbid = nil
	for name := range set {
		d.forbid = append(d.forbid, name)
	}
	sort.Strings(d.forbid)
	return d
}

func (d ifcDemand) empty() bool { return len(d.forbid) == 0 && len(d.covers) == 0 }

// violation reports the first constituent of arg that d rejects, or "".
func (d ifcDemand) violation(arg Label) string {
	for _, cv := range d.covers {
		if l := labelNotCoveredBy(arg, cv); l != "" {
			return l
		}
	}
	for _, f := range d.forbid {
		if CheckSinkLabel(arg, f) != nil {
			return f
		}
	}
	return ""
}

func (d ifcDemand) String() string {
	return fmt.Sprintf("%s{%s|%v}", d.param, strings.Join(d.forbid, ","), d.covers)
}

// ifcDeferred is an application recorded while a body was summarised whose
// argument labels mention that body's label variables (ρ). It is replayed,
// with ρ and probes substituted, when the summarised function is applied.
type ifcDeferred struct {
	target *ifcCallable
	args   []ifcVal
	via    []string
}

func single(f *ifcFn) *ifcCallable { return &ifcCallable{alts: []*ifcFn{f}} }

// joinCallable is the join of two callables: the union of their alternatives
// (deduplicated by identity, order preserved). nil is the identity.
func joinCallable(a, b *ifcCallable) *ifcCallable {
	if a == nil || len(a.alts) == 0 {
		if b == nil || len(b.alts) == 0 {
			return nil
		}
		return b
	}
	if b == nil || len(b.alts) == 0 {
		return a
	}
	seen := make(map[string]bool, len(a.alts)+len(b.alts))
	out := &ifcCallable{}
	for _, f := range append(append([]*ifcFn{}, a.alts...), b.alts...) {
		k := fnKey(f)
		if !seen[k] {
			seen[k] = true
			out.alts = append(out.alts, f)
		}
	}
	if len(out.alts) == len(a.alts) {
		return a
	}
	return out
}

func joinVals(vals ...ifcVal) ifcVal {
	out := ifcVal{label: LabelBottom()}
	for _, v := range vals {
		out.label = LabelJoin(out.label, v.label)
		out.fn = joinCallable(out.fn, v.fn)
	}
	return out
}

// fnKey is a canonical identity for an alternative, content-addressed for
// lambdas because substitution produces new instances of the same literal.
func fnKey(f *ifcFn) string {
	if f.key != "" {
		return f.key
	}
	var k string
	switch f.kind {
	case fnTop:
		k = "T:" + f.name
	case fnProbe:
		k = fmt.Sprintf("P:%s#%d^%d", f.owner, f.idx, f.res)
	case fnOpaque:
		k = "O"
	case fnCtor:
		k = "C"
	case fnDeclared:
		k = "D:" + f.name + demandsKey(f.params) + "->" + callableKey(f.result)
	case fnLambda:
		parts := make([]string, len(f.deferred))
		for i, d := range f.deferred {
			parts[i] = deferredKey(d)
		}
		sort.Strings(parts)
		k = "L:" + f.owner + demandsKey(f.params) + "{" + strings.Join(parts, ";") + "}->" + callableKey(f.result)
	}
	f.key = k
	return k
}

func demandsKey(ds []ifcDemand) string {
	parts := make([]string, len(ds))
	for i, d := range ds {
		parts[i] = d.String()
	}
	return "(" + strings.Join(parts, ",") + ")"
}

func callableKey(c *ifcCallable) string {
	if c == nil {
		return "_"
	}
	parts := make([]string, len(c.alts))
	for i, f := range c.alts {
		parts[i] = fnKey(f)
	}
	sort.Strings(parts)
	return "[" + strings.Join(parts, "|") + "]"
}

func deferredKey(d ifcDeferred) string {
	parts := make([]string, len(d.args))
	for i, a := range d.args {
		parts[i] = a.label.String() + "&" + callableKey(a.fn)
	}
	return callableKey(d.target) + "(" + strings.Join(parts, ",") + ")"
}

// demandOf projects a declared parameter type onto a demand.
func demandOf(name string, t ast.Type) ifcDemand {
	d := ifcDemand{param: name, typ: t}
	if lt, ok := t.(*ast.LabelledType); ok {
		if lt.Label != nil {
			d.covers = []Label{LabelConst(lt.Label.Name)}
		}
		if lt.Refinement != nil {
			d.forbid = []string{lt.Refinement.NotLabel}
		}
	}
	return canonicalDemand(d)
}

// fromSig is the callable of a module function referenced as a value.
func fromSig(sig *ifcSig) *ifcCallable {
	f := &ifcFn{kind: fnTop, name: sig.decl.Name}
	for _, p := range sig.params {
		d := ifcDemand{param: p.name, typ: p.typ}
		if !LabelEqual(p.label, LabelBottom()) {
			d.covers = []Label{p.label}
		}
		if p.hasNot {
			d.forbid = []string{p.notLabel}
		}
		f.params = append(f.params, canonicalDemand(d))
	}
	return single(f)
}

// fromFuncType is the callable a written function type promises: the demands
// of its labelled or refined parameters, and of its result if that is a
// function type. nil when the type declares none (the common case, so plain
// HOF code carries no callable at all).
func (ts *ifcTypes) fromFuncType(t ast.Type, name string) *ifcCallable {
	return ts.callableType(t, name, 0)
}

func (ts *ifcTypes) callableType(t ast.Type, name string, depth int) *ifcCallable {
	if depth > ifcAnalysisDepth {
		panic(ifcAnalysisLimit{reason: "written callable type nesting"})
	}
	ft := ts.funcTypeOf(t)
	if ft == nil {
		return nil
	}
	f := &ifcFn{kind: fnDeclared, name: name, result: ts.callableType(ft.Return, name+"(..)", depth+1)}
	has := f.result != nil
	for i, p := range ft.Params {
		d := demandOf(fmt.Sprintf("#%d", i+1), p)
		has = has || !d.empty()
		f.params = append(f.params, canonicalDemand(d))
	}
	if !has {
		return nil
	}
	return single(f)
}

// funcTypeOf resolves t to a function type through label wrappers and
// non-generic module-local aliases, or returns nil.
func (ts *ifcTypes) funcTypeOf(t ast.Type) *ast.FuncType {
	seen := map[ast.Type]bool{}
	for t != nil && !seen[t] {
		seen[t] = true
		switch tt := t.(type) {
		case *ast.FuncType:
			return tt
		case *ast.LabelledType:
			t = tt.Base
		case *ast.SimpleType:
			td, ok := ts.decls[tt.Name]
			if !ok || len(td.TypeParams) > 0 {
				return nil
			}
			alias, ok := td.Definition.(*ast.TypeAlias)
			if !ok {
				return nil
			}
			t = alias.Target
		case *ast.TypeVar, *ast.ListType, *ast.ArrayType, *ast.TupleType, *ast.TypeApp, *ast.RecordType:
			return nil
		default:
			panic(fmt.Sprintf("ifc funcTypeOf: unhandled ast.Type %T", t))
		}
	}
	return nil
}

// --- diagnostics -----------------------------------------------------------

// demandErrors checks one argument against one demand. With an empty via the
// wording is the pre-#1719 direct-call message, byte for byte.
func demandErrors(call *ast.FuncCall, callee string, d ifcDemand, arg Label, via []string) []*TypeCheckError {
	var out []*TypeCheckError
	for _, cv := range d.covers {
		if labelNotCoveredBy(arg, cv) == "" {
			continue
		}
		if len(via) == 0 {
			out = append(out, newParamLabelCoverError(call, ifcParam{name: d.param, typ: d.typ}, arg))
			continue
		}
		declared := cv.String()
		if d.typ != nil {
			declared = d.typ.String()
		}
		out = append(out, &TypeCheckError{
			Kind:     ParamLabelCoverError,
			Position: call.Pos.String(),
			Message: fmt.Sprintf("information-flow violation: value labelled %s reaches parameter %q of %s (via %s) declared %s, which does not cover it",
				arg, d.param, callee, strings.Join(via, " → "), declared),
			Suggestion: "relabel explicitly through a scoped declassifier before the value reaches this function, or widen the parameter's declared label",
		})
	}
	for _, f := range d.forbid {
		if CheckSinkLabel(arg, f) == nil {
			continue
		}
		if len(via) == 0 {
			out = append(out, newSinkError(call, d.param, f, arg))
			continue
		}
		out = append(out, &TypeCheckError{
			Kind:     SinkRefinementError,
			Position: call.Pos.String(),
			Message: fmt.Sprintf("information-flow violation: value labelled %s reaches parameter %q of %s (via %s) which forbids it (string{not %s})",
				arg, d.param, callee, strings.Join(via, " → "), f),
			Suggestion: fmt.Sprintf("declassify the value through a function declaring ! {Declassify} before it reaches %s; the {not %s} requirement holds through aliases, higher-order functions and wrappers",
				callee, f),
		})
	}
	return out
}
