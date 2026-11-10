package types

import (
	"maps"
	"slices"
)

// One definition of "the variables that occur in a type".
//
// Hindley-Milner generalisation asks two questions of the same type forms:
// which variables occur in the type being generalised, and which of them the
// environment owns. The rule is sound only if both answers come from the same
// definition of "occur". They used to come from separate switch statements that
// drifted apart. The environment walk had no case for records (closed or open),
// record rows or type applications, so a variable owned by an enclosing
// parameter of type `{name: a}`, `{name: a, ...}` or `Option[a]` was invisible
// to it. `let q = p` then generalised `a`, and `q.name + 1` type-checked
// against a string field (PR #2209 review; the closed-record and Option forms
// predate that PR).
//
// varWalker (walkType) is now the only traversal. Every collector of free
// variables is a filter over it: the generalisation side (generalizableVars,
// freeEffectRowVarsInType), the environment side (collectFreeTypeVars and the
// row walks behind TypeEnv's memoised sets), instantiation's kind lookup
// (recordRowNames) and the module-boundary canonicaliser (CanonicalizeScheme).
// TestFreeVars_EveryTypeConstructorIsWalked fails when a Type implementation
// is added without a case here.

// varSort classifies a variable occurrence.
type varSort int

const (
	sortType varSort = iota
	sortRecordRow
	sortEffectRow
)

func sortOfKind(k Kind) varSort {
	switch {
	case k == nil:
		return sortType
	case k.Equals(RecordRow):
		return sortRecordRow
	case k.Equals(EffectRow):
		return sortEffectRow
	default:
		return sortType
	}
}

type varOccurrence struct {
	name string
	sort varSort
	row  bool // occurs as a row variable (RowVar or a row tail), not a TVar/TVar2
}

// varWalker is the one traversal. walkType visits every variable occurrence
// in a type (a name can be visited more than once). With ordered set, record
// fields and row labels are visited in sorted label order, which canonical
// naming needs; the collectors that only build sets skip the sort, because
// they run on every binding of every environment frame.
//
// Cycle-safe: visited (pointer identity) stops at any node already walked.
type varWalker struct {
	ordered bool
	visited nodeSet
	visit   func(name string, s varSort, row bool)
}

// nodeSet is a set of type nodes by pointer identity. Most types have a
// handful of nodes, so the first ones live in an inline array and a map is
// only allocated for large types; this walk runs on every binding of every
// environment frame.
type nodeSet struct {
	small [16]Type
	n     int
	big   map[Type]bool
}

// add records t and reports whether it was already present.
func (s *nodeSet) add(t Type) bool {
	if s.big != nil {
		if s.big[t] {
			return true
		}
		s.big[t] = true
		return false
	}
	for i := 0; i < s.n; i++ {
		if s.small[i] == t {
			return true
		}
	}
	if s.n < len(s.small) {
		s.small[s.n] = t
		s.n++
		return false
	}
	s.big = make(map[Type]bool, 2*len(s.small))
	for _, x := range s.small {
		s.big[x] = true
	}
	s.big[t] = true
	return false
}

func (w *varWalker) walkFields(fields map[string]Type) {
	if !w.ordered {
		for _, ft := range fields {
			w.walkType(ft)
		}
		return
	}
	for _, f := range slices.Sorted(maps.Keys(fields)) {
		w.walkType(fields[f])
	}
}

func (w *varWalker) walkRow(r *Row) {
	if r == nil || w.visited.add(r) {
		return
	}
	w.walkFields(r.Labels)
	if r.Tail != nil {
		w.visit(r.Tail.Name, sortOfKind(r.Tail.Kind), true)
	}
}

func (w *varWalker) walkType(t Type) {
	if t == nil {
		return
	}
	if row, ok := t.(*Row); ok {
		w.walkRow(row)
		return
	}
	if w.visited.add(t) {
		return
	}
	switch typ := t.(type) {
	case *TVar:
		w.visit(typ.Name, sortType, false)
	case *TVar2:
		w.visit(typ.Name, sortOfKind(typ.Kind), false)
	case *RowVar:
		w.visit(typ.Name, sortOfKind(typ.Kind), true)
	case *TCon:
		// no variables
	case *TFunc2:
		for _, p := range typ.Params {
			w.walkType(p)
		}
		w.walkType(typ.Return)
		w.walkRow(typ.EffectRow)
	case *TList:
		w.walkType(typ.Element)
	case *TArray:
		w.walkType(typ.Element)
	case *TMap:
		w.walkType(typ.Key)
		w.walkType(typ.Value)
	case *TTuple:
		for _, e := range typ.Elements {
			w.walkType(e)
		}
	case *TApp:
		w.walkType(typ.Constructor)
		for _, a := range typ.Args {
			w.walkType(a)
		}
	case *TRecord:
		w.walkFields(typ.Fields)
		w.walkType(typ.Row)
	case *TRecord2:
		w.walkRow(typ.Row)
	case *TRecordOpen:
		w.walkFields(typ.Fields)
		w.walkType(typ.Row)
	case *TLabelled:
		w.walkType(typ.Inner)
	}
}

// forEachVar runs the traversal over t, calling visit for each occurrence.
func forEachVar(t Type, ordered bool, visit func(name string, s varSort, row bool)) {
	w := varWalker{ordered: ordered, visit: visit}
	w.walkType(t)
}

// typeVarOccurrences lists every variable occurring in t, once each, in a
// fixed order: function params left to right, then the return, then the effect
// row; record fields, row labels and effect labels in sorted label order, each
// row's tail after its labels.
func typeVarOccurrences(t Type) []varOccurrence {
	var out []varOccurrence
	seen := map[string]bool{}
	forEachVar(t, true, func(name string, s varSort, row bool) {
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, varOccurrence{name, s, row})
		}
	})
	return out
}

// The is*Occ predicates are the filters the collectors below apply to the
// traversal.
func isTypeVarOcc(o varOccurrence) bool   { return !o.row }
func isRowOcc(o varOccurrence) bool       { return o.row }
func isEffectRowOcc(o varOccurrence) bool { return o.row && o.sort == sortEffectRow }
func isRecordRowOcc(o varOccurrence) bool { return o.row && o.sort == sortRecordRow }

// collectVars adds to free every variable of t that keep accepts, except the
// names in bound (a scheme's quantifiers).
func collectVars(t Type, free map[string]bool, keep func(varOccurrence) bool, bound []string) {
	forEachVar(t, false, func(name string, s varSort, row bool) {
		if name != "" && keep(varOccurrence{name, s, row}) && !slices.Contains(bound, name) {
			free[name] = true
		}
	})
}

// collectFreeTypeVars adds the type variables of t to free.
func collectFreeTypeVars(t Type, free map[string]bool) {
	collectVars(t, free, isTypeVarOcc, nil)
}

// freeTypeVars returns the type variables of t.
func freeTypeVars(t Type) map[string]bool {
	free := make(map[string]bool)
	collectFreeTypeVars(t, free)
	return free
}

// freeRowVars returns the row variables of every kind occurring in r.
func freeRowVars(r *Row) map[string]bool {
	free := make(map[string]bool)
	if r != nil {
		collectVars(r, free, isRowOcc, nil)
	}
	return free
}

// freeEffectRowVarsInType returns the EffectRow-kinded row variables of a
// complete type: outer and nested callback effect rows, and rows nested in
// collections, tuples, records (closed and open), labels and type applications.
// M-EFFECT-ROW-SHOW-INTERP (#386) Section B quantifies these so each use of a
// row-polymorphic combinator gets fresh rows.
func freeEffectRowVarsInType(t Type) map[string]bool {
	free := make(map[string]bool)
	collectVars(t, free, isEffectRowOcc, nil)
	return free
}

// recordRowNames adds to out the record-kinded row variables of t.
// Instantiation uses it to freshen each quantified row with its own kind.
func recordRowNames(t Type, out map[string]bool) {
	collectVars(t, out, isRecordRowOcc, nil)
}

// generalizableVars returns, in first-appearance order, the type variables and
// the record-row variables occurring anywhere in t. Effect rows are generalised
// separately (freeEffectRowVarsInType). The environment side of the same
// question is TypeEnv.ownedVars, built from the same traversal.
func generalizableVars(t Type) (typeVars, recordRows []string) {
	for _, o := range typeVarOccurrences(t) {
		switch {
		case isTypeVarOcc(o):
			typeVars = append(typeVars, o.name)
		case isRecordRowOcc(o):
			recordRows = append(recordRows, o.name)
		}
	}
	return typeVars, recordRows
}

// varsUnderSubstitution returns the variables an environment owns once sub is
// applied to it: each variable of the given owned sets, plus every variable of
// what sub maps it to. Generalisation withholds these. Nil sets are skipped.
func varsUnderSubstitution(sub Substitution, owned ...map[string]bool) map[string]bool {
	out := make(map[string]bool)
	for _, set := range owned {
		for v := range set {
			out[v] = true
			if t, ok := sub[v]; ok {
				for _, o := range typeVarOccurrences(ApplySubstitution(sub, t)) {
					out[o.name] = true
				}
			}
		}
	}
	return out
}
