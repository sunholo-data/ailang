package eval

import (
	"fmt"
	"strings"
)

// registerJSONBuiltins registers JSON encoding builtins
func registerJSONBuiltins() {
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
