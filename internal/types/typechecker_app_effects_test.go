package types

import "testing"

func TestApplicationSharesCalleeRow(t *testing.T) {
	row := &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: "e", Kind: EffectRow}}
	ctx := &InferenceContext{}
	fn := &TFunc2{Params: []Type{&TFunc2{Return: TInt, EffectRow: row}}, Return: TInt, EffectRow: row}
	if got := ctx.applicationEffectRow(fn); got != row {
		t.Fatal("application detached the callback/result row")
	}
}

func TestValidationUnionPreservesTail(t *testing.T) {
	tail := &RowVar{Name: "e", Kind: EffectRow}
	open := &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: tail}
	got := UnionEffectRows(open, EmptyEffectRow())
	if got == nil || got.Tail != tail {
		t.Fatal("union discarded open tail")
	}
	if SubsumeEffectRows(open, EmptyEffectRow()) {
		t.Fatal("open requirement accepted as pure")
	}
	if FormatEffectRow(open) == "" {
		t.Fatal("tail diagnostic rendered as pure")
	}
}

func TestApplicationPublicationSnapshotsAndOwnership(t *testing.T) {
	tc := NewCoreTypeChecker()
	row := &Row{Kind: EffectRow, Labels: map[string]Type{"IO": Unit()}, Tail: &RowVar{Name: "e", Kind: EffectRow}, Params: map[string]map[string]string{"IO": {"mode": "test"}}}
	tc.publishApplication(1, []bool{true}, row, map[string]bool{"e": true})
	row.Labels["FS"] = Unit()
	got, ok := tc.ApplicationEffects(1)
	if !ok || got.CallRow.Labels["FS"] != nil {
		t.Fatal("publication aliases input")
	}
	got.CallRow.Tail.Name = "corrupted"
	got.CallRow.Params["IO"]["mode"] = "corrupted"
	got.LatentParamMask[0] = false
	tc.substituteApplicationRows(Substitution{"e": EmptyEffectRow()})
	again, _ := tc.ApplicationEffects(1)
	if again.CallRow.Tail == nil || again.CallRow.Tail.Name != "e" || !again.LatentParamMask[0] || again.CallRow.Params["IO"]["mode"] != "test" {
		t.Fatal("mutation or substitution corrupted owned generic row")
	}
	tc.publishApplication(2, []bool{false}, EmptyEffectRow(), nil)
	if pure, ok := tc.ApplicationEffects(2); !ok || pure.CallRow == nil || pure.CallRow.Tail != nil {
		t.Fatal("pure record missing")
	}
	if _, ok := tc.ApplicationEffects(3); ok {
		t.Fatal("wrong key appeared present")
	}
}

func TestDistinctValidationTails(t *testing.T) {
	row := func(name string) *Row {
		return &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: name, Kind: EffectRow}}
	}
	if _, err := CheckedUnionEffectRows(row("b"), row("a")); err == nil {
		t.Fatal("distinct tails erased")
	}
	same, err := CheckedUnionEffectRows(row("a"), row("a"))
	if err != nil || same.Tail.Name != "a" || !SubsumeEffectRows(same, row("a")) {
		t.Fatal("same tail rejected")
	}
}

func TestApplicationRowZonksFullChain(t *testing.T) {
	tc := NewCoreTypeChecker()
	row := func(name string) *Row {
		return &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: name, Kind: EffectRow}}
	}
	tc.publishApplication(7, nil, row("a"), nil)
	tc.substituteApplicationRows(Substitution{"a": row("b"), "b": &Row{Kind: EffectRow, Labels: map[string]Type{"IO": Unit()}}})
	record, _ := tc.ApplicationEffects(7)
	if record.CallRow.Tail != nil || record.CallRow.Labels["IO"] == nil {
		t.Fatal("partially solved row escaped publication")
	}
	tc.applyApplicationSubstitution(nil)
	tc.substituteApplicationRows(Substitution{"a": EmptyEffectRow()})
	sealed, _ := tc.ApplicationEffects(7)
	if sealed.CallRow.Labels["IO"] == nil {
		t.Fatal("later declaration changed finalized publication")
	}
}
