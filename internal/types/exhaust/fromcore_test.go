package exhaust

import (
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

func TestFromCore(t *testing.T) {
	var rest core.CorePattern = &core.VarPattern{Name: "r"}
	var nilTail core.CorePattern = &core.ListPattern{}
	cases := []struct {
		name string
		in   core.CorePattern
		want Pattern
	}{
		{"wildcard", &core.WildcardPattern{}, Wild{}},
		{"variable", &core.VarPattern{Name: "x"}, Wild{}},
		{"literal", &core.LitPattern{Value: int64(3)}, Lit{Value: int64(3)}},
		{"ctor nested", &core.ConstructorPattern{Name: "Some", TypeName: "Option", Args: []core.CorePattern{
			&core.ConstructorPattern{Name: "Err", TypeName: "Result", Args: []core.CorePattern{&core.WildcardPattern{}}}}},
			Ctor{TypeName: "Option", Name: "Some", Args: []Pattern{Ctor{TypeName: "Result", Name: "Err", Args: []Pattern{Wild{}}}}}},
		{"tuple", &core.TuplePattern{Elements: []core.CorePattern{&core.LitPattern{Value: true}, &core.VarPattern{Name: "y"}}},
			Tuple{Elems: []Pattern{Lit{Value: true}, Wild{}}}},
		{"closed list", &core.ListPattern{Elements: []core.CorePattern{&core.VarPattern{Name: "a"}}},
			List{Elems: []Pattern{Wild{}}}},
		{"spread", &core.ListPattern{Elements: []core.CorePattern{&core.VarPattern{Name: "a"}}, Tail: &rest},
			List{Elems: []Pattern{Wild{}}, Rest: Wild{}}},
		{"cons onto []", &core.ListPattern{Elements: []core.CorePattern{&core.VarPattern{Name: "a"}}, Tail: &nilTail},
			List{Elems: []Pattern{Wild{}}, Rest: List{Elems: []Pattern{}}}},
		{"record", &core.RecordPattern{Fields: map[string]core.CorePattern{"a": &core.LitPattern{Value: "s"}}, Rest: true},
			Record{Fields: map[string]Pattern{"a": Lit{Value: "s"}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := FromCore(tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("FromCore = %#v\nwant %#v", got, tc.want)
			}
		})
	}
	// Errors propagate from any depth; a nil pattern is not a pattern.
	bad := []core.CorePattern{
		nil,
		&core.ConstructorPattern{Name: "C", Args: []core.CorePattern{nil}},
		&core.TuplePattern{Elements: []core.CorePattern{nil, nil}},
		&core.ListPattern{Elements: []core.CorePattern{nil}},
		&core.ListPattern{Tail: new(core.CorePattern)},
		&core.RecordPattern{Fields: map[string]core.CorePattern{"a": nil}},
	}
	for i, p := range bad {
		if _, err := FromCore(p); err == nil {
			t.Errorf("bad[%d]: want an error", i)
		}
	}
}

// A [] tail behaves like a closed list for coverage: `[a, ...[]]` covers
// exactly the one-element lists.
func TestFromCoreCoverage(t *testing.T) {
	var nilTail core.CorePattern = &core.ListPattern{}
	p, err := FromCore(&core.ListPattern{Elements: []core.CorePattern{&core.VarPattern{Name: "a"}}, Tail: &nilTail})
	if err != nil {
		t.Fatal(err)
	}
	res, err := Check([]Arm{{Pattern: p}, {Pattern: List{}}}, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := Describe(res.Missing[0]); got != "[_, _, ..._]" {
		t.Errorf("missing %q, want [_, _, ..._]", got)
	}
}
