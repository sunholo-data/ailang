package iface

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// #1374: formatTypeCanonical fell back to "<%T>" for every type without a case,
// so `ailang iface --json` printed "<*types.TRecord2>" (std/ai's AIError inside
// Result[...]) and likewise leaked TTuple, TArray, TMap, TLabelled and
// TRecordOpen across the stdlib.

func TestFormatTypeCanonical_Record2Closed(t *testing.T) {
	rec := &types.TRecord2{Row: &types.Row{Labels: map[string]types.Type{
		"retryable": &types.TCon{Name: "bool"},
		"code":      &types.TCon{Name: "string"},
		"message":   &types.TCon{Name: "string"},
	}}}
	res := &types.TApp{
		Constructor: &types.TCon{Name: "Result"},
		Args:        []types.Type{&types.TCon{Name: "string"}, rec},
	}
	got := formatTypeCanonical(res, identCanon())
	want := "Result[string,{code: string, message: string, retryable: bool}]"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatTypeCanonical_Record2OpenTail(t *testing.T) {
	rec := &types.TRecord2{Row: &types.Row{
		Labels: map[string]types.Type{"id": &types.TCon{Name: "string"}},
		Tail:   &types.RowVar{Name: "r"},
	}}
	if got := formatTypeCanonical(rec, identCanon()); got != "{id: string, ...r}" {
		t.Errorf("got %q, want {id: string, ...r}", got)
	}
}

func TestFormatTypeCanonical_Record2RendersLikeRecord(t *testing.T) {
	fields := map[string]types.Type{"x": &types.TCon{Name: "int"}, "y": &types.TVar2{Name: "a"}}
	r1 := formatTypeCanonical(&types.TRecord{Fields: fields}, identCanon())
	r2 := formatTypeCanonical(&types.TRecord2{Row: &types.Row{Labels: fields}}, identCanon())
	if r1 != r2 {
		t.Errorf("TRecord2 %q must render like TRecord %q", r2, r1)
	}
	if got := formatTypeCanonical(&types.TRecord2{}, identCanon()); got != "{}" {
		t.Errorf("empty TRecord2: got %q, want {}", got)
	}
}

func TestFormatTypeCanonical_NoGoTypeLeaks(t *testing.T) {
	intT := &types.TCon{Name: "int"}
	strT := &types.TCon{Name: "string"}
	cases := []struct {
		name string
		typ  types.Type
		want string
	}{
		{"tuple", &types.TTuple{Elements: []types.Type{intT, strT}}, "(int,string)"},
		{"unit", &types.TTuple{}, "()"},
		{"array", &types.TArray{Element: intT}, "Array[int]"},
		{"map", &types.TMap{Key: strT, Value: intT}, "Map[string,int]"},
		{"labelled", types.WithLabel(strT, types.LabelConst("secret")), "string<secret>"},
		{"labelled-bottom", &types.TLabelled{Inner: strT, L: types.LabelBottom()}, "string"},
		{"record-open", &types.TRecordOpen{
			Fields: map[string]types.Type{"ver": intT},
			Row:    &types.TVar2{Name: "a"},
		}, "{ver: int, ...a}"},
		{"tvar-v1", &types.TVar{Name: "b"}, "b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatTypeCanonical(tc.typ, identCanon())
			if strings.Contains(got, "<*") {
				t.Fatalf("leaked a Go internal type: %q", got)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
