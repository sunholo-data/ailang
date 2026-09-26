package eval

import (
	"strings"
	"testing"
)

// TestStructuralEqualityRejectsFunctions: a function reaching == is a
// soundness bug upstream; it must error, never answer false. The sprint
// evaluator found `func pick[a](x: a, y: a) { [x] == [y] }` printing false for
// two closures (M-EQ-DERIVE-CONTAINERS round 1).
func TestStructuralEqualityRejectsFunctions(t *testing.T) {
	fn := &FunctionValue{}
	cases := map[string][2]Value{
		"bare":     {fn, fn},
		"in list":  {&ListValue{Elements: []Value{fn}}, &ListValue{Elements: []Value{fn}}},
		"in tuple": {&TupleValue{Elements: []Value{&IntValue{Value: 1}, fn}}, &TupleValue{Elements: []Value{&IntValue{Value: 1}, fn}}},
		"in ADT":   {&TaggedValue{CtorName: "Some", Fields: []Value{fn}}, &TaggedValue{CtorName: "Some", Fields: []Value{fn}}},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			for _, isEq := range []bool{true, false} {
				_, err := makeStructuralEqualityFn(isEq)(args[:])
				if err == nil || !strings.Contains(err.Error(), "function value") {
					t.Errorf("isEq=%v: want a function-value error, got %v", isEq, err)
				}
			}
		})
	}
	// Plain data still compares.
	v, err := makeStructuralEqualityFn(true)([]Value{&IntValue{Value: 1}, &IntValue{Value: 2}})
	if err != nil || v.(*BoolValue).Value {
		t.Errorf("1 == 2: got %v, %v", v, err)
	}
}
