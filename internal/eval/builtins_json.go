package eval

import (
	"encoding/json"
	"fmt"
	"strings"
)

// registerJSONBuiltins registers JSON encoding builtins
func registerJSONBuiltins() {
	// _json_decode: Decode JSON string to Json ADT
	//  Returns: Result[Json, string]
	Builtins["_json_decode"] = &BuiltinFunc{
		Name:    "_json_decode",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(v Value) (Value, error) {
			// Handle already-structured AILANG values by converting to Json ADT
			switch val := v.(type) {
			case *StringValue:
				// Standard path: parse JSON string
				return legacyDecodeString(val)

			case *RecordValue:
				// Pre-parsed data → convert to JObject
				jsonVal := legacyValueToJSON(val)
				return legacyWrapOk(jsonVal), nil

			case *ListValue:
				// Pre-parsed array → convert to JArray
				jsonVal := legacyValueToJSON(val)
				return legacyWrapOk(jsonVal), nil

			case *TaggedValue:
				// Try to unwrap Result/Option wrappers containing a string
				if strVal, ok := unwrapTaggedToString(val); ok {
					return legacyDecodeString(strVal)
				}
				if val.TypeName == "Json" {
					return legacyWrapOk(val), nil
				}
				return nil, fmt.Errorf("_json_decode: expected string, got %s(%s) — unwrap the %s before calling json_decode",
					val.TypeName, val.CtorName, val.TypeName)

			case *IntValue, *FloatValue, *BoolValue:
				jsonVal := legacyValueToJSON(val)
				return legacyWrapOk(jsonVal), nil

			default:
				return nil, fmt.Errorf("_json_decode: expected string, got %T", v)
			}
		},
	}

	// _json_encode: Encode Json ADT to JSON string
	Builtins["_json_encode"] = &BuiltinFunc{
		Name:    "_json_encode",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(v Value) (*StringValue, error) {
			tagged, ok := v.(*TaggedValue)
			if !ok {
				return nil, fmt.Errorf("_json_encode: expected Json ADT, got %T", v)
			}

			if tagged.TypeName != "Json" {
				return nil, fmt.Errorf("_json_encode: expected Json ADT, got %s", tagged.TypeName)
			}

			result, err := encodeJSON(tagged)
			if err != nil {
				return nil, err
			}

			return &StringValue{Value: result}, nil
		},
	}
}

// encodeJSON recursively encodes a Json ADT value
func encodeJSON(v *TaggedValue) (string, error) {
	switch v.CtorName {
	case "JNull":
		return "null", nil

	case "JBool":
		if len(v.Fields) != 1 {
			return "", fmt.Errorf("JBool: expected 1 field, got %d", len(v.Fields))
		}
		b, ok := v.Fields[0].(*BoolValue)
		if !ok {
			return "", fmt.Errorf("JBool: expected bool field, got %T", v.Fields[0])
		}
		if b.Value {
			return "true", nil
		}
		return "false", nil

	case "JNumber":
		if len(v.Fields) != 1 {
			return "", fmt.Errorf("JNumber: expected 1 field, got %d", len(v.Fields))
		}

		// Try float first
		if f, ok := v.Fields[0].(*FloatValue); ok {
			return fmt.Sprintf("%g", f.Value), nil
		}

		// Also accept int (will be converted)
		if i, ok := v.Fields[0].(*IntValue); ok {
			return fmt.Sprintf("%d", i.Value), nil
		}

		return "", fmt.Errorf("JNumber: expected float or int field, got %T", v.Fields[0])

	case "JString":
		if len(v.Fields) != 1 {
			return "", fmt.Errorf("JString: expected 1 field, got %d", len(v.Fields))
		}
		s, ok := v.Fields[0].(*StringValue)
		if !ok {
			return "", fmt.Errorf("JString: expected string field, got %T", v.Fields[0])
		}
		return encodeJSONString(s.Value), nil

	case "JArray":
		if len(v.Fields) != 1 {
			return "", fmt.Errorf("JArray: expected 1 field, got %d", len(v.Fields))
		}
		list, ok := v.Fields[0].(*ListValue)
		if !ok {
			return "", fmt.Errorf("JArray: expected List field, got %T", v.Fields[0])
		}
		return encodeJSONArray(list)

	case "JObject":
		if len(v.Fields) != 1 {
			return "", fmt.Errorf("JObject: expected 1 field, got %d", len(v.Fields))
		}
		list, ok := v.Fields[0].(*ListValue)
		if !ok {
			return "", fmt.Errorf("JObject: expected List field, got %T", v.Fields[0])
		}
		return encodeJSONObject(list)

	default:
		return "", fmt.Errorf("unknown Json constructor: %s", v.CtorName)
	}
}

// encodeJSONString encodes a string with proper JSON escaping
func encodeJSONString(s string) string {
	var b strings.Builder
	b.WriteByte('"')

	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			// Control characters (0x00-0x1F) must be escaped
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else if r > 0xFFFF {
				// Encode as UTF-16 surrogate pair
				r1 := ((r - 0x10000) >> 10) + 0xD800
				r2 := ((r - 0x10000) & 0x3FF) + 0xDC00
				fmt.Fprintf(&b, `\u%04x\u%04x`, r1, r2)
			} else {
				// Normal character
				b.WriteRune(r)
			}
		}
	}

	b.WriteByte('"')
	return b.String()
}

// encodeJSONArray encodes a List[Json] as JSON array
func encodeJSONArray(list *ListValue) (string, error) {
	var b strings.Builder
	b.WriteByte('[')

	for i, elem := range list.Elements {
		if i > 0 {
			b.WriteByte(',')
		}

		tagged, ok := elem.(*TaggedValue)
		if !ok {
			return "", fmt.Errorf("JArray element %d: expected Json ADT, got %T", i, elem)
		}

		encoded, err := encodeJSON(tagged)
		if err != nil {
			return "", fmt.Errorf("JArray element %d: %w", i, err)
		}

		b.WriteString(encoded)
	}

	b.WriteByte(']')
	return b.String(), nil
}

// encodeJSONObject encodes a List[{key, value}] as JSON object
func encodeJSONObject(list *ListValue) (string, error) {
	var b strings.Builder
	b.WriteByte('{')

	for i, elem := range list.Elements {
		if i > 0 {
			b.WriteByte(',')
		}

		// Extract {key, value} record
		rec, ok := elem.(*RecordValue)
		if !ok {
			return "", fmt.Errorf("JObject element %d: expected record, got %T", i, elem)
		}

		keyVal, ok := rec.Fields["key"]
		if !ok {
			return "", fmt.Errorf("JObject element %d: missing 'key' field", i)
		}
		keyStr, ok := keyVal.(*StringValue)
		if !ok {
			return "", fmt.Errorf("JObject element %d: 'key' must be string, got %T", i, keyVal)
		}

		valueVal, ok := rec.Fields["value"]
		if !ok {
			return "", fmt.Errorf("JObject element %d: missing 'value' field", i)
		}
		valueTagged, ok := valueVal.(*TaggedValue)
		if !ok {
			return "", fmt.Errorf("JObject element %d: 'value' must be Json ADT, got %T", i, valueVal)
		}

		// Encode key
		b.WriteString(encodeJSONString(keyStr.Value))
		b.WriteByte(':')

		// Encode value
		encoded, err := encodeJSON(valueTagged)
		if err != nil {
			return "", fmt.Errorf("JObject element %d value: %w", i, err)
		}
		b.WriteString(encoded)
	}

	b.WriteByte('}')
	return b.String(), nil
}

// unwrapTaggedToString extracts a string from common TaggedValue wrappers.
// Handles Ok(string), Some(string), and single-field constructors wrapping a string.
func unwrapTaggedToString(tv *TaggedValue) (*StringValue, bool) {
	if len(tv.Fields) != 1 {
		return nil, false
	}
	if sv, ok := tv.Fields[0].(*StringValue); ok {
		return sv, true
	}
	if inner, ok := tv.Fields[0].(*TaggedValue); ok {
		return unwrapTaggedToString(inner)
	}
	return nil, false
}

// legacyDecodeString parses a JSON string into the Json ADT, returning Result[Json, string].
func legacyDecodeString(str *StringValue) (Value, error) {
	dec := json.NewDecoder(strings.NewReader(str.Value))
	dec.UseNumber()
	var result interface{}
	if err := dec.Decode(&result); err != nil {
		return legacyWrapErr(err.Error()), nil
	}
	jsonVal, err := interfaceToJSON(result)
	if err != nil {
		return legacyWrapErr(err.Error()), nil
	}
	return legacyWrapOk(jsonVal), nil
}

// legacyValueToJSON converts an AILANG value to the Json ADT.
func legacyValueToJSON(v Value) Value {
	switch val := v.(type) {
	case *StringValue:
		return &TaggedValue{ModulePath: "std/json", TypeName: "Json", CtorName: "JString", Fields: []Value{val}}
	case *IntValue:
		return &TaggedValue{ModulePath: "std/json", TypeName: "Json", CtorName: "JNumber", Fields: []Value{&FloatValue{Value: float64(val.Value)}}}
	case *FloatValue:
		return &TaggedValue{ModulePath: "std/json", TypeName: "Json", CtorName: "JNumber", Fields: []Value{val}}
	case *BoolValue:
		return &TaggedValue{ModulePath: "std/json", TypeName: "Json", CtorName: "JBool", Fields: []Value{val}}
	case *UnitValue:
		return &TaggedValue{ModulePath: "std/json", TypeName: "Json", CtorName: "JNull", Fields: []Value{}}
	case *ListValue:
		elements := make([]Value, len(val.Elements))
		for i, elem := range val.Elements {
			elements[i] = legacyValueToJSON(elem)
		}
		return &TaggedValue{ModulePath: "std/json", TypeName: "Json", CtorName: "JArray", Fields: []Value{&ListValue{Elements: elements}}}
	case *RecordValue:
		kvPairs := make([]Value, 0, len(val.Fields))
		for key, fieldVal := range val.Fields {
			kvPairs = append(kvPairs, &RecordValue{
				Fields: map[string]Value{
					"key":   &StringValue{Value: key},
					"value": legacyValueToJSON(fieldVal),
				},
			})
		}
		return &TaggedValue{ModulePath: "std/json", TypeName: "Json", CtorName: "JObject", Fields: []Value{&ListValue{Elements: kvPairs}}}
	case *TaggedValue:
		if val.TypeName == "Json" {
			return val
		}
		fields := []Value{
			&RecordValue{Fields: map[string]Value{"key": &StringValue{Value: "__type"}, "value": legacyValueToJSON(&StringValue{Value: val.TypeName})}},
			&RecordValue{Fields: map[string]Value{"key": &StringValue{Value: "__tag"}, "value": legacyValueToJSON(&StringValue{Value: val.CtorName})}},
		}
		if len(val.Fields) > 0 {
			fieldElements := make([]Value, len(val.Fields))
			for i, f := range val.Fields {
				fieldElements[i] = legacyValueToJSON(f)
			}
			fields = append(fields, &RecordValue{
				Fields: map[string]Value{
					"key":   &StringValue{Value: "fields"},
					"value": &TaggedValue{ModulePath: "std/json", TypeName: "Json", CtorName: "JArray", Fields: []Value{&ListValue{Elements: fieldElements}}},
				},
			})
		}
		return &TaggedValue{ModulePath: "std/json", TypeName: "Json", CtorName: "JObject", Fields: []Value{&ListValue{Elements: fields}}}
	default:
		return &TaggedValue{ModulePath: "std/json", TypeName: "Json", CtorName: "JString", Fields: []Value{&StringValue{Value: fmt.Sprintf("%v", v)}}}
	}
}

func legacyWrapOk(v Value) Value {
	return &TaggedValue{ModulePath: "std/result", TypeName: "Result", CtorName: "Ok", Fields: []Value{v}}
}

func legacyWrapErr(msg string) Value {
	return &TaggedValue{ModulePath: "std/result", TypeName: "Result", CtorName: "Err", Fields: []Value{&StringValue{Value: msg}}}
}

// interfaceToJSON converts Go's json.Decode output to AILANG Json ADT
func interfaceToJSON(v interface{}) (Value, error) {
	switch val := v.(type) {
	case nil:
		// JNull constructor
		return &TaggedValue{
			ModulePath: "std/json",
			TypeName:   "Json",
			CtorName:   "JNull",
			Fields:     []Value{},
		}, nil

	case bool:
		// JBool(bool) constructor
		return &TaggedValue{
			ModulePath: "std/json",
			TypeName:   "Json",
			CtorName:   "JBool",
			Fields:     []Value{&BoolValue{Value: val}},
		}, nil

	case json.Number:
		// JNumber(float) constructor
		// Check if it's a float or integer
		str := string(val)
		if strings.ContainsAny(str, ".eE") {
			// Float
			f, _ := val.Float64()
			return &TaggedValue{
				ModulePath: "std/json",
				TypeName:   "Json",
				CtorName:   "JNumber",
				Fields:     []Value{&FloatValue{Value: f}},
			}, nil
		}
		// Integer - convert to float for consistency
		i, _ := val.Int64()
		return &TaggedValue{
			ModulePath: "std/json",
			TypeName:   "Json",
			CtorName:   "JNumber",
			Fields:     []Value{&FloatValue{Value: float64(i)}},
		}, nil

	case string:
		// JString(string) constructor
		return &TaggedValue{
			ModulePath: "std/json",
			TypeName:   "Json",
			CtorName:   "JString",
			Fields:     []Value{&StringValue{Value: val}},
		}, nil

	case []interface{}:
		// JArray(List[Json]) constructor
		elements := make([]Value, 0, len(val))
		for i, elem := range val {
			jsonElem, err := interfaceToJSON(elem)
			if err != nil {
				return nil, fmt.Errorf("array element %d: %w", i, err)
			}
			elements = append(elements, jsonElem)
		}
		return &TaggedValue{
			ModulePath: "std/json",
			TypeName:   "Json",
			CtorName:   "JArray",
			Fields:     []Value{&ListValue{Elements: elements}},
		}, nil

	case map[string]interface{}:
		// JObject(List[{key: string, value: Json}]) constructor
		// Preserve insertion order by extracting keys first
		kvPairs := make([]Value, 0, len(val))
		for key, value := range val {
			jsonValue, err := interfaceToJSON(value)
			if err != nil {
				return nil, fmt.Errorf("object field %q: %w", key, err)
			}
			kvPairs = append(kvPairs, &RecordValue{
				Fields: map[string]Value{
					"key":   &StringValue{Value: key},
					"value": jsonValue,
				},
			})
		}
		return &TaggedValue{
			ModulePath: "std/json",
			TypeName:   "Json",
			CtorName:   "JObject",
			Fields:     []Value{&ListValue{Elements: kvPairs}},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported JSON type: %T", v)
	}
}
