package eval

import (
	"math"
	"testing"
)

// M-NUMERICS-VEC-ARRAY D1: an all-float array is packed into []float64, every
// other array is boxed, and no reader can tell the two apart.

func floatVals(xs ...float64) []Value {
	out := make([]Value, len(xs))
	for i, x := range xs {
		out[i] = &FloatValue{Value: x}
	}
	return out
}

func TestArrayStoreChoice(t *testing.T) {
	cases := []struct {
		name   string
		arr    *ArrayValue
		packed bool
	}{
		{"all floats", NewArray(floatVals(1.5, 2, 3)), true},
		{"empty", NewArray([]Value{}), false},
		{"ints", NewArray([]Value{&IntValue{Value: 1}}), false},
		{"mixed tail", NewArray(append(floatVals(1), &StringValue{Value: "x"})), false},
		{"float slice", NewFloatArray([]float64{1, 2}), true},
		{"empty float slice", NewFloatArray(nil), false},
		{"copy of floats", NewArrayCopy(floatVals(4, 5)), true},
	}
	for _, c := range cases {
		if got := c.arr.Packed(); got != c.packed {
			t.Errorf("%s: Packed() = %v, want %v", c.name, got, c.packed)
		}
	}
}

// Every reader must see the same thing from both stores. The boxed twin is
// built directly so the test does not depend on the constructor's choice.
func TestArrayPackedAndBoxedReadTheSame(t *testing.T) {
	xs := []float64{1.5, -2, 0, math.Inf(1), 1e-300}
	packed := NewFloatArray(append([]float64(nil), xs...))
	boxed := &ArrayValue{elems: floatVals(xs...)}
	if !packed.Packed() || boxed.Packed() {
		t.Fatal("fixture stores are wrong")
	}
	if packed.Len() != boxed.Len() {
		t.Fatalf("Len: %d vs %d", packed.Len(), boxed.Len())
	}
	if packed.String() != boxed.String() {
		t.Errorf("String: %q vs %q", packed.String(), boxed.String())
	}
	for i := range xs {
		p, ok1 := packed.Get(int64(i))
		b, ok2 := boxed.Get(int64(i))
		if !ok1 || !ok2 || p.String() != b.String() {
			t.Errorf("Get(%d): %v/%v vs %v/%v", i, p, ok1, b, ok2)
		}
	}
	pe := packed.Elements()
	for i, e := range pe {
		if e.(*FloatValue).Value != xs[i] {
			t.Errorf("Elements()[%d] = %v, want %v", i, e, xs[i])
		}
	}
	for _, i := range []int64{-1, int64(len(xs))} {
		if _, ok := packed.Get(i); ok {
			t.Errorf("Get(%d) on packed should be out of bounds", i)
		}
	}
}

func TestArraySetKeepsStoreAndDoesNotMutate(t *testing.T) {
	a := NewFloatArray([]float64{1, 2, 3})
	b, ok := a.Set(1, &FloatValue{Value: 9})
	if !ok || !b.Packed() {
		t.Fatalf("float set on a packed array must stay packed (ok=%v)", ok)
	}
	if a.String() != "#[1.0, 2.0, 3.0]" || b.String() != "#[1.0, 9.0, 3.0]" {
		t.Errorf("got a=%s b=%s", a, b)
	}
	if _, ok := a.Set(3, &FloatValue{Value: 0}); ok {
		t.Error("out-of-bounds Set on a packed array must fail")
	}
	// Only an untyped path can do this; the result must still be correct.
	c, ok := a.Set(0, &StringValue{Value: "x"})
	if !ok || c.Packed() || c.String() != "#[x, 2.0, 3.0]" {
		t.Errorf("non-float set: packed=%v %s", c.Packed(), c)
	}
}

// Elements() of a packed array is fresh: writing to it must not reach the array.
func TestArrayPackedElementsIsACopy(t *testing.T) {
	a := NewFloatArray([]float64{1, 2})
	a.Elements()[0] = &FloatValue{Value: 42}
	if v, _ := a.Get(0); v.(*FloatValue).Value != 1 {
		t.Errorf("packed store was modified through Elements(): %v", v)
	}
}
