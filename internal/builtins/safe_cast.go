package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/eval"
)

// SafeAsString extracts the string value from an eval.Value
// Returns a descriptive error if the value is not a StringValue
func SafeAsString(v eval.Value) (string, error) {
	if sv, ok := v.(*eval.StringValue); ok {
		return sv.Value, nil
	}
	return "", fmt.Errorf("expected string, got %T", v)
}

// SafeAsInt extracts the integer value from an eval.Value
// Returns a descriptive error if the value is not an IntValue
func SafeAsInt(v eval.Value) (int, error) {
	if iv, ok := v.(*eval.IntValue); ok {
		return iv.Value, nil
	}
	return 0, fmt.Errorf("expected int, got %T", v)
}

// SafeAsFloat extracts the float value from an eval.Value
// Returns a descriptive error if the value is not a FloatValue
func SafeAsFloat(v eval.Value) (float64, error) {
	if fv, ok := v.(*eval.FloatValue); ok {
		return fv.Value, nil
	}
	return 0, fmt.Errorf("expected float, got %T", v)
}

// SafeAsBool extracts the boolean value from an eval.Value
// Returns a descriptive error if the value is not a BoolValue
func SafeAsBool(v eval.Value) (bool, error) {
	if bv, ok := v.(*eval.BoolValue); ok {
		return bv.Value, nil
	}
	return false, fmt.Errorf("expected bool, got %T", v)
}

// SafeAsList extracts the list value from an eval.Value
// Returns a descriptive error if the value is not a ListValue
func SafeAsList(v eval.Value) ([]eval.Value, error) {
	if lv, ok := v.(*eval.ListValue); ok {
		return lv.Elements, nil
	}
	return nil, fmt.Errorf("expected list, got %T", v)
}

// SafeAsRecord extracts the record value from an eval.Value
// Returns a descriptive error if the value is not a RecordValue
func SafeAsRecord(v eval.Value) (map[string]eval.Value, error) {
	if rv, ok := v.(*eval.RecordValue); ok {
		return rv.Fields, nil
	}
	return nil, fmt.Errorf("expected record, got %T", v)
}

// SafeAsTagged extracts the tagged value from an eval.Value
// Returns a descriptive error if the value is not a TaggedValue
func SafeAsTagged(v eval.Value) (*eval.TaggedValue, error) {
	if tv, ok := v.(*eval.TaggedValue); ok {
		return tv, nil
	}
	return nil, fmt.Errorf("expected tagged value, got %T", v)
}
