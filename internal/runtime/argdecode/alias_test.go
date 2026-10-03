package argdecode

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Named record types as entry parameters (stapledons_godot, 2026-10-01):
// `func main(a: Args)` failed with "unsupported type constructor: Args".

func aliasTable(m map[string]types.Type) AliasLookup {
	return func(name string) (types.Type, bool) {
		t, ok := m[name]
		return t, ok
	}
}

func TestDecodeJSONWithAliases(t *testing.T) {
	T := types.NewBuilder()
	vec := &types.TRecord{Fields: map[string]types.Type{"x": T.Float(), "y": T.Float()}}
	args := T.Record(
		types.Field("name", T.String()),
		types.Field("origin", T.Con("Vec")),
		types.Field("pts", T.List(T.Con("Vec"))),
	)
	aliases := aliasTable(map[string]types.Type{
		"Vec":   vec,
		"Args":  args,
		"Alias": T.Con("Args"), // chain: Alias -> Args -> record
		"Loop":  T.Con("Loop"), // cycle
	})
	const js = `{"name":"s","origin":{"x":0.5,"y":1.5},"pts":[{"x":1.0,"y":2.0}]}`

	for _, name := range []string{"Args", "Alias"} {
		v, err := DecodeJSONWithAliases(js, T.Con(name), aliases)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		rec := v.(*eval.RecordValue)
		origin := rec.Fields["origin"].(*eval.RecordValue)
		if x := origin.Fields["x"].(*eval.FloatValue).Value; x != 0.5 {
			t.Errorf("%s: origin.x = %v, want 0.5", name, x)
		}
		if n := len(rec.Fields["pts"].(*eval.ListValue).Elements); n != 1 {
			t.Errorf("%s: len(pts) = %d, want 1", name, n)
		}
	}

	if _, err := DecodeJSONWithAliases(`1`, T.Con("Loop"), aliases); err == nil {
		t.Error("cyclic alias decoded; want an error")
	}
	_, err := DecodeJSONWithAliases(`{}`, T.Con("Unknown"), aliases)
	if err == nil || !strings.Contains(err.Error(), "unsupported type constructor: Unknown") {
		t.Errorf("unknown name: err = %v", err)
	}
	if _, err := DecodeJSON(js, T.Con("Args")); err == nil {
		t.Error("DecodeJSON without aliases decoded a named type; the nil-lookup path must stay strict")
	}
}
