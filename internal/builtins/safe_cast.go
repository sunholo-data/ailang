package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/eval"
)

// ailangTypeName returns a user-friendly AILANG type name for a value,
// with conversion hints when applicable.
func ailangTypeName(v eval.Value, expected string) string {
	var typeName string
	switch v.(type) {
	case *eval.IntValue:
		typeName = "int"
	case *eval.FloatValue:
		typeName = "float"
	case *eval.StringValue:
		typeName = "string"
	case *eval.BoolValue:
		typeName = "bool"
	case *eval.ListValue:
		typeName = "list"
	case *eval.RecordValue:
		typeName = "record"
	case *eval.TaggedValue:
		typeName = "tagged value"
	default:
		typeName = fmt.Sprintf("%T", v)
	}

	hint := conversionHint(typeName, expected)
	if hint != "" {
		return typeName + ". " + hint
	}
	return typeName
}

// conversionHint suggests the right conversion function for common type mismatches.
func conversionHint(got, expected string) string {
	switch {
	case expected == "string" && got == "int":
		return "Use intToStr() to convert int to string"
	case expected == "string" && got == "float":
		return "Use floatToStr() to convert float to string"
	case expected == "string" && got == "bool":
		return "Use show() to convert bool to string"
	case expected == "int" && got == "float":
		return "Use floatToInt() to convert float to int"
	case expected == "float" && got == "int":
		return "Use intToFloat() to convert int to float"
	default:
		return ""
	}
}

// SafeAsString extracts the string value from an eval.Value
// Returns a descriptive error if the value is not a StringValue
func SafeAsString(v eval.Value) (string, error) {
	if sv, ok := v.(*eval.StringValue); ok {
		return sv.Value, nil
	}
	return "", fmt.Errorf("expected string, got %s", ailangTypeName(v, "string"))
}

// SafeAsInt extracts the integer value from an eval.Value
// Returns a descriptive error if the value is not an IntValue
func SafeAsInt(v eval.Value) (int, error) {
	if iv, ok := v.(*eval.IntValue); ok {
		return iv.Value, nil
	}
	return 0, fmt.Errorf("expected int, got %s", ailangTypeName(v, "int"))
}

// SafeAsFloat extracts the float value from an eval.Value
// Returns a descriptive error if the value is not a FloatValue
func SafeAsFloat(v eval.Value) (float64, error) {
	if fv, ok := v.(*eval.FloatValue); ok {
		return fv.Value, nil
	}
	return 0, fmt.Errorf("expected float, got %s", ailangTypeName(v, "float"))
}

// SafeAsBool extracts the boolean value from an eval.Value
// Returns a descriptive error if the value is not a BoolValue
func SafeAsBool(v eval.Value) (bool, error) {
	if bv, ok := v.(*eval.BoolValue); ok {
		return bv.Value, nil
	}
	return false, fmt.Errorf("expected bool, got %s", ailangTypeName(v, "bool"))
}

// SafeAsList extracts the list value from an eval.Value
// Returns a descriptive error if the value is not a ListValue
func SafeAsList(v eval.Value) ([]eval.Value, error) {
	if lv, ok := v.(*eval.ListValue); ok {
		return lv.Elements, nil
	}
	return nil, fmt.Errorf("expected list, got %s", ailangTypeName(v, "list"))
}

// SafeAsRecord extracts the record value from an eval.Value
// Returns a descriptive error if the value is not a RecordValue
func SafeAsRecord(v eval.Value) (map[string]eval.Value, error) {
	if rv, ok := v.(*eval.RecordValue); ok {
		return rv.Fields, nil
	}
	return nil, fmt.Errorf("expected record, got %s", ailangTypeName(v, "record"))
}

// SafeAsTagged extracts the tagged value from an eval.Value
// Returns a descriptive error if the value is not a TaggedValue
func SafeAsTagged(v eval.Value) (*eval.TaggedValue, error) {
	if tv, ok := v.(*eval.TaggedValue); ok {
		return tv, nil
	}
	return nil, fmt.Errorf("expected tagged value, got %s", ailangTypeName(v, "tagged value"))
}
