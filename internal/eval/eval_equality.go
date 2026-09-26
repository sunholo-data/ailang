package eval

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/types"
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
		if !equal {
			if err := rejectFunctionEquality(a, b); err != nil {
				return nil, err
			}
		}

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
		if !equal {
			if err := rejectFunctionEquality(args[0], args[1]); err != nil {
				return nil, err
			}
		}
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
		// IEEE, like every other float == path: [nan] == [nan] is false
		// exactly as nan == nan is (types.FloatEq, #1274).
		bv, ok := b.(*FloatValue)
		return ok && types.FloatEq(av.Value, bv.Value)
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

// rejectFunctionEquality fails loudly when == reached a function value. The
// comparator's default answers false for kinds it cannot compare, which for a
// closure would be a silent wrong answer: the type checker must reject Eq on
// functions, so reaching one here is a soundness bug upstream, not a result.
// Only called when the comparison came out unequal, since that default can
// only ever produce false.
func rejectFunctionEquality(a, b Value) error {
	if containsFunctionValue(a) || containsFunctionValue(b) {
		return fmt.Errorf("internal error: == reached a function value (%s vs %s); "+
			"functions have no Eq and the type checker should have rejected this comparison", a.Type(), b.Type())
	}
	return nil
}

// containsFunctionValue reports whether v is, or structurally contains, a
// function. Cycle-safety: walks value trees built by evaluation, which are
// finite; closures are not entered.
func containsFunctionValue(v Value) bool {
	switch x := v.(type) {
	case *FunctionValue, *BuiltinFunction, *ConstructorClosure:
		return true
	case *ListValue:
		for _, e := range x.Elements {
			if containsFunctionValue(e) {
				return true
			}
		}
	case *TupleValue:
		for _, e := range x.Elements {
			if containsFunctionValue(e) {
				return true
			}
		}
	case *RecordValue:
		for _, f := range x.Fields {
			if containsFunctionValue(f) {
				return true
			}
		}
	case *TaggedValue:
		for _, f := range x.Fields {
			if containsFunctionValue(f) {
				return true
			}
		}
	}
	return false
}
