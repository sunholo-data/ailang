package types

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// One record unification (#2210, #2221).
//
// The checker has three record representations: TRecord (literals and
// annotations, optionally open through its Row), TRecordOpen (the shim a field
// access or record update builds, `{x: a | ρ}`) and TRecord2 (a bare row). Each
// pair used to have its own unifier, and two of them were unsound:
//
//   - TRecordOpen ~ TRecord checked the shim's fields and never bound the shim's
//     row (a "Day 2" placeholder), so a variable bound to a shim was never
//     refined by the closed record it met. `best() -> {label, score}` exported
//     the shim `{score: float | ρ}`, and `best().bogus` type-checked (#2210).
//   - TRecordOpen ~ TRecordOpen unified the shared fields and then the two rows
//     with each other, ignoring the fields only one side has. `{x: int | ρ}`
//     against `{y: string | ρ'}` bound ρ := ρ', so a record update lost the
//     other fields and `.y + 1` checked against a string (#2221 part 2).
//
// Every pair now goes through recordShapeOf, which flattens any of the three
// forms into labels plus an optional tail, and unifyRecordShapes, which is the
// textbook row unification: shared labels unify pointwise; each side's extra
// labels must flow into the other side's tail; a closed side absorbs nothing;
// two open sides share one fresh remainder. Every binding goes through the
// occurs check, so `ρ := {self: … ρ …}` is an error rather than a cycle.

// Diagnostic codes (the existing registered ones, errors.go and
// internal/errors):
//   - TC_REC_001: a record lacks a field that a field access, a record update,
//     an open pattern or an open annotation requires. Before #2210 some of
//     these type-checked and failed at run time (`record has no field`).
//   - TC_REC_003: a record row would have to contain itself.
//   - TC_REC_004: a field would have two types, because a row variable shared
//     by two records was bound to a field one of them already has (a "lacks"
//     violation: `{x: int | r}` with r := {x: string}).
//   - TC_OCCURS (TC004): a type variable would have to contain itself
//     (`\x. {self: x} == x`). Before #2221 the record forms had no occurs case
//     and `check` never terminated.

// TC_OCCURS is the registered occurs-check code (internal/errors TC004).
const TC_OCCURS = "TC004"

// RecordRowCodes lists the inline diagnostic codes of record unification and
// the occurs check. `check --json` reports each as itself.
func RecordRowCodes() []string {
	return []string{TC_REC_001, TC_REC_003, TC_REC_004, TC_OCCURS}
}

// recordShape is a record type flattened to its labels and its tail. A nil
// tail is a closed record.
type recordShape struct {
	labels map[string]Type
	tail   *RowVar
	// dups are labels the record has twice: in its fields (or an earlier row
	// link) and again in the row its tail is bound to. A row variable must
	// lack the labels of every record it extends, so the two must be one
	// field: unifyRecordShapes unifies them (TC_REC_004 when they differ).
	dups []dupLabel
	// owned is false while labels is still the record's own Fields map
	// (shared, never written); addMissing copies it before the first insert.
	owned bool
}

// dupLabel is one label a record has twice, with both types.
type dupLabel struct {
	name        string
	first, more Type
}

// isRecordType reports whether t is one of the three record representations.
func isRecordType(t Type) bool {
	switch t.(type) {
	case *TRecord, *TRecordOpen, *TRecord2:
		return true
	}
	return false
}

// recordShapeOf flattens a record type, resolving its tail through sub. It
// reports false for a record whose row is not a row form (a malformed type).
func recordShapeOf(t Type, sub Substitution) (recordShape, bool) {
	var s recordShape
	var row Type
	switch r := t.(type) {
	case *TRecord:
		s.labels, row = r.Fields, r.Row
	case *TRecordOpen:
		s.labels, row = r.Fields, r.Row
	case *TRecord2:
		if r.Row != nil {
			row = r.Row
		}
	default:
		return s, false
	}
	if s.labels == nil {
		s.labels, s.owned = make(map[string]Type), true
	}
	return s.extend(row, sub)
}

// extend adds a row's labels to s (the fields already there win) and follows
// its tail through sub to the first unbound row variable or a closed end.
//
// Cycle-safe: the chase stops at a row variable already seen.
func (s recordShape) extend(row Type, sub Substitution) (recordShape, bool) {
	seen := make(map[string]bool)
	for row != nil {
		if r, ok := row.(*Row); ok {
			s.addMissing(r.Labels)
			if r.Tail == nil {
				return s, true
			}
			row = r.Tail
			continue
		}
		v, ok := asRecordRowVar(row)
		if !ok {
			return s, false
		}
		next, bound := sub[v.Name]
		if !bound || seen[v.Name] {
			s.tail = v
			return s, true
		}
		seen[v.Name] = true
		row = next
	}
	return s, true
}

// addMissing adds the labels s does not have yet; one it has already, with a
// different type node, is recorded as a duplicate. (A closed record literal
// carries its fields in Row.Labels too, as the same nodes: not a duplicate.)
//
// Not sorted: on a record with hundreds of fields, sorting every row on every
// unification dominated `check` (#2210 review, item 6). Only the duplicates,
// which are unified in order, are sorted.
func (s *recordShape) addMissing(labels map[string]Type) {
	from := len(s.dups)
	for name, typ := range labels {
		first, has := s.labels[name]
		switch {
		case !has:
			if !s.owned {
				s.labels, s.owned = maps.Clone(s.labels), true
			}
			s.labels[name] = typ
		case first != typ:
			s.dups = append(s.dups, dupLabel{name: name, first: first, more: typ})
		}
	}
	slices.SortFunc(s.dups[from:], func(a, b dupLabel) int { return strings.Compare(a.name, b.name) })
}

// asRecordRowVar reads a record's row variable, whichever form carries it.
func asRecordRowVar(row Type) (*RowVar, bool) {
	switch r := row.(type) {
	case *RowVar:
		return r, true
	case *TVar2:
		if r.Kind != nil && r.Kind.Equals(RecordRow) {
			return &RowVar{Name: r.Name, Kind: RecordRow}, true
		}
	case *TVar:
		return &RowVar{Name: r.Name, Kind: RecordRow}, true
	}
	return nil, false
}

// unifyRecordTypes unifies two record types of any representation.
func (u *Unifier) unifyRecordTypes(t1, t2 Type, sub Substitution) (Substitution, error) {
	s1, ok1 := recordShapeOf(t1, sub)
	s2, ok2 := recordShapeOf(t2, sub)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("cannot unify record %s with %s", t1, t2)
	}
	return u.unifyRecordShapes(s1, s2, sub)
}

func (u *Unifier) unifyRecordShapes(s1, s2 recordShape, sub Substitution) (Substitution, error) {
	sub, err := u.unifyDuplicates(s1, s2, sub)
	if err != nil {
		return nil, err
	}
	sub, err = u.unifySharedLabels(s1, s2, sub)
	if err != nil {
		return nil, err
	}
	// Unifying a shared field can bind a tail (`{f: ρ-record | ρ}`); start again
	// from the resolved shapes so the tails below are unbound.
	if tailBound(s1.tail, sub) || tailBound(s2.tail, sub) {
		return u.unifyRecordShapes(reshape(s1, sub), reshape(s2, sub), sub)
	}
	return u.bindRemainders(s1, s2, onlyIn(s1, s2), onlyIn(s2, s1), sub)
}

// unifyDuplicates makes each label a record has twice one field. Without it,
// `func f(p: {y: int | r}, q: {x: int | r}) -> {x: int | r} = q` called with
// ({y: 1, x: "s"}, {x: 2}) typed the result {x: int, x: string}, and which x a
// later access saw depended on which of the two maps was read first.
func (u *Unifier) unifyDuplicates(s1, s2 recordShape, sub Substitution) (Substitution, error) {
	for _, d := range append(slices.Clone(s1.dups), s2.dups...) {
		next, err := u.Unify(d.first, d.more, sub)
		if err != nil {
			return nil, fmt.Errorf("%s: field '%s' would have two types, %s and %s: a row variable shared by two records was bound to a field one of them already has: %w",
				TC_REC_004, d.name, ApplySubstitution(sub, d.first), ApplySubstitution(sub, d.more), err)
		}
		sub = next
	}
	return sub, nil
}

// unifySharedLabels unifies the fields both records have, in label order.
//
// Only the shared names are sorted (for a deterministic order of fresh
// variables), found by walking the smaller record: a field access meets a
// 400-field record with one shared label, not 400 sorted ones.
func (u *Unifier) unifySharedLabels(s1, s2 recordShape, sub Substitution) (Substitution, error) {
	small, large := s1.labels, s2.labels
	if len(small) > len(large) {
		small, large = large, small
	}
	var shared []string
	for name := range small {
		if _, ok := large[name]; ok {
			shared = append(shared, name)
		}
	}
	slices.Sort(shared)
	for _, name := range shared {
		var err error
		sub, err = u.Unify(s1.labels[name], s2.labels[name], sub)
		if err != nil {
			return nil, fmt.Errorf("failed to unify record field '%s': %w", name, err)
		}
	}
	return sub, nil
}

// onlyIn returns the fields a has and b lacks.
func onlyIn(a, b recordShape) map[string]Type {
	out := make(map[string]Type)
	for name, typ := range a.labels {
		if _, shared := b.labels[name]; !shared {
			out[name] = typ
		}
	}
	return out
}

// bindRemainders sends each side's extra fields into the other side's tail.
func (u *Unifier) bindRemainders(s1, s2 recordShape, only1, only2 map[string]Type, sub Substitution) (Substitution, error) {
	switch {
	case s1.tail == nil && s2.tail == nil:
		if len(only1) > 0 || len(only2) > 0 {
			return nil, recordFieldMismatchError(s1.labels, s2.labels)
		}
		return sub, nil
	case s2.tail == nil:
		return u.absorbIntoOpen(s1.tail, only2, only1, s2.labels, sub)
	case s1.tail == nil:
		return u.absorbIntoOpen(s2.tail, only1, only2, s1.labels, sub)
	case s1.tail.Name == s2.tail.Name:
		// {a | ρ} ~ {b | ρ} would need ρ = {b | ρ'} = {a | ρ'}: no finite row.
		if len(only1) > 0 || len(only2) > 0 {
			return nil, fmt.Errorf("%s: occurs check failed: record row %s would have to contain itself (fields %s vs %s)",
				TC_REC_003, s1.tail.Name, labelList(only1), labelList(only2))
		}
		return sub, nil
	default:
		rest := &RowVar{Name: u.freshRowVarName(), Kind: RecordRow}
		var err error
		sub, err = u.bindRecordTail(s1.tail, &Row{Kind: RecordRow, Labels: only2, Tail: rest}, sub)
		if err != nil {
			return nil, err
		}
		return u.bindRecordTail(s2.tail, &Row{Kind: RecordRow, Labels: only1, Tail: rest}, sub)
	}
}

// absorbIntoOpen closes an open record's tail over the closed record's extra
// fields. The open side's own extra fields (missing) are fields the closed
// record does not have.
func (u *Unifier) absorbIntoOpen(tail *RowVar, extras, missing, closed map[string]Type, sub Substitution) (Substitution, error) {
	if len(missing) > 0 {
		return nil, recordMissingFieldsError(missing, closed)
	}
	return u.bindRecordTail(tail, &Row{Kind: RecordRow, Labels: extras}, sub)
}

// bindRecordTail binds an unbound record tail to a row, through the occurs
// check: a tail that occurs in the labels it absorbs denotes an infinite type.
//
// The labels are resolved one by one rather than as one Row: most are ground
// (ApplySubstitution returns a leaf without allocating), and the whole-row
// walk allocated and cleared a visited map per binding, which dominated `check`
// on a record with hundreds of fields (#2210 review, item 6).
func (u *Unifier) bindRecordTail(tail *RowVar, row *Row, sub Substitution) (Substitution, error) {
	labels := make(map[string]Type, len(row.Labels))
	for name, typ := range row.Labels {
		labels[name] = ApplySubstitution(sub, typ)
	}
	resolved := &Row{Kind: RecordRow, Labels: labels, Tail: row.Tail}
	if u.occurs(tail.Name, resolved, RecordRow) {
		return nil, fmt.Errorf("%s: occurs check failed: record row %s occurs in %s (an infinite type)", TC_REC_003, tail.Name, resolved)
	}
	sub[tail.Name] = resolved
	u.debugSink.OnSubstitute(tail, resolved)
	return sub, nil
}

func tailBound(tail *RowVar, sub Substitution) bool {
	if tail == nil {
		return false
	}
	_, bound := sub[tail.Name]
	return bound
}

func reshape(s recordShape, sub Substitution) recordShape {
	if s.tail == nil {
		return s
	}
	out, _ := recordShapeOf(&TRecordOpen{Fields: s.labels, Row: s.tail}, sub)
	return out
}

// recordMissingFieldsError reports fields a record is required to have (by a
// field access, an update, an open pattern or an open annotation) that a
// closed record does not have.
//
// The wording `field 'x' not found in record` is what the eval harness's
// TC_REC_001 classifier matches (internal/eval_harness/errors.go).
func recordMissingFieldsError(missing, have map[string]Type) error {
	names := slices.Sorted(maps.Keys(missing))
	msg := fmt.Sprintf("%s: field '%s' not found in record %s", TC_REC_001, names[0], labelList(have))
	if len(names) > 1 {
		rest := make([]string, len(names)-1)
		for i, n := range names[1:] {
			rest[i] = "'" + n + "'"
		}
		msg += " (also missing: " + strings.Join(rest, ", ") + ")"
	}
	return fmt.Errorf("%s", msg)
}

func labelList(labels map[string]Type) string {
	return "{" + strings.Join(slices.Sorted(maps.Keys(labels)), ", ") + "}"
}

// recordWithRow rebuilds an open record after its row has been substituted. A
// row that resolved to labels is folded into the fields, so a shim whose row
// met a closed record becomes that closed record (`{score: float | ρ}` with
// ρ := {label: string} is `{label: string, score: float}`), not a shim
// carrying a row of fields that renderers and back ends do not look inside.
//
// A row that repeats one of the fields with a different type is NOT folded:
// the record keeps both (fields + row), so the next unification sees the
// duplicate and makes the two one field (unifyDuplicates, TC_REC_004). Folding
// it would silently keep one. Equal copies fold.
func recordWithRow(fields map[string]Type, row Type) Type {
	r, ok := row.(*Row)
	if !ok || rowRepeatsAField(r, fields) {
		return &TRecordOpen{Fields: fields, Row: row}
	}
	merged := make(map[string]Type, len(fields)+len(r.Labels))
	maps.Copy(merged, r.Labels)
	maps.Copy(merged, fields)
	if r.Tail == nil {
		return &TRecord{Fields: merged}
	}
	return &TRecordOpen{Fields: merged, Row: r.Tail}
}

// rowRepeatsAField reports whether a row has a label the fields already have
// with a DIFFERENT type. A repeat whose two copies are equal — the state a
// duplicate is in once unifyDuplicates has unified it and the substitution is
// applied — is folded like any other row, so no consumer (iface rendering,
// interface_hash, diagnostics) ever sees the same field twice (#2288 review).
func rowRepeatsAField(r *Row, fields map[string]Type) bool {
	for name, typ := range r.Labels {
		if f, has := fields[name]; has && !SafeEquals(f, typ) {
			return true
		}
	}
	return false
}

// rowSuffixLabels returns the labels of a concrete row that the record's
// fields do not already have: what a renderer shows after the fields, so a
// field is never printed twice (#2288 review).
func rowSuffixLabels(r *Row, fields map[string]Type) map[string]Type {
	out := make(map[string]Type, len(r.Labels))
	for name, typ := range r.Labels {
		if _, has := fields[name]; !has {
			out[name] = typ
		}
	}
	return out
}
