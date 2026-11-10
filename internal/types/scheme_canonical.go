package types

import (
	"fmt"
	"maps"
	"slices"
	"sort"
)

// Canonical scheme names at the module boundary (M-V1-TYVAR-HYGIENE, #2076/#2157).
//
// The raw names of one inference context (αN, ρN, εN, and source names such as
// `a` or `e`) used to cross the module boundary as identity: they reached the
// interface digest, so the digest depended on Go map order and on unrelated
// declarations, and an unquantified αN collided with the importer's own αN.
// CanonicalizeScheme renames every variable, in order of first appearance under
// one fixed traversal, into a reserved namespace that neither the lexer nor any
// fresh-variable supply can produce:
//
//	't0 't1 …  type variables          'r0 …  record-row variables
//	'e0 …      effect-row variables    'f0 …  variables the scheme leaves FREE
//
// Free variables get their own spelling so an unclosed export is visible in
// iface.json instead of looking quantified. Whether an unclosed export should be
// an error is decision D-1 of the design doc and is not taken here.
const (
	canonTypeVarPrefix   = "'t"
	canonRecordRowPrefix = "'r"
	canonEffectRowPrefix = "'e"
	canonFreeVarPrefix   = "'f"
)

// varSort classifies a variable occurrence for canonical naming.
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

// orderedSchemeVars lists every variable occurring in t, once each, in the
// canonical traversal order: function params left to right, then the return,
// then the effect row; record fields, row labels and effect labels in sorted
// label order, each row's tail after its labels.
//
// Cycle-safe: a visited set on type pointers stops at any node already walked.
func orderedSchemeVars(t Type) []varOccurrence {
	var out []varOccurrence
	seen := map[string]bool{}
	visited := map[Type]bool{}
	note := func(name string, s varSort, row bool) {
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, varOccurrence{name, s, row})
		}
	}
	var walk func(Type)
	walkRow := func(r *Row) {
		if r == nil || visited[r] {
			return
		}
		visited[r] = true
		for _, l := range slices.Sorted(maps.Keys(r.Labels)) {
			walk(r.Labels[l])
		}
		if r.Tail != nil {
			note(r.Tail.Name, sortOfKind(r.Tail.Kind), true)
		}
	}
	walk = func(t Type) {
		if t == nil {
			return
		}
		if row, ok := t.(*Row); ok {
			walkRow(row)
			return
		}
		if visited[t] {
			return
		}
		visited[t] = true
		switch typ := t.(type) {
		case *TVar:
			note(typ.Name, sortType, false)
		case *TVar2:
			note(typ.Name, sortOfKind(typ.Kind), false)
		case *RowVar:
			note(typ.Name, sortOfKind(typ.Kind), true)
		case *TFunc2:
			for _, p := range typ.Params {
				walk(p)
			}
			walk(typ.Return)
			walkRow(typ.EffectRow)
		case *TList:
			walk(typ.Element)
		case *TArray:
			walk(typ.Element)
		case *TMap:
			walk(typ.Key)
			walk(typ.Value)
		case *TTuple:
			for _, e := range typ.Elements {
				walk(e)
			}
		case *TApp:
			walk(typ.Constructor)
			for _, a := range typ.Args {
				walk(a)
			}
		case *TRecord:
			for _, f := range slices.Sorted(maps.Keys(typ.Fields)) {
				walk(typ.Fields[f])
			}
			walk(typ.Row)
		case *TRecord2:
			walkRow(typ.Row)
		case *TRecordOpen:
			for _, f := range slices.Sorted(maps.Keys(typ.Fields)) {
				walk(typ.Fields[f])
			}
			walk(typ.Row)
		case *TLabelled:
			walk(typ.Inner)
		}
	}
	walk(t)
	return out
}

// renameSchemeVars returns t with every variable named in m renamed. Every
// other field of every node (labels, budgets, provenance, nominal names,
// effect-contract flags) is preserved. Unlike ApplySubstitution it never
// rewrites a row's shape, so it is a pure α-renaming.
//
// Cycle-safe: memo maps each visited node to its result, so a cyclic type is
// rebuilt once and a revisit returns the node in progress.
func renameSchemeVars(t Type, m map[string]string) Type {
	memo := map[Type]Type{}
	var ren func(Type) Type
	renRow := func(r *Row) *Row {
		if r == nil {
			return nil
		}
		if done, ok := memo[r]; ok {
			return done.(*Row)
		}
		cp := *r
		memo[r] = &cp
		if r.Labels != nil {
			cp.Labels = make(map[string]Type, len(r.Labels))
			for l, lt := range r.Labels {
				cp.Labels[l] = ren(lt)
			}
		}
		if r.Tail != nil {
			if to, ok := m[r.Tail.Name]; ok {
				cp.Tail = &RowVar{Name: to, Kind: r.Tail.Kind}
			}
		}
		return &cp
	}
	ren = func(t Type) Type {
		if t == nil {
			return nil
		}
		if row, ok := t.(*Row); ok {
			return renRow(row)
		}
		if done, ok := memo[t]; ok {
			return done
		}
		switch typ := t.(type) {
		case *TVar:
			if to, ok := m[typ.Name]; ok {
				return &TVar{Name: to}
			}
			return t
		case *TVar2:
			if to, ok := m[typ.Name]; ok {
				return &TVar2{Name: to, Kind: typ.Kind}
			}
			return t
		case *RowVar:
			if to, ok := m[typ.Name]; ok {
				return &RowVar{Name: to, Kind: typ.Kind}
			}
			return t
		case *TFunc2:
			cp := *typ
			memo[t] = &cp
			cp.Params = make([]Type, len(typ.Params))
			for i, p := range typ.Params {
				cp.Params[i] = ren(p)
			}
			cp.Return = ren(typ.Return)
			cp.EffectRow = renRow(typ.EffectRow)
			return &cp
		case *TList:
			cp := &TList{}
			memo[t] = cp
			cp.Element = ren(typ.Element)
			return cp
		case *TArray:
			cp := &TArray{}
			memo[t] = cp
			cp.Element = ren(typ.Element)
			return cp
		case *TMap:
			cp := &TMap{}
			memo[t] = cp
			cp.Key, cp.Value = ren(typ.Key), ren(typ.Value)
			return cp
		case *TTuple:
			cp := &TTuple{Elements: make([]Type, len(typ.Elements))}
			memo[t] = cp
			for i, e := range typ.Elements {
				cp.Elements[i] = ren(e)
			}
			return cp
		case *TApp:
			cp := &TApp{Args: make([]Type, len(typ.Args))}
			memo[t] = cp
			cp.Constructor = ren(typ.Constructor)
			for i, a := range typ.Args {
				cp.Args[i] = ren(a)
			}
			return cp
		case *TRecord:
			cp := &TRecord{TypeName: typ.TypeName}
			memo[t] = cp
			if typ.Fields != nil {
				cp.Fields = make(map[string]Type, len(typ.Fields))
				for f, ft := range typ.Fields {
					cp.Fields[f] = ren(ft)
				}
			}
			cp.Row = ren(typ.Row)
			return cp
		case *TRecord2:
			cp := &TRecord2{}
			memo[t] = cp
			cp.Row = renRow(typ.Row)
			return cp
		case *TRecordOpen:
			cp := &TRecordOpen{}
			memo[t] = cp
			if typ.Fields != nil {
				cp.Fields = make(map[string]Type, len(typ.Fields))
				for f, ft := range typ.Fields {
					cp.Fields[f] = ren(ft)
				}
			}
			cp.Row = ren(typ.Row)
			return cp
		case *TLabelled:
			cp := &TLabelled{L: typ.L}
			memo[t] = cp
			cp.Inner = ren(typ.Inner)
			return cp
		default:
			return t // TCon and other leaves carry no variables
		}
	}
	return ren(t)
}

// CanonicalizeScheme returns an α-equivalent copy of s with canonical variable
// names (see the file comment) and the original names of any variables s leaves
// free. Two α-equivalent schemes canonicalise to identical values, so anything
// derived from the result (iface.json, the interface digest) no longer depends
// on map order or on how many fresh variables earlier declarations used.
//
// Quantifier lists are rewritten in order of first appearance. A quantified
// variable that occurs only in a constraint is named after all type occurrences;
// one that occurs nowhere is dropped (it quantifies nothing). Constraints are
// renamed with the same map and sorted by (class, type).
func CanonicalizeScheme(s *Scheme) (*Scheme, []string) {
	if s == nil {
		return nil, nil
	}
	// A variable keeps the quantifier list it came from; its sort only picks the
	// spelling of a quantified row ('r vs 'e).
	quantified := map[string]bool{}
	isRowVar := map[string]bool{}
	for _, v := range s.TypeVars {
		quantified[v] = true
	}
	for _, v := range s.RowVars {
		quantified[v] = true
		isRowVar[v] = true
	}

	occs := orderedSchemeVars(s.Type)
	// Variables that occur only in constraints, in a deterministic order: by
	// class, then by first appearance within the constraint's own type.
	cons := slices.Clone(s.Constraints)
	sort.SliceStable(cons, func(i, j int) bool { return cons[i].Class < cons[j].Class })
	seen := map[string]bool{}
	for _, o := range occs {
		seen[o.name] = true
	}
	for _, c := range cons {
		for _, o := range orderedSchemeVars(c.Type) {
			if !seen[o.name] && quantified[o.name] {
				seen[o.name] = true
				occs = append(occs, o)
			}
		}
	}

	rename := make(map[string]string, len(occs))
	counters := map[string]int{}
	next := func(prefix string) string {
		n := counters[prefix]
		counters[prefix] = n + 1
		return fmt.Sprintf("%s%d", prefix, n)
	}
	out := &Scheme{TypeVars: []string{}, RowVars: []string{}}
	var free []string
	for _, o := range occs {
		if !quantified[o.name] {
			free = append(free, o.name)
			rename[o.name] = next(canonFreeVarPrefix)
			continue
		}
		switch {
		case isRowVar[o.name] && o.sort == sortRecordRow:
			rename[o.name] = next(canonRecordRowPrefix)
			out.RowVars = append(out.RowVars, rename[o.name])
		case isRowVar[o.name]:
			rename[o.name] = next(canonEffectRowPrefix)
			out.RowVars = append(out.RowVars, rename[o.name])
		default:
			rename[o.name] = next(canonTypeVarPrefix)
			out.TypeVars = append(out.TypeVars, rename[o.name])
		}
	}

	out.Type = renameSchemeVars(s.Type, rename)
	if len(s.Constraints) > 0 {
		out.Constraints = make([]Constraint, len(cons))
		for i, c := range cons {
			out.Constraints[i] = Constraint{Class: c.Class, Type: renameSchemeVars(c.Type, rename)}
		}
		sort.SliceStable(out.Constraints, func(i, j int) bool {
			a, b := out.Constraints[i], out.Constraints[j]
			if a.Class != b.Class {
				return a.Class < b.Class
			}
			return a.Type.String() < b.Type.String()
		})
	}
	return out, free
}

// generalizableVars returns, in first-appearance order, the type variables and
// the record-row variables occurring anywhere in t, including under open records
// (TRecordOpen) and record rows. The old collector (collectFreeVars) skips both,
// so a field type such as std/sem.is_newer's `ver` stayed free in the exported
// scheme and collided with the importer's own variables (#2157). Effect rows are
// generalised separately (freeEffectRowVarsInType).
func generalizableVars(t Type) (typeVars, recordRows []string) {
	for _, o := range orderedSchemeVars(t) {
		switch {
		case !o.row:
			typeVars = append(typeVars, o.name)
		case o.sort == sortRecordRow:
			recordRows = append(recordRows, o.name)
		}
	}
	return typeVars, recordRows
}

// maxRowKindDepth bounds recordRowNames so a cyclic type cannot recurse forever.
const maxRowKindDepth = 64

// recordRowNames adds to out the names of the record-kinded row variables in t.
// Instantiation uses it to freshen each quantified row with its own kind.
//
// Cycle-safe: recursion stops at maxRowKindDepth.
func recordRowNames(t Type, out map[string]bool, depth int) {
	if t == nil || depth > maxRowKindDepth {
		return
	}
	rowTail := func(r *Row) {
		if r == nil {
			return
		}
		if r.Tail != nil && sortOfKind(r.Tail.Kind) == sortRecordRow {
			out[r.Tail.Name] = true
		}
		for _, lt := range r.Labels {
			recordRowNames(lt, out, depth+1)
		}
	}
	switch typ := t.(type) {
	case *RowVar:
		if sortOfKind(typ.Kind) == sortRecordRow {
			out[typ.Name] = true
		}
	case *Row:
		rowTail(typ)
	case *TFunc2:
		for _, p := range typ.Params {
			recordRowNames(p, out, depth+1)
		}
		recordRowNames(typ.Return, out, depth+1)
	case *TList:
		recordRowNames(typ.Element, out, depth+1)
	case *TArray:
		recordRowNames(typ.Element, out, depth+1)
	case *TMap:
		recordRowNames(typ.Key, out, depth+1)
		recordRowNames(typ.Value, out, depth+1)
	case *TTuple:
		for _, e := range typ.Elements {
			recordRowNames(e, out, depth+1)
		}
	case *TApp:
		for _, a := range typ.Args {
			recordRowNames(a, out, depth+1)
		}
	case *TRecord:
		for _, ft := range typ.Fields {
			recordRowNames(ft, out, depth+1)
		}
		recordRowNames(typ.Row, out, depth+1)
	case *TRecord2:
		rowTail(typ.Row)
	case *TRecordOpen:
		for _, ft := range typ.Fields {
			recordRowNames(ft, out, depth+1)
		}
		recordRowNames(typ.Row, out, depth+1)
	case *TLabelled:
		recordRowNames(typ.Inner, out, depth+1)
	}
}

// QuantifiedRecordRows returns the scheme's quantified row variables that are
// record rows, in quantifier order. The iface builder keeps them when it
// recomputes a scheme's effect-row quantifiers after label restoration.
func QuantifiedRecordRows(s *Scheme) []string {
	if s == nil || len(s.RowVars) == 0 {
		return nil
	}
	recordRows := map[string]bool{}
	recordRowNames(s.Type, recordRows, 0)
	var out []string
	for _, v := range s.RowVars {
		if recordRows[v] {
			out = append(out, v)
		}
	}
	return out
}
