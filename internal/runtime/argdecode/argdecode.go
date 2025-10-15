// Package argdecode converts JSON arguments to AILANG eval.Value types
package argdecode

import (
	"encoding/json"
	"fmt"

	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// DecodeError represents an argument decoding error
type DecodeError struct {
	Expected string // Expected type (pretty-printed)
	Got      string // JSON value received
	Reason   string // Human-readable reason
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("ARG_DECODE_MISMATCH: expected %s, got %s\n  %s", e.Expected, e.Got, e.Reason)
}

// DecodeJSON converts a JSON string to an eval.Value based on the expected type
// Supports: null→(), number→int, string, bool, array→list, object→record
// Constraint: Only handles simple, non-polymorphic types for v0.1.0
func DecodeJSON(jsonStr string, expectedType types.Type) (eval.Value, error) {
	// Parse JSON
	var raw interface{}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return decodeValue(raw, expectedType)
}

// decodeValue recursively converts JSON values to eval.Value
func decodeValue(raw interface{}, expectedType types.Type) (eval.Value, error) {
	switch typ := expectedType.(type) {
	case *types.TCon:
		// Check for unit type
		if typ.Name == "Unit" || typ.Name == "unit" || typ.Name == "()" {
			if raw == nil {
				return &eval.UnitValue{}, nil
			}
			return nil, &DecodeError{
				Expected: "()",
				Got:      fmt.Sprintf("%v", raw),
				Reason:   "expected null for unit type",
			}
		}
		// Handle other TCon cases
		switch typ.Name {
		case "Int", "int":
			return decodeInt(raw)
		case "Float", "float":
			return decodeFloat(raw)
		case "String", "string":
			return decodeString(raw)
		case "Bool", "bool":
			return decodeBool(raw)
		default:
			return nil, fmt.Errorf("unsupported type constructor: %s", typ.Name)
		}

	case *types.TList:
		return decodeList(raw, typ.Element)

	case *types.TRecord:
		return decodeRecord(raw, typ)

	case *types.TVar2:
		// Type variable - try to infer from JSON structure
		// For v0.1.0, we'll do simple inference
		switch v := raw.(type) {
		case nil:
			return &eval.UnitValue{}, nil
		case float64:
			return &eval.IntValue{Value: int(v)}, nil
		case string:
			return &eval.StringValue{Value: v}, nil
		case bool:
			return &eval.BoolValue{Value: v}, nil
		case []interface{}:
			// Default to [int] for now
			return decodeList(raw, &types.TCon{Name: "int"})
		case map[string]interface{}:
			// Can't infer record type from type variable alone
			return nil, fmt.Errorf("cannot infer record type from JSON object with polymorphic type")
		default:
			return nil, fmt.Errorf("cannot infer type from JSON value: %v", raw)
		}

	default:
		return nil, fmt.Errorf("unsupported type for argument decoding: %T", expectedType)
	}
}

func decodeInt(raw interface{}) (eval.Value, error) {
	switch v := raw.(type) {
	case float64:
		return &eval.IntValue{Value: int(v)}, nil
	default:
		return nil, &DecodeError{
			Expected: "int",
			Got:      fmt.Sprintf("%v (%T)", raw, raw),
			Reason:   "expected JSON number for int type",
		}
	}
}

func decodeFloat(raw interface{}) (eval.Value, error) {
	switch v := raw.(type) {
	case float64:
		return &eval.FloatValue{Value: v}, nil
	default:
		return nil, &DecodeError{
			Expected: "float",
			Got:      fmt.Sprintf("%v (%T)", raw, raw),
			Reason:   "expected JSON number for float type",
		}
	}
}

func decodeString(raw interface{}) (eval.Value, error) {
	switch v := raw.(type) {
	case string:
		return &eval.StringValue{Value: v}, nil
	default:
		return nil, &DecodeError{
			Expected: "string",
			Got:      fmt.Sprintf("%v (%T)", raw, raw),
			Reason:   "expected JSON string",
		}
	}
}

func decodeBool(raw interface{}) (eval.Value, error) {
	switch v := raw.(type) {
	case bool:
		return &eval.BoolValue{Value: v}, nil
	default:
		return nil, &DecodeError{
			Expected: "bool",
			Got:      fmt.Sprintf("%v (%T)", raw, raw),
			Reason:   "expected JSON boolean",
		}
	}
}

func decodeList(raw interface{}, elemType types.Type) (eval.Value, error) {
	arr, ok := raw.([]interface{})
	if !ok {
		return nil, &DecodeError{
			Expected: fmt.Sprintf("[%s]", elemType),
			Got:      fmt.Sprintf("%v (%T)", raw, raw),
			Reason:   "expected JSON array for list type",
		}
	}

	elements := make([]eval.Value, len(arr))
	for i, elem := range arr {
		val, err := decodeValue(elem, elemType)
		if err != nil {
			return nil, fmt.Errorf("list element %d: %w", i, err)
		}
		elements[i] = val
	}

	return &eval.ListValue{Elements: elements}, nil
}

func decodeRecord(raw interface{}, recordType *types.TRecord) (eval.Value, error) {
	obj, ok := raw.(map[string]interface{})
	if !ok {
		return nil, &DecodeError{
			Expected: "record {...}",
			Got:      fmt.Sprintf("%v (%T)", raw, raw),
			Reason:   "expected JSON object for record type",
		}
	}

	fields := make(map[string]eval.Value)

	// Check all expected fields are present
	for fieldName, fieldType := range recordType.Fields {
		jsonVal, exists := obj[fieldName]
		if !exists {
			return nil, &DecodeError{
				Expected: fmt.Sprintf("record with field '%s'", fieldName),
				Got:      fmt.Sprintf("object missing field '%s'", fieldName),
				Reason:   fmt.Sprintf("required field '%s' not found in JSON", fieldName),
			}
		}

		val, err := decodeValue(jsonVal, fieldType)
		if err != nil {
			return nil, fmt.Errorf("field '%s': %w", fieldName, err)
		}
		fields[fieldName] = val
	}

	return &eval.RecordValue{Fields: fields}, nil
}
