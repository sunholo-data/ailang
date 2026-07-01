package eval

import "testing"

// TestValuesStructurallyEqual covers the derived-(Eq) structural comparison used
// by both the derived-Eq dictionary method and the `==`/`!=` runtime fallback
// (the latter fixes deriving(Eq) failing at runtime inside polymorphic lambda
// closures over a user ADT — Felix ticket fb_cef305).
func TestValuesStructurallyEqual(t *testing.T) {
	tag := func(typ, ctor string, fields ...Value) *TaggedValue {
		return &TaggedValue{TypeName: typ, CtorName: ctor, Fields: fields}
	}
	i := func(n int) Value { return &IntValue{Value: n} }

	tests := []struct {
		name string
		a, b Value
		want bool
	}{
		{"nullary same ctor", tag("Color", "Red"), tag("Color", "Red"), true},
		{"nullary diff ctor", tag("Color", "Red"), tag("Color", "Blue"), false},
		{"fields equal", tag("Shape", "Rectangle", i(3), i(4)), tag("Shape", "Rectangle", i(3), i(4)), true},
		{"fields differ", tag("Shape", "Rectangle", i(3), i(4)), tag("Shape", "Rectangle", i(3), i(9)), false},
		{"diff ctor", tag("Shape", "Circle", i(1)), tag("Shape", "Rectangle", i(1), i(1)), false},
		{"nested ADT equal", tag("Box", "B", tag("Color", "Red")), tag("Box", "B", tag("Color", "Red")), true},
		{"nested ADT differ", tag("Box", "B", tag("Color", "Red")), tag("Box", "B", tag("Color", "Blue")), false},
		{"list equal", &ListValue{Elements: []Value{i(1), i(2)}}, &ListValue{Elements: []Value{i(1), i(2)}}, true},
		{"list differ", &ListValue{Elements: []Value{i(1), i(2)}}, &ListValue{Elements: []Value{i(1), i(9)}}, false},
		{"primitives equal", i(5), i(5), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := valuesStructurallyEqual(tt.a, tt.b); got != tt.want {
				t.Fatalf("valuesStructurallyEqual = %v, want %v", got, tt.want)
			}
		})
	}
}
