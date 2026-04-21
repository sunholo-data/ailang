package builtins

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// JSON encoder - converts AILANG Json ADT to JSON string

// Builtin registration

func init() {
	registerJSONEncode()
}

func registerJSONEncode() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/json",
		Name:    "_json_encode",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeJSONEncodeType,
		Impl:    jsonEncodeImpl,
		Metadata: &BuiltinMetadata{
			Description: "Convert a Json ADT value to a JSON string",
			LongDesc: `Converts AILANG's Json algebraic data type to a JSON string using RFC 8259 compliant encoding.
Supports all JSON types: objects, arrays, strings, numbers, booleans, and null.
String escaping follows RFC 8259 specification with proper handling of control characters and unicode.`,
			Params: []ParamDoc{
				{Name: "value", Description: "Json ADT value to encode"},
			},
			Returns: "string - JSON representation of the value",
			Examples: []Example{
				{Code: `_json_encode(JNull)`, Description: `Returns "null"`},
				{Code: `_json_encode(JBool(true))`, Description: `Returns "true"`},
				{Code: `_json_encode(JNumber(42.0))`, Description: `Returns "42"`},
				{Code: `_json_encode(JString("hello"))`, Description: `Returns "\"hello\""`},
				{Code: `_json_encode(JArray([JNumber(1.0), JNumber(2.0)]))`, Description: `Returns "[1,2]"`},
				{Code: `_json_encode(JObject([{key: "name", value: JString("Alice")}]))`, Description: `Returns "{\"name\":\"Alice\"}"`},
			},
			SeeAlso:   []string{"std/json.decode", "std/json module"},
			Since:     "v0.3.23",
			Stability: StabilityStable,
			Tags:      []string{"json", "encoding", "serialization", "data"},
			Category:  "json",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _json_encode: %v", err))
	}
}

func makeJSONEncodeType() types.Type {
	T := types.NewBuilder()
	// Type signature: Json -> string
	jsonType := T.Con("Json")
	return T.Func(jsonType).Returns(T.String()).Build()
}

// GetJSONEncodeImpl exports the implementation for legacy registry integration
func GetJSONEncodeImpl() EffectImpl {
	return jsonEncodeImpl
}

func jsonEncodeImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Extract Json ADT value
	jsonVal, ok := args[0].(*eval.TaggedValue)
	if !ok {
		return nil, fmt.Errorf("_json_encode: expected Json, got %T", args[0])
	}

	// Encode to string
	var buf strings.Builder
	if err := encodeValue(jsonVal, &buf); err != nil {
		return nil, err
	}

	// Return string value
	return &eval.StringValue{Value: buf.String()}, nil
}

// encodeValue recursively encodes a Json ADT value to JSON string
func encodeValue(val eval.Value, buf *strings.Builder) error {
	tagged, ok := val.(*eval.TaggedValue)
	if !ok {
		return fmt.Errorf("expected TaggedValue, got %T", val)
	}

	// Verify it's a Json ADT
	if tagged.TypeName != "Json" {
		return fmt.Errorf("expected Json ADT, got %s", tagged.TypeName)
	}

	switch tagged.CtorName {
	case "JNull":
		buf.WriteString("null")
		return nil

	case "JBool":
		if len(tagged.Fields) != 1 {
			return fmt.Errorf("JBool expected 1 field, got %d", len(tagged.Fields))
		}
		boolVal, ok := tagged.Fields[0].(*eval.BoolValue)
		if !ok {
			return fmt.Errorf("JBool field expected BoolValue, got %T", tagged.Fields[0])
		}
		if boolVal.Value {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
		return nil

	case "JNumber":
		if len(tagged.Fields) != 1 {
			return fmt.Errorf("JNumber expected 1 field, got %d", len(tagged.Fields))
		}
		// Accept both FloatValue and IntValue — JS numbers are always float64,
		// so round-tripping through WASM bridge (jsToAILANGValue) converts
		// whole numbers to IntValue. Both must be handled here.
		switch numVal := tagged.Fields[0].(type) {
		case *eval.FloatValue:
			buf.WriteString(formatNumber(numVal.Value))
		case *eval.IntValue:
			buf.WriteString(fmt.Sprintf("%d", numVal.Value))
		default:
			return fmt.Errorf("JNumber field expected numeric value, got %T", tagged.Fields[0])
		}
		return nil

	case "JString":
		if len(tagged.Fields) != 1 {
			return fmt.Errorf("JString expected 1 field, got %d", len(tagged.Fields))
		}
		strVal, ok := tagged.Fields[0].(*eval.StringValue)
		if !ok {
			return fmt.Errorf("JString field expected StringValue, got %T", tagged.Fields[0])
		}
		buf.WriteByte('"')
		escapeString(strVal.Value, buf)
		buf.WriteByte('"')
		return nil

	case "JArray":
		if len(tagged.Fields) != 1 {
			return fmt.Errorf("JArray expected 1 field, got %d", len(tagged.Fields))
		}
		listVal, ok := tagged.Fields[0].(*eval.ListValue)
		if !ok {
			return fmt.Errorf("JArray field expected ListValue, got %T", tagged.Fields[0])
		}
		buf.WriteByte('[')
		for i, elem := range listVal.Elements {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := encodeValue(elem, buf); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
		return nil

	case "JObject":
		if len(tagged.Fields) != 1 {
			return fmt.Errorf("JObject expected 1 field, got %d", len(tagged.Fields))
		}
		listVal, ok := tagged.Fields[0].(*eval.ListValue)
		if !ok {
			return fmt.Errorf("JObject field expected ListValue, got %T", tagged.Fields[0])
		}
		buf.WriteByte('{')
		for i, kvElem := range listVal.Elements {
			// Each element should be a record with {key: string, value: Json}
			kvRecord, ok := kvElem.(*eval.RecordValue)
			if !ok {
				return fmt.Errorf("JObject key-value pair expected RecordValue, got %T", kvElem)
			}

			// Extract key field
			keyVal, ok := kvRecord.Fields["key"]
			if !ok {
				return fmt.Errorf("JObject record missing 'key' field")
			}
			keyStr, ok := keyVal.(*eval.StringValue)
			if !ok {
				return fmt.Errorf("JObject key expected StringValue, got %T", keyVal)
			}

			// Extract value field
			valueVal, ok := kvRecord.Fields["value"]
			if !ok {
				return fmt.Errorf("JObject record missing 'value' field")
			}

			if i > 0 {
				buf.WriteByte(',')
			}

			// Encode key
			buf.WriteByte('"')
			escapeString(keyStr.Value, buf)
			buf.WriteByte('"')
			buf.WriteByte(':')

			// Encode value (recursive)
			if err := encodeValue(valueVal, buf); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
		return nil

	default:
		return fmt.Errorf("unknown Json constructor: %s", tagged.CtorName)
	}
}

// formatNumber formats a float for JSON, removing unnecessary decimal points
func formatNumber(f float64) string {
	// Handle special cases
	if math.IsNaN(f) || math.IsInf(f, 0) {
		// JSON doesn't support NaN or Infinity - this shouldn't happen with well-formed Json ADT
		return "null"
	}

	// Check if the number is effectively an integer
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}

	// Format as float with minimal precision
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// escapeString escapes a string for JSON according to RFC 8259
func escapeString(s string, buf *strings.Builder) {
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			// Control characters (U+0000 to U+001F) must be escaped
			if r < 0x20 {
				buf.WriteString(fmt.Sprintf(`\u%04x`, r))
			} else {
				// Normal characters pass through (UTF-8)
				buf.WriteRune(r)
			}
		}
	}
}
