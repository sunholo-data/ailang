package types

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// Numeric variants (#2378, #2333): a top-level binding generic over `Num a`
// is compiled once per numeric instance of a, not once at the default.
//
// `func inc(x) = x + 1` has the type `Num a => a -> a`, and an importer sees
// that type. The runtime has no dictionary passing, so the body cannot learn
// a's instance at a call: compiled once, at the default `int` (top-level
// defaulting pins the body's recorded types), its literal `1` is an Int and
// `inc(2.5)` failed at run time ("numeric dictionary method add received an
// int and a float"). Num has exactly two instances, int and float, so the
// pipeline compiles the body again for every other assignment of its Num
// variables (internal/pipeline/numeric_variants.go): `inc$numF`, typed at
// float throughout, its literal a Float. A use of the binding picks the
// variant its instantiation names; the int one is the binding itself, so
// int code runs exactly the code it ran before.
//
// The type checker supplies three things here:
//   - which module declarations have variants, over which variables, in a
//     fixed order (NumVariantVars) an importer recomputes from the exported
//     scheme alone, so the interface does not change;
//   - every reference to such a binding, in-module or imported (NumericRefs),
//     whose instantiation picks the variant (NumericRefInstance);
//   - the type each node of a defaulted generic body had before defaulting
//     (VariantType, VariantConstraint), from which a variant's node types and
//     dictionaries are read.
//
// A let-bound lambda inside a declaration is not given variants: it is one
// run-time value. A use of a binding that has variants therefore keeps the
// enclosing let monomorphic, exactly as an integer literal does
// (literal_restriction.go), so that let's uses decide the variant.

// NumVariantEager bounds the variables a binding gets every variant over
// (2^k - 1 copies of its body). A binding over more (a top-level record of
// nine integer fields has nine) gets only the variants its own module uses,
// so another module may use it only at int (NumVariantLimitCode).
const NumVariantEager = 4

// numericVariantState is the CoreTypeChecker's numeric-variant bookkeeping.
type numericVariantState struct {
	numSchemes map[*Scheme]string    // module declaration's scheme → its name
	numDecls   map[string][]string   // module declaration → NumVariantVars, as its body names them
	numRefs    map[uint64]NumericRef // reference node → the binding it uses
	// genericTI / genericRC: a defaulted node's type, and its class
	// constraints, before top-level defaulting replaced their variables.
	genericTI map[uint64]Type
	genericRC map[uint64][]ClassConstraint
	// defaults: every variable top-level defaulting replaced, and with what.
	defaults Substitution
	// lastSnapTI / lastSnapRC: the ids the last declaration's snapshot wrote
	// (closeGeneric).
	lastSnapTI, lastSnapRC []uint64
	// renamed: a variable free in the environment (a declaration's own
	// variable, seen from its contract or test) that a later inference bound
	// to another variable, which defaulting then replaced: the variant of the
	// first is the variant of the second.
	renamed map[string]string
}

// NumericRef is the binding a reference node uses, when that binding has
// numeric variants. Module is "" for a binding of the module being checked.
type NumericRef struct {
	Module string
	Name   string
	Scheme *Scheme
	Where  string // the use's source position
}

// NumVariantVars returns the quantified variables of s that a Num constraint,
// and no Fractional one, ranges over, in first-appearance order in s's type.
// These are the variables a binding has numeric variants over (Fractional has
// one instance, float, which is already its default). The exporting and the
// importing module both compute it from the scheme, so they agree on a
// variant's name without the interface naming it.
func NumVariantVars(s *Scheme) []string {
	if s == nil || len(s.Constraints) == 0 || len(s.TypeVars) == 0 {
		return nil
	}
	quantified := make(map[string]bool, len(s.TypeVars))
	for _, v := range s.TypeVars {
		quantified[v] = true
	}
	num, frac := map[string]bool{}, map[string]bool{}
	for _, c := range s.Constraints {
		v := extractVarName(c.Type)
		if v == "" || !quantified[v] {
			continue
		}
		switch c.Class {
		case "Num":
			num[v] = true
		case "Fractional":
			frac[v] = true
		}
	}
	if len(num) == 0 {
		return nil
	}
	var out []string
	for _, v := range FreeTypeVarNames(s.Type) {
		if num[v] && !frac[v] {
			out = append(out, v)
		}
	}
	return out
}

// NumVariantName is the name of name's variant whose i-th variable
// (NumVariantVars order) is float when floats[i]; name itself when none is.
// `$` cannot occur in a source identifier.
func NumVariantName(name string, floats []bool) string {
	any := false
	var b strings.Builder
	for _, f := range floats {
		if f {
			any = true
			b.WriteByte('F')
		} else {
			b.WriteByte('I')
		}
	}
	if !any {
		return name
	}
	return name + "$num" + b.String()
}

// IsNumVariantOf reports whether binding is a numeric variant of name.
func IsNumVariantOf(binding, name string) bool {
	rest, ok := strings.CutPrefix(binding, name+"$num")
	if !ok || rest == "" {
		return false
	}
	for _, r := range rest {
		if r != 'I' && r != 'F' {
			return false
		}
	}
	return true
}

// noteNumericDecl records a module declaration's binding that has numeric
// variants. Only the declaration being checked counts: an inner let is one
// run-time value.
func (tc *CoreTypeChecker) noteNumericDecl(ctx *InferenceContext, binder core.CoreExpr, name string, binding interface{}) {
	s, ok := binding.(*Scheme)
	if !ok || ctx == nil || s == nil || !tc.moduleDeclMode || binder == nil || ctx.declRoot != binder {
		return
	}
	vars := NumVariantVars(s)
	if len(vars) == 0 {
		return
	}
	// A signature variable is written in the scheme as the author wrote it
	// (`a`) but is `a@{f}` in the body's recorded types, which the variants
	// re-read: keep the body's names.
	if sig := tc.RigidSignatureOf(name); sig != nil {
		inner := make(map[string]string, len(sig.Vars))
		for _, v := range sig.Vars {
			if !v.Row {
				inner[v.Display] = v.Name
			}
		}
		for i, v := range vars {
			if n, ok := inner[v]; ok {
				vars[i] = n
			}
		}
	}
	if tc.numSchemes == nil {
		tc.numSchemes = make(map[*Scheme]string)
		tc.numDecls = make(map[string][]string)
	}
	tc.numSchemes[s] = name
	tc.numDecls[name] = vars
}

// NumVariantLimitCode is the diagnostic code of a use, from another module, of
// a binding generic over more than NumVariantEager numeric types at a type
// other than int.
const NumVariantLimitCode = "TC_NUMVAR_001"

// noteNumericRef records that reference node id uses the binding scheme
// belongs to, when that binding has numeric variants, and reports whether it
// does. module is the binding's module ("" for this module's own).
func (tc *CoreTypeChecker) noteNumericRef(id uint64, module, name, where string, s *Scheme) bool {
	if module == "" {
		declName, ok := tc.numSchemes[s]
		if !ok {
			return false
		}
		name = declName
	} else if module == "$builtin" || module == "$adt" || strings.HasPrefix(name, "_") || len(NumVariantVars(s)) == 0 {
		return false
	}
	if tc.numRefs == nil {
		tc.numRefs = make(map[uint64]NumericRef)
	}
	tc.numRefs[id] = NumericRef{Module: module, Name: name, Scheme: s, Where: where}
	return true
}

// NumericDecls returns this module's bindings that have numeric variants,
// each with its NumVariantVars.
func (tc *CoreTypeChecker) NumericDecls() map[string][]string { return tc.numDecls }

// NumericRefs returns every reference node that uses a binding with numeric
// variants.
func (tc *CoreTypeChecker) NumericRefs() map[uint64]NumericRef { return tc.numRefs }

// snapshotGeneric records, for the declaration just inferred, the type each
// node and class constraint had before top-level defaulting (defaulting)
// replaced its variables: the generic body a numeric variant re-reads.
func (tc *CoreTypeChecker) snapshotGeneric(sub, defaulting Substitution, unsolved []ClassConstraint) {
	tc.lastSnapTI, tc.lastSnapRC = tc.lastSnapTI[:0], tc.lastSnapRC[:0]
	if len(defaulting) == 0 {
		return
	}
	if tc.genericTI == nil {
		tc.genericTI = make(map[uint64]Type)
		tc.genericRC = make(map[uint64][]ClassConstraint)
		tc.defaults = make(Substitution)
	}
	for v, t := range defaulting {
		tc.defaults[v] = t
	}
	for v, t := range sub {
		if w := extractVarName(ApplySubstitution(sub, t)); w != "" && w != v && defaulting[w] != nil {
			if tc.renamed == nil {
				tc.renamed = make(map[string]string)
			}
			tc.renamed[v] = w
		}
	}
	mentions := func(t Type) bool {
		for _, v := range FreeTypeVarNames(t) {
			if defaulting[v] != nil {
				return true
			}
		}
		return false
	}
	for _, id := range tc.declNodes {
		t, ok := tc.CoreTI[id]
		if !ok {
			continue
		}
		if g := ApplySubstitution(sub, t); mentions(g) {
			tc.genericTI[id] = g
			tc.lastSnapTI = append(tc.lastSnapTI, id)
		}
	}
	for _, c := range unsolved {
		if c.NodeID == 0 {
			continue
		}
		if g := ApplySubstitution(sub, c.Type); mentions(g) {
			tc.genericRC[c.NodeID] = append(tc.genericRC[c.NodeID], ClassConstraint{Class: c.Class, Type: g, NodeID: c.NodeID})
			tc.lastSnapRC = append(tc.lastSnapRC, c.NodeID)
		}
	}
}

// closeGeneric applies to the declaration's snapshots what the declaration's
// final substitution adds after defaulting (closing its lambdas' open effect
// tails, #1958), so a variant's lambda carries the effect row the original's
// does; the defaulted variables themselves stay unapplied.
func (tc *CoreTypeChecker) closeGeneric(final Substitution) {
	if len(tc.lastSnapTI) == 0 && len(tc.lastSnapRC) == 0 {
		return
	}
	s := make(Substitution, len(final))
	for v, t := range final {
		if tc.defaults[v] == nil {
			s[v] = t
		}
	}
	for _, id := range tc.lastSnapTI {
		tc.genericTI[id] = ApplySubstitution(s, tc.genericTI[id])
	}
	for _, id := range tc.lastSnapRC {
		cs := tc.genericRC[id]
		for i := range cs {
			cs[i].Type = ApplySubstitution(s, cs[i].Type)
		}
	}
	tc.lastSnapTI, tc.lastSnapRC = tc.lastSnapTI[:0], tc.lastSnapRC[:0]
}

// finalizeGeneric applies the deferred let-boundary substitution to the
// snapshots, as FinalizeSubstitutions does to CoreTI. The deferred
// substitution also carries top-level defaulting's own bindings; those are
// exactly what the snapshots keep unapplied.
func (tc *CoreTypeChecker) finalizeGeneric(pending Substitution) {
	sub := make(Substitution, len(pending))
	for v, t := range pending {
		if tc.defaults[v] == nil {
			sub[v] = t
		}
	}
	if len(sub) == 0 {
		return
	}
	for id, t := range tc.genericTI {
		tc.genericTI[id] = ApplySubstitution(sub, t)
	}
	for id, cs := range tc.genericRC {
		for i := range cs {
			cs[i].Type = ApplySubstitution(sub, cs[i].Type)
		}
		tc.genericRC[id] = cs
	}
}

// variantSubst is sigma followed by the recorded defaults: the type of a
// generic body's node in the variant where sigma's variables are float.
func (tc *CoreTypeChecker) variantSubst(t Type, sigma Substitution) Type {
	if len(sigma) > 0 {
		s := make(Substitution, len(sigma))
		for v, f := range sigma {
			s[v] = f
			if w, ok := tc.renamed[v]; ok {
				s[w] = f
			}
		}
		t = ApplySubstitution(s, t)
	}
	return ApplySubstitution(tc.defaults, t)
}

// VariantType returns node id's type in the variant sigma (variable → float),
// and false when the node's type did not depend on a defaulted variable (it is
// then the same in every variant: CoreTI's).
func (tc *CoreTypeChecker) VariantType(id uint64, sigma Substitution) (Type, bool) {
	g, ok := tc.genericTI[id]
	if !ok {
		return nil, false
	}
	return tc.variantSubst(g, sigma), true
}

// VariantConstraint returns the ground type of node id's class constraint in
// the variant sigma, and false when it did not depend on a defaulted variable.
func (tc *CoreTypeChecker) VariantConstraint(id uint64, class string, sigma Substitution) (Type, bool) {
	for _, c := range tc.genericRC[id] {
		if c.Class == class || (class == "Fractional" && c.Class == "Num") {
			return tc.variantSubst(c.Type, sigma), true
		}
	}
	return nil, false
}

// VariantResolved returns the resolved constraint of node id in the variant
// sigma: the original with its type (and Num → Fractional at float, as
// resolveGroundConstraints does) read from the generic body; nil when the node
// has none.
func (tc *CoreTypeChecker) VariantResolved(id uint64, sigma Substitution) *ResolvedConstraint {
	rc, ok := tc.resolvedConstraints[id]
	if !ok {
		return nil
	}
	out := *rc
	t, ok := tc.VariantConstraint(id, rc.ClassName, sigma)
	if !ok {
		return &out
	}
	// As resolveGroundConstraints names it: a synthesized Eq (list, tuple,
	// Option…) dispatches to the one structural dictionary.
	name := NormalizeTypeName(t)
	if inst, err := tc.instanceEnv.Lookup(out.ClassName, t); err == nil && inst.Structural {
		name = StructuralEqTypeName
	}
	out.Type = &TCon{Name: name}
	if name == "Float" && out.ClassName == "Num" {
		out.ClassName = "Fractional"
	}
	return &out
}

// AddResolvedConstraint records rc for node id (a variant's copy of a node).
func (tc *CoreTypeChecker) AddResolvedConstraint(id uint64, rc *ResolvedConstraint) {
	if rc == nil {
		return
	}
	c := *rc
	c.NodeID = id
	tc.resolvedConstraints[id] = &c
}

// NumericRefInstance returns, for reference node id in the variant sigma (nil:
// the binding as compiled), whether each NumVariantVars variable of the binding
// it uses is float there. The instantiation is read by matching the binding's
// scheme against the node's type.
func (tc *CoreTypeChecker) NumericRefInstance(id uint64, sigma Substitution) ([]bool, error) {
	ref, ok := tc.numRefs[id]
	if !ok {
		return nil, nil
	}
	nodeType, ok := tc.VariantType(id, sigma)
	if !ok {
		if nodeType, ok = tc.CoreTI[id]; !ok {
			return nil, fmt.Errorf("numeric variant: no type recorded for the use of %s", ref.Name)
		}
	}
	vars := NumVariantVars(ref.Scheme)
	want := make(map[string]bool, len(vars))
	for _, v := range vars {
		want[v] = true
	}
	bound := map[string]Type{}
	matchSchemeVars(ref.Scheme.Type, nodeType, want, bound, 0)
	floats := make([]bool, len(vars))
	for i, v := range vars {
		t, ok := bound[v]
		if !ok {
			return nil, fmt.Errorf("numeric variant: cannot read the instance of %s's type variable %s from %s", ref.Name, v, nodeType)
		}
		switch NormalizeTypeName(StripLabel(t)) {
		case "Int":
		case "Float":
			floats[i] = true
		default:
			return nil, fmt.Errorf("numeric variant: %s is used with its numeric type variable %s at %s, not int or float", ref.Name, v, t)
		}
	}
	return floats, nil
}

// matchSchemeVars walks a scheme's type and an instance of it side by side,
// binding each variable of want to the instance's type at its position.
// Records are matched by field name, so an open row that the instance
// closed does not matter. Cycle-safety: scheme types are acyclic (they are
// built by substitution, which the occurs check keeps finite), and depth is
// bounded.
func matchSchemeVars(pat, inst Type, want map[string]bool, bound map[string]Type, depth int) {
	if pat == nil || inst == nil || depth > 64 {
		return
	}
	if l, ok := inst.(*TLabelled); ok {
		inst = l.Inner
	}
	switch p := pat.(type) {
	case *TVar:
		if want[p.Name] && bound[p.Name] == nil {
			bound[p.Name] = inst
		}
	case *TVar2:
		if want[p.Name] && bound[p.Name] == nil {
			bound[p.Name] = inst
		}
	case *TLabelled:
		matchSchemeVars(p.Inner, inst, want, bound, depth+1)
	case *TList:
		if i, ok := inst.(*TList); ok {
			matchSchemeVars(p.Element, i.Element, want, bound, depth+1)
		}
	case *TArray:
		if i, ok := inst.(*TArray); ok {
			matchSchemeVars(p.Element, i.Element, want, bound, depth+1)
		}
	case *TMap:
		if i, ok := inst.(*TMap); ok {
			matchSchemeVars(p.Key, i.Key, want, bound, depth+1)
			matchSchemeVars(p.Value, i.Value, want, bound, depth+1)
		}
	case *TTuple:
		if i, ok := inst.(*TTuple); ok && len(i.Elements) == len(p.Elements) {
			for k := range p.Elements {
				matchSchemeVars(p.Elements[k], i.Elements[k], want, bound, depth+1)
			}
		}
	case *TApp:
		if i, ok := inst.(*TApp); ok && len(i.Args) == len(p.Args) {
			matchSchemeVars(p.Constructor, i.Constructor, want, bound, depth+1)
			for k := range p.Args {
				matchSchemeVars(p.Args[k], i.Args[k], want, bound, depth+1)
			}
		}
	case *TFunc2:
		if i, ok := inst.(*TFunc2); ok && len(i.Params) == len(p.Params) {
			for k := range p.Params {
				matchSchemeVars(p.Params[k], i.Params[k], want, bound, depth+1)
			}
			matchSchemeVars(p.Return, i.Return, want, bound, depth+1)
		}
	case *TRecord:
		matchFields(p.Fields, inst, want, bound, depth)
		if r, ok := p.Row.(*Row); ok {
			matchFields(r.Labels, inst, want, bound, depth)
		}
	case *TRecordOpen:
		matchFields(p.Fields, inst, want, bound, depth)
	case *TRecord2:
		if p.Row != nil {
			matchFields(p.Row.Labels, inst, want, bound, depth)
		}
	case *Row:
		matchFields(p.Labels, inst, want, bound, depth)
	}
}

// matchFields matches a scheme record's fields against the instance's fields
// of the same names.
func matchFields(fields map[string]Type, inst Type, want map[string]bool, bound map[string]Type, depth int) {
	have := recordFieldsOf(inst)
	for name, ft := range fields {
		if it, ok := have[name]; ok {
			matchSchemeVars(ft, it, want, bound, depth+1)
		}
	}
}

// recordFieldsOf returns the fields of a record type of any representation,
// its row's labels included.
func recordFieldsOf(t Type) map[string]Type {
	out := map[string]Type{}
	add := func(m map[string]Type) {
		for k, v := range m {
			if _, ok := out[k]; !ok {
				out[k] = v
			}
		}
	}
	switch r := t.(type) {
	case *TRecord:
		add(r.Fields)
		if row, ok := r.Row.(*Row); ok {
			add(row.Labels)
		}
	case *TRecordOpen:
		add(r.Fields)
		if row, ok := r.Row.(*Row); ok {
			add(row.Labels)
		}
	case *TRecord2:
		if r.Row != nil {
			add(r.Row.Labels)
		}
	case *Row:
		add(r.Labels)
	}
	return out
}

// MarkDefaultedNode records node id (a variant's copy of a defaulted node) as
// one whose recorded type stands in for a generic body's (DefaultedNodes).
func (tc *CoreTypeChecker) MarkDefaultedNode(id uint64) {
	if tc.defaulted == nil {
		tc.defaulted = make(map[uint64]bool)
	}
	tc.defaulted[id] = true
}

// NumVariantBase returns the binding a numeric variant's name varies (inc for
// inc$numF), and false for any other name.
func NumVariantBase(name string) (string, bool) {
	i := strings.LastIndex(name, "$num")
	if i <= 0 || !IsNumVariantOf(name, name[:i]) {
		return "", false
	}
	return name[:i], true
}
