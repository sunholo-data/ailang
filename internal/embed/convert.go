package embed

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/sunholo/ailang/internal/eval"
)

// FromGo converts a Go value to an AILANG value.
// Supported types:
//   - nil → UnitValue
//   - bool → BoolValue
//   - int, int8, int16, int32, int64 → IntValue
//   - uint, uint8, uint16, uint32, uint64 → IntValue
//   - float32, float64 → FloatValue
//   - string → StringValue
//   - []byte → BytesValue
//   - []T → ListValue (recursive)
//   - map[string]T → RecordValue (recursive)
//   - struct → RecordValue (exported fields only)
func FromGo(v interface{}) (eval.Value, error) {
	return fromGoInternal(v, false)
}

// FromGoPreserveFloats converts a Go value to an AILANG value, preserving float64
// as FloatValue even for whole numbers. Use this for direct Go calls where you want
// float64 to remain as float (not converted to int for JSON compatibility).
func FromGoPreserveFloats(v interface{}) (eval.Value, error) {
	return fromGoInternal(v, true)
}

func fromGoInternal(v interface{}, preserveFloats bool) (eval.Value, error) {
	if v == nil {
		return &eval.UnitValue{}, nil
	}

	// If the value is already an eval.Value, return it directly.
	// This happens when multipart uploads pass *eval.BytesValue as args.
	if ev, ok := v.(eval.Value); ok {
		return ev, nil
	}

	rv := reflect.ValueOf(v)
	return fromReflect(rv, preserveFloats)
}

func fromReflect(rv reflect.Value, preserveFloats bool) (eval.Value, error) {
	// Handle pointers
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return &eval.UnitValue{}, nil
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Bool:
		return &eval.BoolValue{Value: rv.Bool()}, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &eval.IntValue{Value: int(rv.Int())}, nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &eval.IntValue{Value: int(rv.Uint())}, nil

	case reflect.Float32, reflect.Float64:
		f := rv.Float()
		// JSON unmarshals all numbers as float64. If the value is a whole number,
		// convert to IntValue for compatibility with AILANG int-typed functions.
		// However, when preserveFloats is true (direct Go calls), keep float64 as FloatValue.
		if !preserveFloats && f == float64(int(f)) && f >= -1e15 && f <= 1e15 {
			return &eval.IntValue{Value: int(f)}, nil
		}
		return &eval.FloatValue{Value: f}, nil

	case reflect.String:
		return &eval.StringValue{Value: rv.String()}, nil

	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			// []byte → BytesValue
			return &eval.BytesValue{Value: rv.Bytes()}, nil
		}
		// Other slices → ListValue
		elements := make([]eval.Value, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			elem, err := fromReflect(rv.Index(i), preserveFloats)
			if err != nil {
				return nil, fmt.Errorf("slice element %d: %w", i, err)
			}
			elements[i] = elem
		}
		return &eval.ListValue{Elements: elements}, nil

	case reflect.Array:
		elements := make([]eval.Value, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			elem, err := fromReflect(rv.Index(i), preserveFloats)
			if err != nil {
				return nil, fmt.Errorf("array element %d: %w", i, err)
			}
			elements[i] = elem
		}
		return &eval.ArrayValue{Elements: elements}, nil

	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("map keys must be strings, got %s", rv.Type().Key())
		}
		fields := make(map[string]eval.Value)
		iter := rv.MapRange()
		for iter.Next() {
			key := iter.Key().String()
			val, err := fromReflect(iter.Value(), preserveFloats)
			if err != nil {
				return nil, fmt.Errorf("map value for key %q: %w", key, err)
			}
			fields[key] = val
		}
		return &eval.RecordValue{Fields: fields}, nil

	case reflect.Struct:
		fields := make(map[string]eval.Value)
		rt := rv.Type()
		for i := 0; i < rv.NumField(); i++ {
			field := rt.Field(i)
			if !field.IsExported() {
				continue
			}
			// Use JSON tag if present, otherwise field name
			name := field.Name
			if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
				name = tag
			}
			val, err := fromReflect(rv.Field(i), preserveFloats)
			if err != nil {
				return nil, fmt.Errorf("struct field %s: %w", field.Name, err)
			}
			fields[name] = val
		}
		return &eval.RecordValue{Fields: fields}, nil

	default:
		return nil, fmt.Errorf("unsupported type: %s", rv.Type())
	}
}

// ToGo converts an AILANG value to a Go value.
// The result type depends on the input:
//   - UnitValue → nil
//   - BoolValue → bool
//   - IntValue → int
//   - FloatValue → float64
//   - StringValue → string
//   - BytesValue → []byte
//   - ListValue → []interface{}
//   - ArrayValue → []interface{}
//   - TupleValue → []interface{}
//   - RecordValue → map[string]interface{}
//   - TaggedValue → map[string]interface{} with "__tag" key
func ToGo(v eval.Value) (interface{}, error) {
	if v == nil {
		return nil, nil
	}

	switch val := v.(type) {
	case *eval.UnitValue:
		return nil, nil

	case *eval.BoolValue:
		return val.Value, nil

	case *eval.IntValue:
		return val.Value, nil

	case *eval.FloatValue:
		return val.Value, nil

	case *eval.StringValue:
		return val.Value, nil

	case *eval.BytesValue:
		return val.Value, nil

	case *eval.ListValue:
		result := make([]interface{}, len(val.Elements))
		for i, elem := range val.Elements {
			goVal, err := ToGo(elem)
			if err != nil {
				return nil, fmt.Errorf("list element %d: %w", i, err)
			}
			result[i] = goVal
		}
		return result, nil

	case *eval.ArrayValue:
		result := make([]interface{}, len(val.Elements))
		for i, elem := range val.Elements {
			goVal, err := ToGo(elem)
			if err != nil {
				return nil, fmt.Errorf("array element %d: %w", i, err)
			}
			result[i] = goVal
		}
		return result, nil

	case *eval.TupleValue:
		result := make([]interface{}, len(val.Elements))
		for i, elem := range val.Elements {
			goVal, err := ToGo(elem)
			if err != nil {
				return nil, fmt.Errorf("tuple element %d: %w", i, err)
			}
			result[i] = goVal
		}
		return result, nil

	case *eval.RecordValue:
		result := make(map[string]interface{})
		for key, elem := range val.Fields {
			goVal, err := ToGo(elem)
			if err != nil {
				return nil, fmt.Errorf("record field %q: %w", key, err)
			}
			result[key] = goVal
		}
		return result, nil

	case *eval.TaggedValue:
		// Convert ADT constructors to maps with a tag
		result := map[string]interface{}{
			"__type": val.TypeName,
			"__tag":  val.CtorName,
		}
		if len(val.Fields) > 0 {
			fields := make([]interface{}, len(val.Fields))
			for i, f := range val.Fields {
				goVal, err := ToGo(f)
				if err != nil {
					return nil, fmt.Errorf("tagged value field %d: %w", i, err)
				}
				fields[i] = goVal
			}
			result["fields"] = fields
		}
		return result, nil

	case *eval.FunctionValue:
		return nil, fmt.Errorf("cannot convert function to Go value")

	case *eval.ErrorValue:
		return nil, fmt.Errorf("AILANG error: %s", val.Message)

	case *eval.IndirectValue:
		if val.Cell != nil && val.Cell.Val != nil {
			return ToGo(val.Cell.Val)
		}
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown AILANG value type: %T", v)
	}
}

// ToJSON converts an AILANG value to JSON bytes.
func ToJSON(v eval.Value) ([]byte, error) {
	goVal, err := ToGo(v)
	if err != nil {
		return nil, err
	}
	return json.Marshal(goVal)
}

// FromJSON converts JSON bytes to an AILANG value.
func FromJSON(data []byte) (eval.Value, error) {
	var goVal interface{}
	if err := json.Unmarshal(data, &goVal); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return FromGo(goVal)
}

// Type-safe extraction helpers

// ToInt extracts an int from an AILANG value.
func ToInt(v eval.Value) (int, error) {
	if iv, ok := v.(*eval.IntValue); ok {
		return iv.Value, nil
	}
	return 0, fmt.Errorf("expected int, got %T", v)
}

// ToFloat extracts a float64 from an AILANG value.
func ToFloat(v eval.Value) (float64, error) {
	switch val := v.(type) {
	case *eval.FloatValue:
		return val.Value, nil
	case *eval.IntValue:
		return float64(val.Value), nil
	default:
		return 0, fmt.Errorf("expected float, got %T", v)
	}
}

// ToString extracts a string from an AILANG value.
func ToString(v eval.Value) (string, error) {
	if sv, ok := v.(*eval.StringValue); ok {
		return sv.Value, nil
	}
	return "", fmt.Errorf("expected string, got %T", v)
}

// ToBool extracts a bool from an AILANG value.
func ToBool(v eval.Value) (bool, error) {
	if bv, ok := v.(*eval.BoolValue); ok {
		return bv.Value, nil
	}
	return false, fmt.Errorf("expected bool, got %T", v)
}

// ToBytes extracts a []byte from an AILANG value.
func ToBytes(v eval.Value) ([]byte, error) {
	if bv, ok := v.(*eval.BytesValue); ok {
		return bv.Value, nil
	}
	return nil, fmt.Errorf("expected bytes, got %T", v)
}

// ToList extracts a []eval.Value from an AILANG value.
func ToList(v eval.Value) ([]eval.Value, error) {
	if lv, ok := v.(*eval.ListValue); ok {
		return lv.Elements, nil
	}
	return nil, fmt.Errorf("expected list, got %T", v)
}

// ToRecord extracts a map[string]eval.Value from an AILANG value.
func ToRecord(v eval.Value) (map[string]eval.Value, error) {
	if rv, ok := v.(*eval.RecordValue); ok {
		return rv.Fields, nil
	}
	return nil, fmt.Errorf("expected record, got %T", v)
}

// IsUnit checks if a value is the unit value (equivalent to nil/void).
func IsUnit(v eval.Value) bool {
	_, ok := v.(*eval.UnitValue)
	return ok
}
