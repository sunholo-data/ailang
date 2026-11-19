package eval

import (
	"math"
	"strings"
	"testing"
	"unicode/utf8"
)

// The show contract, in this package (M-V1-SHOW-TOTAL). The two-engine
// version of the table is internal/vm/show_engines_test.go; this one pins
// the evaluator renderer and the bounded/preview writers on their own.

func TestShowSpecRowsEvaluator(t *testing.T) {
	str := func(s string) Value { return &StringValue{Value: s} }
	i := func(n int) Value { return &IntValue{Value: n} }
	f := func(x float64) Value { return &FloatValue{Value: x} }
	m := &MapValue{Entries: map[string]*MapEntry{}}
	for _, kv := range [][2]Value{{i(9), str("a")}, {i(10), str("b")}, {i(2), str("c")}} {
		k, err := MapKey(kv[0])
		if err != nil {
			t.Fatal(err)
		}
		m.Entries[k] = &MapEntry{Key: kv[0], Value: kv[1]}
	}
	deep := Value(&TaggedValue{CtorName: "Z"})
	for n := 0; n < 3; n++ {
		deep = &TaggedValue{CtorName: "S", Fields: []Value{deep}}
	}
	cases := []struct {
		v    Value
		want string
	}{
		{i(-7), "-7"},
		{f(math.Copysign(0, -1)), "-0.0"},
		{f(2), "2.0"},
		{f(1e21), "1000000000000000000000.0"},
		{f(math.NaN()), "NaN"},
		{f(math.Inf(1)), "Inf"},
		{f(math.Inf(-1)), "-Inf"},
		{&BoolValue{Value: false}, "false"},
		{&UnitValue{}, "()"},
		{str("top \"bare\""), "top \"bare\""},
		{&ListValue{Elements: []Value{str("a"), str("b, c"), str("")}}, `["a", "b, c", ""]`},
		{NewArray([]Value{f(1.5), f(-2)}), "#[1.5, -2.0]"},
		{NewArray(nil), "#[]"},
		{&TupleValue{Elements: []Value{i(1), str("x")}}, `(1, "x")`},
		{&RecordValue{Fields: map[string]Value{"b": str("two"), "a": i(1)}}, `{a: 1, b: "two"}`},
		{&RecordValue{Fields: map[string]Value{}}, "{}"},
		{m, `Map{10: "b", 2: "c", 9: "a"}`},
		{&MapValue{Entries: map[string]*MapEntry{}}, "Map{}"},
		{&TaggedValue{CtorName: "Named", Fields: []Value{str("n\"m"), &TaggedValue{CtorName: "Dot"}}}, `Named("n\"m", Dot)`},
		{deep, "S(S(S(Z)))"},
		{&BytesValue{Value: []byte(strings.Repeat("\x01", 40))}, "<bytes:" + strings.Repeat("01", 40) + ">"},
		{&FunctionValue{}, "<function>"},
		{&BuiltinFunction{Name: "x"}, "<function>"},
		{&ConstructorClosure{CtorName: "Some", Arity: 1}, "<function>"},
		{&ErrorValue{Message: "boom"}, "Error: boom"},
		{&IndirectValue{Cell: &RefCell{Val: &ListValue{Elements: []Value{str("x")}}, Init: true}}, `["x"]`},
		{&IndirectValue{Cell: &RefCell{}}, "<uninitialized>"},
		{&tailCall{}, (&tailCall{}).String()},
		{&ListValue{Elements: []Value{str("e\U00000301"), str("\n\U00000303")}}, `["e\u{301}", "\n\u{303}"]`},
	}
	for _, c := range cases {
		if got := Show(c.v); got != c.want {
			t.Errorf("Show(%T) = %q, want %q", c.v, got, c.want)
		}
		if got := RenderedLen(c.v); got != len(c.want) {
			t.Errorf("RenderedLen(%T) = %d, want %d", c.v, got, len(c.want))
		}
	}
}

// A value type the renderer does not know is spelled "<unknown>", never a
// panic or an empty string.
type alienValue struct{}

func (alienValue) Type() string   { return "alien" }
func (alienValue) String() string { return "alien" }

func TestShowUnknownValue(t *testing.T) {
	if got := Show(&ListValue{Elements: []Value{alienValue{}}}); got != "[<unknown>]" {
		t.Fatalf("got %q", got)
	}
}

func TestShowPreviewEvaluator(t *testing.T) {
	list := &ListValue{Elements: []Value{&IntValue{Value: 1}, &IntValue{Value: 2}, &IntValue{Value: 3}}}
	cases := []struct {
		v    Value
		max  int
		want string
	}{
		{list, 9, "[1, 2, 3]"},
		{list, 5, "[1, 2…(+4 chars)"},
		{list, 0, "…(+9 chars)"},
		{list, -1, "…(+9 chars)"},
		{&StringValue{Value: "abc"}, 5, `"abc"`},
		{&StringValue{Value: "abc"}, 4, `"abc…(+1 chars)`},
		{&StringValue{Value: "ééé"}, 3, `"éé…(+2 chars)`},
		{&BytesValue{Value: []byte("hi")}, 7, "<bytes:…(+5 chars)"},
	}
	for _, c := range cases {
		if got := ShowPreview(c.v, c.max); got != c.want {
			t.Errorf("ShowPreview(%s, %d) = %q, want %q", Show(c.v), c.max, got, c.want)
		}
	}
}

// The byte budget (traces) never splits a character and, once it has cut,
// writes nothing more — not even a later short segment that would fit.
func TestShowBoundedCutsOnRuneBoundary(t *testing.T) {
	v := &ListValue{Elements: []Value{&StringValue{Value: "ab😀"}, &IntValue{Value: 1}}}
	full := Show(v) // ["ab😀", 1]
	for budget := 1; budget < len(full); budget++ {
		got := ShowBounded(v, budget)
		if !utf8.ValidString(got) {
			t.Fatalf("@%d: invalid UTF-8 %q", budget, got)
		}
		prefix := got[:strings.Index(got, "…(+")]
		if !strings.HasPrefix(full, prefix) || len(prefix) > budget {
			t.Fatalf("@%d: %q is not a prefix of %q within budget", budget, prefix, full)
		}
	}
}

func TestRedactedValueSpelling(t *testing.T) {
	var v Value = redactedValue{}
	if v.Type() != "redacted" || v.String() != RedactedMarker || Show(v) != "<unknown>" || ShowTraceBounded(v, 0) != RedactedMarker {
		t.Fatalf("redacted value: Type %q String %q Show %q trace %q", v.Type(), v.String(), Show(v), ShowTraceBounded(v, 0))
	}
}
