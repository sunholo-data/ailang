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
// An ifcVal now carries, beside its label, an *ifcCallable: the set of
// function ATOMS the value may be. An atom names a function-like entity
// (ifcFun) by identity — a module function, a lambda or function literal (by
// AST node), a written function type, a parameter of the body being walked
// (a probe), or an imported/unknown function. What an entity requires of its
// arguments lives on the entity (ifc_summary_local.go), so the domain is
// finite and the module fixed point is polynomial. Checks A/C run at EVERY
// application against the atoms of the callee VALUE. Requirements stay in the
// IFC domain: nothing here touches HM types, CoreTI, codegen or the iface
// (design F1).

// ifcVal is the IFC abstract value of an expression.
type ifcVal struct {
	label Label
	fn    *ifcCallable // nil: not (known to be) a function with requirements
}

type atomKind int

const (
	atomFun    atomKind = iota // a module function, lambda or written function type (ifcFun)
	atomProbe                  // parameter idx of the entity `owner` (symbolic)
	atomOpaque                 // imported or unknown function: arg-join fallback (design F2)
	atomCtor                   // data constructor: stores its arguments, applies none
)

type ifcAtom struct {
	kind  atomKind
	fun   string // atomFun: entity key
	owner string // atomProbe
	idx   int    // atomProbe
}

func (a ifcAtom) key() string {
	switch a.kind {
	case atomFun:
		return "F:" + a.fun
	case atomProbe:
		return probeKey(a.owner, a.idx)
	case atomOpaque:
		return "O"
	default:
		return "C"
	}
}

func probeKey(owner string, idx int) string { return fmt.Sprintf("P:%s#%d", owner, idx) }

// ifcCallable is an immutable set of atoms. The join is set union, so a
// joined value is checked against every demand of either side: at least as
// strict as each.
type ifcCallable struct {
	atoms map[string]ifcAtom
}

func single(a ifcAtom) *ifcCallable {
	return &ifcCallable{atoms: map[string]ifcAtom{a.key(): a}}
}

func funAtom(key string) *ifcCallable { return single(ifcAtom{kind: atomFun, fun: key}) }

func probeAtom(owner string, idx int) *ifcCallable {
	return single(ifcAtom{kind: atomProbe, owner: owner, idx: idx})
}

var (
	opaqueCallable = single(ifcAtom{kind: atomOpaque})
	ctorCallable   = single(ifcAtom{kind: atomCtor})
)

// joinCallable is the union of two atom sets; nil is the identity.
func joinCallable(a, b *ifcCallable) *ifcCallable {
	if a == nil || len(a.atoms) == 0 {
		if b == nil || len(b.atoms) == 0 {
			return nil
		}
		return b
	}
	if b == nil || len(b.atoms) == 0 {
		return a
	}
	grow := false
	for k := range b.atoms {
		if _, ok := a.atoms[k]; !ok {
			grow = true
			break
		}
	}
	if !grow {
		return a
	}
	out := &ifcCallable{atoms: make(map[string]ifcAtom, len(a.atoms)+len(b.atoms))}
	for k, v := range a.atoms {
		out.atoms[k] = v
	}
	for k, v := range b.atoms {
		out.atoms[k] = v
	}
	return out
}

// sorted lists the atoms in key order (deterministic diagnostics).
func (fn *ifcCallable) sorted() []ifcAtom {
	if fn == nil {
		return nil
	}
	keys := make([]string, 0, len(fn.atoms))
	for k := range fn.atoms {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]ifcAtom, len(keys))
	for i, k := range keys {
		out[i] = fn.atoms[k]
	}
	return out
}

// relevant drops the atoms that never lead to a demand when a value is stored
// and passed on (opaque functions, constructors), so data built by imported
// functions does not drag atoms along. nil when nothing relevant remains.
func relevant(fn *ifcCallable) *ifcCallable {
	if fn == nil {
		return nil
	}
	var out *ifcCallable
	for k, a := range fn.atoms {
		if a.kind == atomFun || a.kind == atomProbe {
			if out == nil {
				out = &ifcCallable{atoms: map[string]ifcAtom{}}
			}
			out.atoms[k] = a
		}
	}
	if out != nil && len(out.atoms) == len(fn.atoms) {
		return fn
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

// --- demands -----------------------------------------------------------------

// ifcDemand is one requirement on one parameter position. A concrete demand
// forbids a label ({not ℓ}) or requires coverage by one (<ℓ>); a symbolic
// demand says "this argument is applied, at position pos, by whatever function
// is bound to probe". A symbolic demand is resolved against the actual
// function argument at each call (context-sensitive), or through the module's
// flow sets when the probe is out of scope (a returned closure).
type ifcDemand struct {
	forbid string // {not ℓ}
	cover  Label  // positive <ℓ>; nil when not a cover demand

	probe string // symbolic: probe key
	owner string // symbolic: the probe's owner entity
	idx   int    // symbolic: the probe's parameter index
	pos   int    // symbolic: position the probe is applied at

	leaf  string   // function that declares the concrete requirement
	param string   // its parameter name
	typ   ast.Type // its declared type, for messages (may be nil)
	via   []string // hops from the entity carrying the demand to the leaf
}

func (d *ifcDemand) symbolic() bool { return d.probe != "" }

// key identifies a demand for set membership; the via trail is provenance,
// not identity (the first path recorded is the one reported).
func (d *ifcDemand) key() string {
	if d.symbolic() {
		return fmt.Sprintf("S|%s|%d", d.probe, d.pos)
	}
	cover := ""
	if d.cover != nil {
		cover = d.cover.String()
	}
	return "D|" + d.forbid + "|" + cover + "|" + d.leaf + "|" + d.param
}

// violation reports what arg violates in a concrete demand, or "".
func (d *ifcDemand) violation(arg Label) string {
	if d.cover != nil {
		if l := labelNotCoveredBy(arg, d.cover); l != "" {
			return l
		}
	}
	if d.forbid != "" && CheckSinkLabel(arg, d.forbid) != nil {
		return d.forbid
	}
	return ""
}

// demandsOfType projects a declared parameter type onto its demands.
func demandsOfType(leaf, param string, t ast.Type) []*ifcDemand {
	lt, ok := t.(*ast.LabelledType)
	if !ok {
		return nil
	}
	var out []*ifcDemand
	if lt.Label != nil {
		out = append(out, &ifcDemand{cover: LabelConst(lt.Label.Name), leaf: leaf, param: param, typ: t})
	}
	if lt.Refinement != nil {
		out = append(out, &ifcDemand{forbid: lt.Refinement.NotLabel, leaf: leaf, param: param, typ: t})
	}
	return out
}

// funcTypeOf resolves t to a function type through label wrappers and
// non-generic module-local aliases, or returns nil.
func (ts *ifcTypes) funcTypeOf(t ast.Type) *ast.FuncType {
	for depth := 0; depth < 16 && t != nil; depth++ {
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

// demandError renders a violated concrete demand. With an empty via the
// wording is the pre-#1719 direct-call message, byte for byte.
func demandError(call *ast.FuncCall, d *ifcDemand, arg Label, via []string) *TypeCheckError {
	if d.cover != nil {
		if len(via) == 0 {
			return newParamLabelCoverError(call, ifcParam{name: d.param, typ: d.typ}, arg)
		}
		declared := d.cover.String()
		if d.typ != nil {
			declared = d.typ.String()
		}
		return &TypeCheckError{
			Kind:     ParamLabelCoverError,
			Position: call.Pos.String(),
			Message: fmt.Sprintf("information-flow violation: value labelled %s reaches parameter %q of %s (via %s) declared %s, which does not cover it",
				arg, d.param, d.leaf, strings.Join(via, " → "), declared),
			Suggestion: "relabel explicitly through a scoped declassifier before the value reaches this function, or widen the parameter's declared label",
		}
	}
	if len(via) == 0 {
		return newSinkError(call, d.param, d.forbid, arg)
	}
	return &TypeCheckError{
		Kind:     SinkRefinementError,
		Position: call.Pos.String(),
		Message: fmt.Sprintf("information-flow violation: value labelled %s reaches parameter %q of %s (via %s) which forbids it (string{not %s})",
			arg, d.param, d.leaf, strings.Join(via, " → "), d.forbid),
		Suggestion: fmt.Sprintf("declassify the value through a function declaring ! {Declassify} before it reaches %s; the {not %s} requirement holds through aliases, higher-order functions and wrappers",
			d.leaf, d.forbid),
	}
}
