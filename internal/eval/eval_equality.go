package eval

import (
	"fmt"
	"math"
)

// Structural equality behind every derived and synthesized Eq instance
// (M-DX19 ADTs; M-EQ-DERIVE-CONTAINERS lists, Option, Result, tuples, records).

// makeADTEqualityFn creates an equality function for ADT types with deriving (Eq).
// M-DX19: Compares TaggedValue instances structurally.
// If isEq is true, returns eq function (true if equal).
// If isEq is false, returns neq function (true if not equal).
func makeADTEqualityFn(typeName string, isEq bool) func([]Value) (Value, error) {
	return func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("derived Eq for %s expects 2 arguments, got %d", typeName, len(args))
		}

		// Both values must be TaggedValue of the same ADT type
		a, okA := args[0].(*TaggedValue)
		b, okB := args[1].(*TaggedValue)

		if !okA || !okB {
			return nil, fmt.Errorf("derived Eq for %s: expected TaggedValue, got %T and %T", typeName, args[0], args[1])
		}

		// Compare structurally
		equal := taggedValuesEqual(a, b)

		if isEq {
			return &BoolValue{Value: equal}, nil
		}
		return &BoolValue{Value: !equal}, nil
	}
}

// makeStructuralEqualityFn backs every synthesized Eq instance (lists, Option,
// Result, tuples, derived records). The type checker has already proved every
// part has Eq, so structural comparison is exactly the composed semantics.
// M-EQ-DERIVE-CONTAINERS
func makeStructuralEqualityFn(isEq bool) func([]Value) (Value, error) {
	return func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("structural Eq expects 2 arguments, got %d", len(args))
		}
		equal := valuesStructurallyEqual(args[0], args[1])
		return &BoolValue{Value: equal == isEq}, nil
	}
}

// taggedValuesEqual compares two TaggedValue instances structurally.
// M-DX19: Returns true if they have the same constructor and all fields are equal.
func taggedValuesEqual(a, b *TaggedValue) bool {
	// Must have same constructor
	if a.CtorName != b.CtorName {
		return false
	}

	// Must have same number of fields
	if len(a.Fields) != len(b.Fields) {
		return false
	}

	// Compare each field recursively
	for i := range a.Fields {
		if !valuesStructurallyEqual(a.Fields[i], b.Fields[i]) {
			return false
		}
	}

	return true
}

// valuesStructurallyEqual compares two values for structural equality.
// M-DX19: Supports primitive types, TaggedValues, lists, records, and tuples.
func valuesStructurallyEqual(a, b Value) bool {
	// Handle nil case
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Compare by type
	switch av := a.(type) {
	case *IntValue:
		bv, ok := b.(*IntValue)
		return ok && av.Value == bv.Value
	case *FloatValue:
		// Lawful, like the Eq[Float] dictionary: NaN == NaN. Structural equality
		// must agree with the composed dictionaries, so [nan] == [nan] is true
		// exactly as nan == nan is (M-EQ-DERIVE-CONTAINERS).
		bv, ok := b.(*FloatValue)
		if !ok {
			return false
		}
		if math.IsNaN(av.Value) && math.IsNaN(bv.Value) {
			return true
		}
		return av.Value == bv.Value
	case *StringValue:
		bv, ok := b.(*StringValue)
		return ok && av.Value == bv.Value
	case *BoolValue:
		bv, ok := b.(*BoolValue)
		return ok && av.Value == bv.Value
	case *UnitValue:
		_, ok := b.(*UnitValue)
		return ok
	case *TaggedValue:
		bv, ok := b.(*TaggedValue)
		return ok && taggedValuesEqual(av, bv)
	case *ListValue:
		bv, ok := b.(*ListValue)
		if !ok || len(av.Elements) != len(bv.Elements) {
			return false
		}
		for i := range av.Elements {
			if !valuesStructurallyEqual(av.Elements[i], bv.Elements[i]) {
				return false
			}
		}
		return true
	case *TupleValue:
		bv, ok := b.(*TupleValue)
		if !ok || len(av.Elements) != len(bv.Elements) {
			return false
		}
		for i := range av.Elements {
			if !valuesStructurallyEqual(av.Elements[i], bv.Elements[i]) {
				return false
			}
		}
		return true
	case *RecordValue:
		bv, ok := b.(*RecordValue)
		if !ok || len(av.Fields) != len(bv.Fields) {
			return false
		}
		for k, v := range av.Fields {
			bField, exists := bv.Fields[k]
			if !exists || !valuesStructurallyEqual(v, bField) {
				return false
			}
		}
		return true
	default:
		// Unknown types are not equal
		return false
	}
}
