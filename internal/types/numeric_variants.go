package types

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
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
//     fixed order (NumVariantVars): the Num variables that carry an integer
//     literal, directly or through a callee (Constraint.Literal). The flag is
//     exported with the scheme, so an importer computes the same variables;
//     a literal-free generic (`add(a, b) = a + b`) has none, its one body
//     being right at every instance;
//   - every reference to such a binding, in-module or imported (NumericRefs),
//     whose instantiation picks the variant (NumericRefInstance);
//   - the type each node of a defaulted generic body had before defaulting
//     (VariantType, VariantConstraint), from which a variant's node types and
//     dictionaries are read.
//
// A let-bound function (a non-expansive value ending in a lambda, the values
// the generaliser generalises) follows the same rule as a top-level one: its
// literal-carrying parameter variables get a copy per use (noteNumericDecl).
// A literal-carrying variable that occurs in no parameter position of the
// binding's type is not generic at all: defaulted to int at a module
// declaration (declResultOnlyVars), decided by its uses at an inner let
// (literalWithhold).

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
	// innerSchemes / innerVars: let-bound functions copied per use (node id of
	// their let or letrec).
	innerSchemes map[*Scheme]uint64
	innerVars    map[uint64][]string
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
	renamed map[string][]string
	// forcedPending / forcedSchemes: declarations whose literal-carrying
	// variable was defaulted to int, and where the literal is; declForcedUses:
	// those the declaration being checked uses (numeric_diagnostics.go).
	forcedPending  map[string]string
	forcedSchemes  map[*Scheme]forcedLiteral
	declForcedUses []forcedLiteral
	// variantsPlanned: the caller compiles numeric variants, let-bound
	// functions' per-use copies included (SetNumericVariants).
	variantsPlanned bool
}

// SetNumericVariants tells the checker that its caller compiles numeric
// variants (the module pipeline does). Only then is a let-bound function with
// a literal-carrying parameter variable generalised and copied per use; any
// other caller (expression mode, the REPL, a contract body) keeps it at one
// type decided by its uses, so it never runs an int body at float.
func (tc *CoreTypeChecker) SetNumericVariants(on bool) { tc.variantsPlanned = on }

// copiesPerUse reports whether binder is a let-bound function this check
// copies per use (specialisableLet, inside a module declaration of a caller
// that compiles variants).
func (tc *CoreTypeChecker) copiesPerUse(ctx *InferenceContext, binder core.CoreExpr) bool {
	return tc.variantsPlanned && tc.moduleDeclMode && ctx != nil && ctx.declRoot != nil && specialisableLet(binder)
}

// NumericRef is the binding a reference node uses, when that binding has
// numeric variants. Module is "" for a binding of the module being checked.
type NumericRef struct {
	Module   string
	Name     string
	Scheme   *Scheme
	Where    string  // the use's source position
	Pos      ast.Pos // the same, for a located report
	InnerLet uint64  // the let node, for a let-bound lambda copied per use
}

// NumVariantVars returns the quantified variables of s that a Literal Num
// constraint, and no Fractional one, ranges over, in first-appearance order in
// s's type. These are the variables a binding has numeric variants over: its
// int body is wrong at float there (Fractional has one instance, float, which
// is already its default; a literal-free variable's body is right at both).
// The exporting and the importing module both compute it from the scheme, the
// flag included, so they agree on a variant's name.
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
			if c.Literal {
				num[v] = true
			}
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

// noteNumericDecl records a binding that has numeric variants: a module
// declaration's, or a let-bound function's. Its variables are the scheme's
// Literal ones (NumVariantVars), whose int body is wrong at float.
//
// A let-bound function is copied per use over them, as a top-level one is
// compiled per instance (pipeline/numeric_variants_clone.go): `let h = \u.
// u + 1` and `let g = \u. inc(u)` run as h(2) == 3 and h(2.5) == 3.5. A
// literal-free one is one run-time value, corrected from its run-time values
// as before (M-POLY-ARITH), and keeps exactly the code it had.
func (tc *CoreTypeChecker) noteNumericDecl(ctx *InferenceContext, binder core.CoreExpr, name string, binding interface{}) {
	tc.registerForced(name, binding)
	s, ok := binding.(*Scheme)
	if !ok || ctx == nil || s == nil || binder == nil {
		return
	}
	vars := NumVariantVars(s)
	if len(vars) == 0 {
		return
	}
	declared := tc.moduleDeclMode && ctx.declRoot == binder
	if !declared && !tc.copiesPerUse(ctx, binder) {
		return
	}
	// A signature variable is written in the scheme as the author wrote it
	// (`a`) but is `a@{f}` in the body's recorded types, which the variants
	// re-read: keep the body's names. (Mutation-tested as equivalent so far:
	// the scheme keeps the written name only when nothing bound `a@{f}` to
	// another variable, and a literal at it is TC_RIGID_001 while a call of a
	// literal-carrying binding at it binds the call's variable to it, which
	// `renamed` then maps; what is left are operators, which the run-time
	// correction already computes at float. The mapping keeps the variant's
	// types right without relying on that.)
	body := append([]string(nil), vars...)
	if sig := tc.RigidSignatureOf(name); declared && sig != nil {
		inner := make(map[string]string, len(sig.Vars))
		for _, v := range sig.Vars {
			if !v.Row {
				inner[v.Display] = v.Name
			}
		}
		for i, v := range body {
			if n, ok := inner[v]; ok {
				body[i] = n
			}
		}
	}
	if tc.numSchemes == nil {
		tc.numSchemes = make(map[*Scheme]string)
		tc.numDecls = make(map[string][]string)
		tc.innerSchemes = make(map[*Scheme]uint64)
		tc.innerVars = make(map[uint64][]string)
	}
	if declared {
		tc.numSchemes[s] = name
		tc.numDecls[name] = body
		return
	}
	tc.innerSchemes[s] = binder.ID()
	tc.innerVars[binder.ID()] = body
}

// specialisableLet reports whether binder binds a let-bound function a use can
// be given its own copy of: a let whose value is non-expansive (isCoreValue,
// the values the generaliser generalises) and ends in a lambda (`\u. …`, or a
// block of values ending in one), or a letrec of one such binding. An
// expansive value (`if … then \u. … else …`, a call) is one run-time value.
func specialisableLet(binder core.CoreExpr) bool {
	switch b := binder.(type) {
	case *core.Let:
		return isCoreValue(b.Value) && endsInLambda(b.Value)
	case *core.LetRec:
		return len(b.Bindings) == 1 && isCoreValue(b.Bindings[0].Value) && endsInLambda(b.Bindings[0].Value)
	}
	return false
}

// endsInLambda reports whether e is a lambda, or lets whose body is one.
func endsInLambda(e core.CoreExpr) bool {
	for depth := 0; depth < 256; depth++ {
		switch x := e.(type) {
		case *core.Lambda:
			return true
		case *core.Let:
			e = x.Body
		case *core.LetRec:
			e = x.Body
		default:
			return false
		}
	}
	return false
}

// InnerLetVars returns the numeric variables a let-bound lambda (node id of
// its let) is copied over per use, nil when it is not.
func (tc *CoreTypeChecker) InnerLetVars(id uint64) []string { return tc.innerVars[id] }

// NumVariantLimitCode is the diagnostic code of a use, from another module, of
// a binding at numeric types its module compiled no variant for.
const NumVariantLimitCode = "TC_NUMVAR_001"

// noteNumericRef records that reference node id uses the binding scheme
// belongs to, when that binding has numeric variants. module is the binding's
// module ("" for this module's own); another module's binding has variants
// over the variables its exported scheme flags Literal.
func (tc *CoreTypeChecker) noteNumericRef(id uint64, module, name string, pos ast.Pos, s *Scheme) {
	ref := NumericRef{Module: module, Name: name, Scheme: s, Where: pos.String(), Pos: pos}
	switch {
	case module != "":
		if module == "$builtin" || module == "$adt" || strings.HasPrefix(name, "_") || len(NumVariantVars(s)) == 0 {
			return
		}
	case tc.numSchemes[s] != "":
		ref.Name = tc.numSchemes[s]
	case tc.innerSchemes[s] != 0:
		ref.InnerLet = tc.innerSchemes[s]
	default:
		return
	}
	if tc.numRefs == nil {
		tc.numRefs = make(map[uint64]NumericRef)
	}
	tc.numRefs[id] = ref
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
				tc.renamed = make(map[string][]string)
			}
			// One per inference that bound v: each contract clause and test
			// body binds the declaration's variable to its own.
			tc.renamed[v] = append(tc.renamed[v], w)
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
			for _, w := range tc.renamed[v] {
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
	// Num → Fractional at Float, exactly as resolveGroundConstraints writes a
	// monomorphic float body's constraints, so a variant's dictionary
	// references match its twin's. (Mutation-tested equivalent: Num[float]
	// and Fractional[float] register the same add/sub/mul/div/neg/abs/fromInt.)
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

// declResultOnlyVars returns, when binder is the module declaration being
// checked, the numeric variables defaulting settles instead of generalising
// (the #2378 rule): those that BOTH carry an integer literal (directly, or
// through a callee's literal-carrying variable: IntLiteral or PolyNumeric) AND
// occur in no parameter position of any function type anywhere in the
// binding's type (paramPositionVars). Such a variable has no argument to tell
// a call its type (`let K = 3`, `func cfg() = {lo: 1}`, `func one(x) = 1`'s
// result), so it is int (float for `3.0`). Everything else stays generic as
// it was: a literal-free variable (`let ops = (add, mul)`) and one that a
// function in the result takes (`func mk() = \x. x + 1`). In a recursive
// group, a parameter position of any member's. nil for any other binder.
func (tc *CoreTypeChecker) declResultOnlyVars(ctx *InferenceContext, binder core.CoreExpr, sub Substitution, constraints []ClassConstraint, valueTypes ...Type) map[string]bool {
	if ctx == nil || !tc.moduleDeclMode || binder == nil || ctx.declRoot != binder {
		return nil
	}
	lit := map[string]string{} // variable → where its first literal is
	for _, c := range constraints {
		if c.Class != "Num" || (!c.IntLiteral && !c.PolyNumeric) {
			continue
		}
		vars := map[string]bool{}
		collectFreeTypeVars(ApplySubstitution(sub, c.Type), vars)
		for v := range vars {
			if _, ok := lit[v]; !ok && len(c.Path) > 0 {
				lit[v] = c.Path[0]
			}
		}
	}
	if len(lit) == 0 {
		return nil
	}
	params, all := map[string]bool{}, map[string]bool{}
	for _, t := range valueTypes {
		collectFreeTypeVars(t, all)
		paramPositionVars(t, params, 0)
	}
	var out map[string]bool
	for v := range all {
		if where, ok := lit[v]; ok && !params[v] {
			if out == nil {
				out = map[string]bool{}
			}
			out[v] = true
			tc.noteForcedLiteral(binder, where)
		}
	}
	return out
}

// paramPositionVars adds to out the type variables occurring in a parameter
// type of any function type within t: the spine's, and those of functions in
// tuples, lists, records, options, results and other functions' parameters or
// results. Cycle-safety: types are finite trees after substitution; depth
// bounds a malformed one.
func paramPositionVars(t Type, out map[string]bool, depth int) {
	if t == nil || depth > 64 {
		return
	}
	switch x := t.(type) {
	case *TLabelled:
		paramPositionVars(x.Inner, out, depth+1)
	case *TFunc2:
		for _, p := range x.Params {
			collectFreeTypeVars(p, out)
			paramPositionVars(p, out, depth+1)
		}
		paramPositionVars(x.Return, out, depth+1)
	case *TList:
		paramPositionVars(x.Element, out, depth+1)
	case *TArray:
		paramPositionVars(x.Element, out, depth+1)
	case *TMap:
		paramPositionVars(x.Key, out, depth+1)
		paramPositionVars(x.Value, out, depth+1)
	case *TTuple:
		for _, e := range x.Elements {
			paramPositionVars(e, out, depth+1)
		}
	case *TApp:
		paramPositionVars(x.Constructor, out, depth+1)
		for _, a := range x.Args {
			paramPositionVars(a, out, depth+1)
		}
	case *TRecord, *TRecordOpen, *TRecord2, *Row:
		for _, ft := range recordFieldsOf(t) {
			paramPositionVars(ft, out, depth+1)
		}
	}
}
