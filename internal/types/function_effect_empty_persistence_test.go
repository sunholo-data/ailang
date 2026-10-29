package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExplicitEmptyTypePersistence(t *testing.T) {
	fn := &TFunc2{Params: []Type{&TVar2{Name: "a", Kind: Star}}, Return: &TVar2{Name: "a", Kind: Star}, EffectRow: EmptyEffectRow(), ConcreteEffectContract: true}
	if !strings.Contains(fn.String(), "! {}") {
		t.Fatalf("internal contract string missing: %s", fn)
	}
	sub := fn.Substitute(map[string]Type{"a": TInt}).(*TFunc2)
	raw, err := json.Marshal(sub)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := UnmarshalType(raw)
	if err != nil {
		t.Fatal(err)
	}
	got := decoded.(*TFunc2)
	if got.EffectRow.Tail != nil || !got.ConcreteEffectContract || got.ImplicitEffectContract || !got.Params[0].Equals(TInt) {
		t.Fatalf("contract changed through substitution/JSON: %s", raw)
	}
}
