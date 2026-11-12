package types

import (
	"strings"
	"testing"
)

// Unit tests for the one record unification (#2210, #2221).

func rv(name string) *RowVar { return &RowVar{Name: name, Kind: RecordRow} }

func tv(name string) *TVar2 { return &TVar2{Name: name, Kind: Star} }

func closedRec(fields map[string]Type) *TRecord { return &TRecord{Fields: fields} }

// The field-access shim meets a closed record: its row is bound to the
// record's remaining fields, closed (the #2210 placeholder left it unbound).
func TestUnifyRecords_ShimMeetsClosedRecordBindsRow(t *testing.T) {
	u := NewUnifier()
	shim := &TRecordOpen{Fields: map[string]Type{"score": tv("a")}, Row: rv("ρ1")}
	rec := closedRec(map[string]Type{"label": TString, "score": TFloat})
	sub, err := u.Unify(shim, rec, Substitution{})
	if err != nil {
		t.Fatal(err)
	}
	got := ApplySubstitution(sub, shim)
	want := closedRec(map[string]Type{"label": TString, "score": TFloat})
	if !got.Equals(want) {
		t.Fatalf("shim resolved to %s, want %s", got, want)
	}
	// And now a second shim for a field the record lacks is rejected.
	bogus := &TRecordOpen{Fields: map[string]Type{"bogus": tv("b")}, Row: rv("ρ2")}
	_, err = u.Unify(got, bogus, sub)
	if err == nil || !strings.Contains(err.Error(), TC_REC_001) || !strings.Contains(err.Error(), "field 'bogus' not found") {
		t.Fatalf("missing field accepted or mis-coded: %v", err)
	}
}

// Two shims (an update and an access on different fields) share one fresh
// remainder, so neither loses the other's field (#2221 part 2).
func TestUnifyRecords_TwoShimsShareRemainder(t *testing.T) {
	u := NewUnifier()
	a := &TRecordOpen{Fields: map[string]Type{"x": TInt}, Row: rv("ρ1")}
	b := &TRecordOpen{Fields: map[string]Type{"y": TString}, Row: rv("ρ2")}
	sub, err := u.Unify(a, b, Substitution{})
	if err != nil {
		t.Fatal(err)
	}
	ra := ApplySubstitution(sub, a).(*TRecordOpen)
	rb := ApplySubstitution(sub, b).(*TRecordOpen)
	for _, r := range []*TRecordOpen{ra, rb} {
		if len(r.Fields) != 2 || r.Fields["x"] == nil || r.Fields["y"] == nil {
			t.Fatalf("expected {x, y | ρ}, got %s", r)
		}
	}
	if ra.Row.(*RowVar).Name != rb.Row.(*RowVar).Name {
		t.Fatalf("remainders differ: %s vs %s", ra, rb)
	}
	// Closing it with a record that has a string x fails.
	if _, err := u.Unify(ra, closedRec(map[string]Type{"x": TString, "y": TString}), sub); err == nil {
		t.Fatal("x: int unified with x: string")
	}
}

// Closed records: same fields required, with the long-standing mismatch text.
func TestUnifyRecords_ClosedPairs(t *testing.T) {
	u := NewUnifier()
	if _, err := u.Unify(closedRec(map[string]Type{"a": TInt}), closedRec(map[string]Type{"a": TInt}), Substitution{}); err != nil {
		t.Fatal(err)
	}
	_, err := u.Unify(closedRec(map[string]Type{"a": TInt}), closedRec(map[string]Type{"a": TInt, "b": TInt}), Substitution{})
	if err == nil || !strings.Contains(err.Error(), "record field mismatch") {
		t.Fatalf("closed width mismatch: %v", err)
	}
}

// The same row on both sides with different extra fields has no finite
// solution.
func TestUnifyRecords_SameRowDifferentFields(t *testing.T) {
	u := NewUnifier()
	a := &TRecordOpen{Fields: map[string]Type{"x": TInt}, Row: rv("ρ")}
	b := &TRecordOpen{Fields: map[string]Type{"y": TInt}, Row: rv("ρ")}
	_, err := u.Unify(a, b, Substitution{})
	if err == nil || !strings.Contains(err.Error(), TC_REC_003) {
		t.Fatalf("expected %s, got %v", TC_REC_003, err)
	}
}

// #2221 part 1: a variable occurs in every record form, so binding it to a
// record that contains it fails instead of building a cyclic type.
func TestOccurs_EveryRecordFormIsWalked(t *testing.T) {
	u := NewUnifier()
	a := tv("a")
	forms := map[string]Type{
		"TRecord field":     closedRec(map[string]Type{"self": a}),
		"TRecord row":       &TRecord{Fields: map[string]Type{}, Row: &Row{Kind: RecordRow, Labels: map[string]Type{"self": a}}},
		"TRecordOpen field": &TRecordOpen{Fields: map[string]Type{"self": a}, Row: rv("ρ")},
		"TRecord2":          &TRecord2{Row: &Row{Kind: RecordRow, Labels: map[string]Type{"self": a}}},
		"TLabelled":         &TLabelled{Inner: closedRec(map[string]Type{"self": a})},
	}
	for name, form := range forms {
		if !u.occurs("a", form, Star) {
			t.Errorf("%s: a does not occur in %s", name, form)
		}
	}
	_, err := u.Unify(a, closedRec(map[string]Type{"self": a}), Substitution{})
	if err == nil || !strings.Contains(err.Error(), TC_OCCURS) {
		t.Fatalf("α := {self: α} accepted: %v", err)
	}
	// A record row that would absorb a field mentioning itself.
	shim := &TRecordOpen{Fields: map[string]Type{"x": TInt}, Row: rv("ρ")}
	rec := closedRec(map[string]Type{"x": TInt, "y": &TRecordOpen{Fields: map[string]Type{}, Row: rv("ρ")}})
	if _, err := u.Unify(shim, rec, Substitution{}); err == nil || !strings.Contains(err.Error(), TC_REC_003) {
		t.Fatalf("ρ := {y: {| ρ}} accepted: %v", err)
	}
}

// A record row bound through a chain resolves every link, with the links'
// field types substituted.
func TestApplySubstitution_RecordRowChain(t *testing.T) {
	sub := Substitution{
		"ρ1": &Row{Kind: RecordRow, Labels: map[string]Type{"b": tv("β")}, Tail: rv("ρ2")},
		"ρ2": &Row{Kind: RecordRow, Labels: map[string]Type{"c": TInt}, Tail: rv("ρ3")},
		"ρ3": &Row{Kind: RecordRow, Labels: map[string]Type{"d": TBool}},
		"β":  TString,
	}
	got := ApplySubstitution(sub, &TRecordOpen{Fields: map[string]Type{"a": TInt}, Row: rv("ρ1")})
	want := closedRec(map[string]Type{"a": TInt, "b": TString, "c": TInt, "d": TBool})
	if !got.Equals(want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

// Every row form a record can carry flattens to the same shape.
func TestRecordShapeOf_RowForms(t *testing.T) {
	sub := Substitution{"ρb": &Row{Kind: RecordRow, Labels: map[string]Type{"b": TInt}, Tail: rv("ρc")}}
	cases := map[string]struct {
		t      Type
		labels int
		tail   string
		ok     bool
	}{
		"closed TRecord":       {closedRec(map[string]Type{"a": TInt}), 1, "", true},
		"TRecord row chain":    {&TRecord{Fields: map[string]Type{"a": TInt}, Row: rv("ρb")}, 2, "ρc", true},
		"TVar2 row tail":       {&TRecordOpen{Fields: map[string]Type{"a": TInt}, Row: &TVar2{Name: "ρb", Kind: RecordRow}}, 2, "ρc", true},
		"TVar row tail":        {&TRecordOpen{Fields: map[string]Type{"a": TInt}, Row: &TVar{Name: "ρz"}}, 1, "ρz", true},
		"TRecord2":             {&TRecord2{Row: &Row{Kind: RecordRow, Labels: map[string]Type{"a": TInt}, Tail: rv("ρq")}}, 1, "ρq", true},
		"empty TRecord2":       {&TRecord2{}, 0, "", true},
		"star TVar2 row (bad)": {&TRecordOpen{Fields: map[string]Type{}, Row: tv("α")}, 0, "", false},
		"non-row row (bad)":    {&TRecordOpen{Fields: map[string]Type{}, Row: TInt}, 0, "", false},
		"not a record (bad)":   {TInt, 0, "", false},
	}
	for name, c := range cases {
		s, ok := recordShapeOf(c.t, sub)
		if ok != c.ok {
			t.Errorf("%s: ok = %v, want %v", name, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		tail := ""
		if s.tail != nil {
			tail = s.tail.Name
		}
		if len(s.labels) != c.labels || tail != c.tail {
			t.Errorf("%s: got %d labels tail %q, want %d tail %q", name, len(s.labels), tail, c.labels, c.tail)
		}
	}
	// A cyclic substitution stops at the repeated variable.
	cyc := Substitution{"ρ1": rv("ρ2"), "ρ2": rv("ρ1")}
	if s, ok := recordShapeOf(&TRecordOpen{Fields: map[string]Type{}, Row: rv("ρ1")}, cyc); !ok || s.tail == nil {
		t.Fatalf("cyclic row chase: ok=%v tail=%v", ok, s.tail)
	}
	// A malformed record row is an error, not a silent pass.
	if _, err := NewUnifier().Unify(&TRecordOpen{Fields: map[string]Type{}, Row: TInt}, closedRec(map[string]Type{}), Substitution{}); err == nil {
		t.Fatal("malformed record row unified")
	}
}

// A shared field whose unification binds a tail is redone on the resolved
// shapes: the inner shim's row ρ1 is also the outer record's row.
func TestUnifyRecords_SharedFieldBindsTail(t *testing.T) {
	u := NewUnifier()
	a := &TRecordOpen{Fields: map[string]Type{"f": &TRecordOpen{Fields: map[string]Type{}, Row: rv("ρ1")}}, Row: rv("ρ1")}
	b := &TRecordOpen{Fields: map[string]Type{"f": closedRec(map[string]Type{"z": TInt})}, Row: rv("ρ2")}
	sub, err := u.Unify(a, b, Substitution{})
	if err != nil {
		t.Fatal(err)
	}
	got, ok := ApplySubstitution(sub, a).(*TRecord)
	if !ok || got.Fields["z"] == nil || got.Fields["f"] == nil {
		t.Fatalf("expected the outer record closed over {f, z}, got %s", ApplySubstitution(sub, a))
	}
}

// Several missing fields are named, the first in the classifier's wording.
func TestRecordMissingFieldsError_Several(t *testing.T) {
	err := recordMissingFieldsError(map[string]Type{"b": TInt, "a": TInt, "c": TInt}, map[string]Type{"id": TString})
	want := "TC_REC_001: field 'a' not found in record {id} (also missing: 'b', 'c')"
	if err.Error() != want {
		t.Fatalf("got %q, want %q", err.Error(), want)
	}
}

// Row-variable links (RowVar, TVar2) in a record row chain resolve; an
// unexpected binding stops the chase.
func TestApplySubstitution_RecordRowChainLinks(t *testing.T) {
	sub := Substitution{
		"ρ1": rv("ρ2"),
		"ρ2": &TVar2{Name: "ρ3", Kind: RecordRow},
		"ρ3": &Row{Kind: RecordRow, Labels: map[string]Type{"x": TInt}, Tail: rv("ρ4")},
		"ρ4": TInt, // malformed: stops the chase at ρ4
	}
	got := ApplySubstitution(sub, &Row{Kind: RecordRow, Labels: map[string]Type{}, Tail: rv("ρ1")}).(*Row)
	if got.Labels["x"] == nil || got.Tail == nil || got.Tail.Name != "ρ4" {
		t.Fatalf("got %s", got)
	}
	// An open record whose row is still a variable stays a shim.
	if r := recordWithRow(map[string]Type{"a": TInt}, rv("ρ9")); r.(*TRecordOpen).Row.(*RowVar).Name != "ρ9" {
		t.Fatalf("got %s", r)
	}
	// A label two links of one chain both carry keeps the EARLIER link's type,
	// as recordShape.extend keeps the record's own field (review mutant N8:
	// "later link wins"). Only an un-unified duplicate reaches this.
	dupChain := Substitution{
		"ρa": &Row{Kind: RecordRow, Labels: map[string]Type{"x": TInt}, Tail: rv("ρb")},
		"ρb": &Row{Kind: RecordRow, Labels: map[string]Type{"x": TString}},
	}
	chained := ApplySubstitution(dupChain, &Row{Kind: RecordRow, Labels: map[string]Type{}, Tail: rv("ρa")}).(*Row)
	if chained.Labels["x"] != TInt {
		t.Fatalf("later chain link won: x = %s", chained.Labels["x"])
	}
	// TRecordOpen.Substitute folds a bound row too.
	folded := (&TRecordOpen{Fields: map[string]Type{"a": TInt}, Row: rv("ρ")}).Substitute(map[string]Type{"ρ": &Row{Kind: RecordRow, Labels: map[string]Type{"b": TBool}}})
	if want := closedRec(map[string]Type{"a": TInt, "b": TBool}); !folded.Equals(want) {
		t.Fatalf("Substitute: got %s, want %s", folded, want)
	}
}

// Every record representation dispatches to the one record unification, a
// type variable on the right swaps, and a non-record is a mismatch.
func TestUnifyRecords_Dispatch(t *testing.T) {
	rec := closedRec(map[string]Type{"a": TInt})
	open := &TRecordOpen{Fields: map[string]Type{"a": TInt}, Row: rv("ρ")}
	r2 := &TRecord2{Row: &Row{Kind: RecordRow, Labels: map[string]Type{"a": TInt}}}
	for _, l := range []Type{rec, open, r2} {
		for _, r := range []Type{rec, open, r2, tv("β")} {
			if _, err := NewUnifier().Unify(l, r, Substitution{}); err != nil {
				t.Errorf("%s ~ %s: %v", l, r, err)
			}
		}
		if _, err := NewUnifier().Unify(l, &TList{Element: TInt}, Substitution{}); err == nil {
			t.Errorf("%s ~ [int] unified", l)
		}
		if _, err := NewUnifier().Unify(l, &TCon{Name: "Nope"}, Substitution{}); err == nil {
			t.Errorf("%s ~ Nope unified", l)
		}
	}
	if got := RecordRowCodes(); len(got) != 4 {
		t.Fatalf("RecordRowCodes = %v", got)
	}
}

// A label a record has in its fields and again in its bound row is one field:
// the two types unify (TC_REC_004 when they differ), whichever copy a reader
// would have picked. Substitution keeps such a record unfolded so the next
// unification sees both copies.
func TestUnifyRecords_DuplicateLabelIsOneField(t *testing.T) {
	sub := Substitution{"ρ": &Row{Kind: RecordRow, Labels: map[string]Type{"x": TString}}}
	q := &TRecordOpen{Fields: map[string]Type{"x": TInt}, Row: rv("ρ")}
	resolved := ApplySubstitution(sub, q)
	if _, folded := resolved.(*TRecord); folded {
		t.Fatalf("a row repeating a field was folded away: %s", resolved)
	}
	s, ok := recordShapeOf(resolved, sub)
	if !ok || len(s.dups) != 1 || s.dups[0].name != "x" {
		t.Fatalf("duplicate not recorded: %+v", s)
	}
	_, err := NewUnifier().Unify(q, closedRec(map[string]Type{"x": TInt}), sub)
	if err == nil || !strings.Contains(err.Error(), TC_REC_004) {
		t.Fatalf("x: int and x: string accepted as one field: %v", err)
	}
	// The unfolded record never prints the field twice.
	if got := resolved.String(); strings.Count(got, "x:") != 1 {
		t.Fatalf("duplicate printed twice: %s", got)
	}
	// Agreeing copies are one field and unify, and substitution folds them.
	agree := Substitution{"ρ": &Row{Kind: RecordRow, Labels: map[string]Type{"x": TInt}}}
	if folded, ok := ApplySubstitution(agree, q).(*TRecord); !ok || len(folded.Fields) != 1 {
		t.Fatalf("agreeing duplicate not folded: %s", ApplySubstitution(agree, q))
	}
	if _, err := NewUnifier().Unify(q, closedRec(map[string]Type{"x": TInt}), agree); err != nil {
		t.Fatalf("agreeing duplicate rejected: %v", err)
	}
	// A record literal's own row copy (same nodes) is not a duplicate.
	fields := map[string]Type{"a": TInt}
	lit := &TRecord{Fields: fields, Row: &Row{Kind: RecordRow, Labels: fields}}
	if s, _ := recordShapeOf(lit, Substitution{}); len(s.dups) != 0 {
		t.Fatalf("literal row copy counted as duplicate: %+v", s.dups)
	}
	if got := lit.String(); strings.Contains(got, "...") {
		t.Fatalf("literal renders its row copy: %s", got)
	}
}

// Two closed records of the same size with different labels name the
// difference, not "expected 2 fields, got 2".
func TestRecordFieldMismatchError_SameSize(t *testing.T) {
	msg := recordFieldMismatchError(map[string]Type{"a": TInt, "b": TInt}, map[string]Type{"a": TInt, "c": TInt}).Error()
	for _, want := range []string{"both records have 2 fields, but not the same ones", "extra fields:    c", "missing fields:  b"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message lacks %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "expected 2 fields, got 2") {
		t.Errorf("message still says expected 2, got 2:\n%s", msg)
	}
}

// The occurs check also sees an old-style TVar.
func TestOccurs_TVar(t *testing.T) {
	if !NewUnifier().occurs("a", closedRec(map[string]Type{"s": &TVar{Name: "a"}}), Star) {
		t.Fatal("a does not occur in {s: a}")
	}
}
