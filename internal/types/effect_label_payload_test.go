package types

import "testing"

func TestEffectLabelPayload(t *testing.T) {
	fn := NewBuilder().Func(TInt).Returns(TInt).Effects("IO").(*TFunc2)
	if fn.EffectRow.Labels["IO"].String() != Unit().String() {
		t.Fatal("effect payload must be Unit")
	}
}

func TestRowPayloadKinds(t *testing.T) {
	for _, tt := range []struct {
		name   string
		kind   Kind
		reject bool
	}{
		{"effect", EffectRow, false}, {"record", RecordRow, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRowUnifier().UnifyRows(
				&Row{Kind: tt.kind, Labels: map[string]Type{"IO": &TCon{Name: "IO"}}},
				&Row{Kind: tt.kind, Labels: map[string]Type{"IO": Unit()}}, Substitution{})
			if (err != nil) != tt.reject {
				t.Fatalf("unexpected result: %v", err)
			}
		})
	}
}
