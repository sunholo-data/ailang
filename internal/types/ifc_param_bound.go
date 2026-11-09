package types

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
)

// ifc_param_bound.go — what a parameter's declared type demands of the
// argument (Checks A and C), structurally over the type (#2185 follow-up to
// M-IFC-CROSS-MODULE).
//
// Until this pass the demand was read off the parameter type's TOP-LEVEL
// LabelledType only, so `takeOne(x: string<sqlsafe>)` rejected a <arg> value
// while `takeList(xs: [string<sqlsafe>])` accepted `[s]` for the same s. A label
// or refinement written inside a list, array, tuple, record, type-constructor
// argument or module-local declared type is a demand too.
//
// Values carry ONE flat label — the join of every label reachable in them (a
// list literal's label is the join of its elements) — so a constituent of the
// argument's label cannot be attributed to a position, and the demand is flat:
//
//   - forbid(T): every {not ℓ} written at a covariant data position of T. The
//     argument's label must contain none of them.
//   - cover(T): each root-to-leaf path of T through covariant data positions is
//     bounded by the labels written along it (`[string<a>]<b>`: the element path
//     carries both b and a). A path whose labels all name ℓ admits ℓ; a path
//     naming two different labels admits only ⊥ (each label is a bound, and
//     distinct constants have no common non-⊥ constituent). When T writes any
//     positive label at a data position, every constituent of the argument's
//     label must be admitted by SOME labelled path: cover(T) is the join of the
//     admitted labels.
//
// For a top-level `x: T<ℓ>` with no label inside T this is exactly the original
// rule (cover ℓ). Two consequences of flat labels, both conservative (they
// reject, never admit):
//
//   - an unlabelled leaf beside a labelled one (`{q: string<sqlsafe>, note: string}`,
//     `Raw(string) | Safe(string<sqlsafe>)`) is bounded by the labelled paths;
//   - a label admitted at one position is not distinguished from another
//     position of the same parameter (`{a: string<x>, b: string<y>}` accepts
//     y-data in a). Position-precise checking needs structured value labels.
//
// Variance. Data positions (list/array elements, tuple elements, record fields,
// constructor fields, type-constructor arguments) are covariant: the argument's
// contents are bounded. A function type is a leaf: labels written INSIDE it add
// nothing here. Its parameter positions are contravariant — demands the callee
// owes when it applies the function value, enforced through
// fromFuncType/callableType at that application — and its return position is
// left to a follow-up, because a named function referenced as a value does not
// carry its result label the way a lambda does, so bounding it would check
// lambdas but not references.
//
// Type-constructor arguments of a constructor this module does not declare
// (Option, Result, Map, an imported type) are taken as covariant data, the same
// assumption deepLabel makes. A module-local generic declaration is walked with
// its parameters bound to the written argument types, so a parameter used only
// under a function type contributes nothing.

// ifcBound is the flat demand a parameter type places on its argument.
type ifcBound struct {
	cover  Label // join of the labels some labelled path admits; meaningful only when has
	has    bool  // some covariant data position writes a positive label
	forbid map[string]bool
}

// ifcPath is the conjunction of positive labels along one path: none, one
// label name, or contradictory (two different names: only ⊥ satisfies both).
type ifcPath struct {
	name   string
	contra bool
}

func (p ifcPath) with(name string) ifcPath {
	switch {
	case p.contra || p.name == name:
		return p
	case p.name == "":
		return ifcPath{name: name}
	default:
		return ifcPath{contra: true}
	}
}

// ifcTypeArg is a generic declaration's type argument, with the bindings in
// force where it was written.
type ifcTypeArg struct {
	t      ast.Type
	params map[string]ifcTypeArg
}

func (b *ifcBound) leaf(p ifcPath) {
	if p.name == "" || p.contra {
		return // unlabelled (bounded by the other paths) or ⊥-only
	}
	if b.cover == nil {
		b.cover = LabelBottom()
	}
	b.cover = LabelJoin(b.cover, LabelConst(p.name))
}

func (b *ifcBound) addForbid(f string) {
	if b.forbid == nil {
		b.forbid = map[string]bool{}
	}
	b.forbid[f] = true
}

// forbidden returns the refinement labels as a sorted (canonical) slice.
func (b ifcBound) forbidden() []string {
	out := make([]string, 0, len(b.forbid))
	for f := range b.forbid {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// paramBound is the demand declared parameter type t places on its argument.
func (ts *ifcTypes) paramBound(t ast.Type) ifcBound {
	var b ifcBound
	ts.addBound(&b, t, ifcPath{}, nil, map[string]bool{}, 0)
	if b.has && b.cover == nil {
		b.cover = LabelBottom() // every labelled path contradictory: only ⊥ passes
	}
	return b
}

// demandOf projects a declared parameter type onto a demand (structurally).
func (ts *ifcTypes) demandOf(name string, t ast.Type) ifcDemand {
	d := ifcDemand{param: name, typ: t}
	b := ts.paramBound(t)
	if b.has {
		d.covers = []Label{b.cover}
	}
	d.forbid = b.forbidden()
	return canonicalDemand(d)
}

// addBound accumulates t's demand into b along path. params binds a
// module-local generic declaration's type parameters.
func (ts *ifcTypes) addBound(b *ifcBound, t ast.Type, path ifcPath, params map[string]ifcTypeArg, visiting map[string]bool, depth int) {
	if depth > ifcAnalysisDepth {
		panic(ifcAnalysisLimit{reason: "parameter type nesting"})
	}
	switch tt := t.(type) {
	case nil, *ast.FuncType:
		b.leaf(path) // a function type is a leaf: see the file comment
	case *ast.LabelledType:
		if tt.Label != nil {
			b.has = true
			path = path.with(tt.Label.Name)
		}
		if tt.Refinement != nil {
			b.addForbid(tt.Refinement.NotLabel)
		}
		ts.addBound(b, tt.Base, path, params, visiting, depth+1)
	case *ast.ListType:
		ts.addBound(b, tt.Element, path, params, visiting, depth+1)
	case *ast.ArrayType:
		ts.addBound(b, tt.Element, path, params, visiting, depth+1)
	case *ast.TupleType:
		ts.addAll(b, tt.Elements, path, params, visiting, depth)
	case *ast.RecordType:
		if len(tt.Fields) == 0 {
			b.leaf(path)
		}
		for _, f := range tt.Fields {
			if f != nil {
				ts.addBound(b, f.Type, path, params, visiting, depth+1)
			}
		}
	case *ast.TypeVar:
		ts.addName(b, tt.Name, nil, path, params, visiting, depth)
	case *ast.SimpleType:
		ts.addName(b, tt.Name, nil, path, params, visiting, depth)
	case *ast.TypeApp:
		args := make([]ifcTypeArg, len(tt.Args))
		for i, a := range tt.Args {
			args[i] = ifcTypeArg{t: a, params: params}
		}
		ts.addName(b, tt.Constructor, args, path, params, visiting, depth)
	default:
		panic(fmt.Sprintf("ifc paramBound: unhandled ast.Type %T", t))
	}
}

func (ts *ifcTypes) addAll(b *ifcBound, types []ast.Type, path ifcPath, params map[string]ifcTypeArg, visiting map[string]bool, depth int) {
	if len(types) == 0 {
		b.leaf(path)
	}
	for _, el := range types {
		ts.addBound(b, el, path, params, visiting, depth+1)
	}
}

// addName resolves a type name applied to args: a bound type parameter, a
// module-local declaration (walked with its parameters bound to args), or a
// builtin/imported name — a leaf whose written arguments are covariant data.
func (ts *ifcTypes) addName(b *ifcBound, name string, args []ifcTypeArg, path ifcPath, params map[string]ifcTypeArg, visiting map[string]bool, depth int) {
	if arg, ok := params[name]; ok && len(args) == 0 {
		ts.addBound(b, arg.t, path, arg.params, visiting, depth+1)
		return
	}
	td, local := ts.decls[name]
	if !local {
		if len(args) == 0 {
			b.leaf(path)
		}
		for _, a := range args {
			ts.addBound(b, a.t, path, a.params, visiting, depth+1)
		}
		return
	}
	// A recursive occurrence adds no path the first visit has not: cutting it
	// can only drop admitted labels, the conservative direction (has is set by
	// any label the walk meets).
	if visiting[name] {
		return
	}
	visiting[name] = true
	defer delete(visiting, name)
	bound := make(map[string]ifcTypeArg, len(td.TypeParams))
	for i, p := range td.TypeParams {
		if i < len(args) {
			bound[p] = args[i]
		} else {
			bound[p] = ifcTypeArg{} // unapplied: an unknown leaf
		}
	}
	ts.addAll(b, declBodyTypes(td), path, bound, visiting, depth)
}

// topLabel is the positive label written at the top of t, if any. It is the
// label a parameter is SEEDED with (a source), as distinct from paramBound,
// which is what the argument must satisfy: a nested cover bounds the argument
// as a whole but says nothing about which component carries what, so it may not
// replace the parameter's symbolic label (field projection would under-label).
func topLabel(t ast.Type) (Label, bool) {
	if lt, ok := t.(*ast.LabelledType); ok && lt.Label != nil {
		return LabelConst(lt.Label.Name), true
	}
	return LabelBottom(), false
}
