package types

import "testing"

// A let-bound reader's scheme quantifies the record row of its undecided
// access (`\t. t.x`: the row occurs only there, never in the scheme's type).
// Instantiating it must give a fresh RECORD row (#2239, with #2210's fresh row
// per access). Record unification identifies a row variable by name whatever
// its kind, so an effect-kinded row here would type-check today; it is still a
// kind error waiting for the first kind-strict consumer, and nothing else
// observes it — hence this direct check.
func TestInstantiate_AccessOnlyRowIsARecordRow(t *testing.T) {
	recv := &TVar2{Name: "a", Kind: Star}
	field := &TVar2{Name: "b", Kind: Star}
	row := &RowVar{Name: "p", Kind: RecordRow}
	s := &Scheme{
		TypeVars:      []string{"a", "b"},
		RowVars:       []string{"p"},
		Type:          &TFunc2{Params: []Type{recv}, Return: field, EffectRow: &Row{Kind: EffectRow, Labels: map[string]Type{}}},
		fieldAccesses: []*fieldAccess{{recv: recv, field: "x", fieldType: field, row: row}},
	}
	ctx := NewInferenceContext()
	_, _, accesses := s.instantiate(ctx.freshType, nil)
	if len(accesses) != 1 {
		t.Fatalf("want the scheme's one access instantiated, got %d", len(accesses))
	}
	got, ok := accesses[0].row.(*RowVar)
	if !ok || got.Kind != RecordRow || got.Name == "p" {
		t.Fatalf("access row instantiated as %#v, want a fresh record row", accesses[0].row)
	}
	if r, ok := accesses[0].recv.(*TVar2); !ok || r.Name == "a" {
		t.Fatalf("receiver not instantiated: %v", accesses[0].recv)
	}
}
