package bytecode

import (
	"math"
	"testing"
)

// stubProto is a test-only FuncPrototypeRef so value tests don't depend on
// image.go.
type stubProto struct {
	name   string
	regs   uint8
	params uint8
}

func (s *stubProto) ProtoName() string    { return s.name }
func (s *stubProto) NumRegisters() uint8  { return s.regs }
func (s *stubProto) NumParameters() uint8 { return s.params }

func TestConstructors_Tags(t *testing.T) {
	cases := []struct {
		v   Value
		tag ValueTag
	}{
		{NewInt(42), TagInt},
		{NewFloat(3.14), TagFloat},
		{NewBool(true), TagBool},
		{Unit(), TagUnit},
		{NewString("hi"), TagString},
		{NewList([]Value{NewInt(1), NewInt(2)}), TagList},
		{NewTuple([]Value{NewInt(1), NewBool(false)}), TagTuple},
		{NewRecord([]RecordField{{"x", NewInt(1)}}), TagRecord},
		{NewClosure(&stubProto{name: "f"}, nil), TagClosure},
		{NewADT(0, []Value{NewInt(1)}), TagADT},
	}
	for _, tc := range cases {
		if tc.v.Tag != tc.tag {
			t.Errorf("got tag %v, want %v", tc.v.Tag, tc.tag)
		}
	}
}

func TestRecord_AlphabeticalOrdering(t *testing.T) {
	v := NewRecord([]RecordField{
		{"zebra", NewInt(1)},
		{"alpha", NewInt(2)},
		{"middle", NewInt(3)},
	})
	fields := v.AsRecord()
	if len(fields) != 3 {
		t.Fatalf("got %d fields, want 3", len(fields))
	}
	want := []string{"alpha", "middle", "zebra"}
	for i, name := range want {
		if fields[i].Name != name {
			t.Errorf("field %d: got %q, want %q", i, fields[i].Name, name)
		}
	}
}

func TestRecord_DuplicateFieldPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic on duplicate field")
		}
	}()
	_ = NewRecord([]RecordField{{"x", NewInt(1)}, {"x", NewInt(2)}})
}

func TestEqual_Reflexive(t *testing.T) {
	cases := []Value{
		NewInt(0), NewInt(42), NewInt(-1),
		NewFloat(0.0), NewFloat(3.14),
		NewBool(true), NewBool(false),
		Unit(),
		NewString(""), NewString("hello"),
		NewList(nil), NewList([]Value{NewInt(1), NewInt(2)}),
		NewTuple([]Value{NewInt(1), NewBool(false)}),
		NewRecord([]RecordField{{"a", NewInt(1)}, {"b", NewInt(2)}}),
		NewADT(3, []Value{NewInt(1), NewString("x")}),
	}
	for _, v := range cases {
		if !v.Equal(v) {
			t.Errorf("not reflexive: %v", v)
		}
	}
}

func TestEqual_DistinctTags(t *testing.T) {
	if NewInt(1).Equal(NewFloat(1.0)) {
		t.Error("Int(1) should not equal Float(1.0)")
	}
	if NewBool(true).Equal(NewInt(1)) {
		t.Error("Bool(true) should not equal Int(1)")
	}
}

func TestEqual_Records_DifferentOrderInputSameValue(t *testing.T) {
	a := NewRecord([]RecordField{{"x", NewInt(1)}, {"y", NewInt(2)}})
	b := NewRecord([]RecordField{{"y", NewInt(2)}, {"x", NewInt(1)}})
	if !a.Equal(b) {
		t.Error("records constructed with different field order should be equal")
	}
}

func TestEqual_Records_DifferentValues(t *testing.T) {
	a := NewRecord([]RecordField{{"x", NewInt(1)}})
	b := NewRecord([]RecordField{{"x", NewInt(2)}})
	if a.Equal(b) {
		t.Error("records with different values should not be equal")
	}
}

func TestEqual_Lists_RecursiveStructural(t *testing.T) {
	a := NewList([]Value{NewInt(1), NewList([]Value{NewInt(2), NewInt(3)})})
	b := NewList([]Value{NewInt(1), NewList([]Value{NewInt(2), NewInt(3)})})
	c := NewList([]Value{NewInt(1), NewList([]Value{NewInt(2), NewInt(4)})})
	if !a.Equal(b) {
		t.Error("nested lists with same values should be equal")
	}
	if a.Equal(c) {
		t.Error("nested lists with different leaf should not be equal")
	}
}

func TestEqual_Float_NaN(t *testing.T) {
	nan := NewFloat(math.NaN())
	if !nan.Equal(nan) {
		t.Error("NaN should equal NaN under structural Equal (for dedup)")
	}
}

func TestEqual_ADT_TagDifferentiates(t *testing.T) {
	a := NewADT(0, []Value{NewInt(1)})
	b := NewADT(1, []Value{NewInt(1)})
	if a.Equal(b) {
		t.Error("ADTs with different tags should not be equal")
	}
}

func TestEqual_Closure_PrototypeIdentity(t *testing.T) {
	p1 := &stubProto{name: "f"}
	p2 := &stubProto{name: "f"}
	c1 := NewClosure(p1, []Value{NewInt(1)})
	c2 := NewClosure(p1, []Value{NewInt(1)})
	c3 := NewClosure(p2, []Value{NewInt(1)})
	c4 := NewClosure(p1, []Value{NewInt(2)})
	if !c1.Equal(c2) {
		t.Error("same proto + same captures should be equal")
	}
	if c1.Equal(c3) {
		t.Error("different proto identity should differ even if names match")
	}
	if c1.Equal(c4) {
		t.Error("different captures should not be equal")
	}
}

func TestAccessors_PanicOnTagMismatch(t *testing.T) {
	cases := []func(){
		func() { NewInt(1).AsString() },
		func() { NewString("x").AsList() },
		func() { Unit().AsRecord() },
		func() { NewBool(true).AsClosure() },
	}
	for i, fn := range cases {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("case %d: expected panic", i)
				}
			}()
			fn()
		}()
	}
}

func TestString_Format(t *testing.T) {
	// Smoke test — just ensure no panic and non-empty output for every tag.
	cases := []Value{
		NewInt(1), NewFloat(1.5), NewBool(true), Unit(),
		NewString("x"), NewList([]Value{NewInt(1)}),
		NewTuple([]Value{NewInt(1), NewBool(false)}),
		NewRecord([]RecordField{{"k", NewInt(1)}}),
		NewClosure(&stubProto{name: "f"}, []Value{NewInt(1)}),
		NewADT(2, []Value{NewInt(1)}),
	}
	for _, v := range cases {
		if v.String() == "" {
			t.Errorf("empty String() for tag %v", v.Tag)
		}
	}
}

func TestTag_String(t *testing.T) {
	tags := []ValueTag{TagInt, TagFloat, TagBool, TagUnit, TagString, TagList, TagTuple, TagRecord, TagClosure, TagADT}
	for _, tag := range tags {
		if tag.String() == "" {
			t.Errorf("empty String() for tag %d", tag)
		}
	}
}
