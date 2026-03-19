package builtins

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// JSON streaming builder - converts encoding/json token stream to AILANG Json ADT

// Builtin registration

func init() {
	registerJSONDecode()
}

func registerJSONDecode() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/json",
		Name:    "_json_decode",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeJSONDecodeType,
		Impl:    jsonDecodeImpl,
		Metadata: &BuiltinMetadata{
			Description: "Parse a JSON string into a Json ADT value",
			LongDesc: `Parses a JSON string using Go's encoding/json package and converts it to AILANG's Json algebraic data type.
Supports all JSON types: objects, arrays, strings, numbers, booleans, and null.
Returns Result[Json, string] - Ok(json) on success, Err(message) on parse error.`,
			Params: []ParamDoc{
				{Name: "input", Description: "JSON string to parse"},
			},
			Returns: "Result[Json, string] - Ok(Json) on success, Err(error message) on invalid JSON",
			Examples: []Example{
				{Code: `_json_decode("{\"name\":\"Alice\",\"age\":30}")`, Description: "Returns Ok(JObject(...))"},
				{Code: `_json_decode("[1,2,3]")`, Description: "Returns Ok(JArray([JNumber(1.0), JNumber(2.0), JNumber(3.0)]))"},
				{Code: `_json_decode("\"hello\"")`, Description: "Returns Ok(JString(\"hello\"))"},
				{Code: `_json_decode("42")`, Description: "Returns Ok(JNumber(42.0))"},
				{Code: `_json_decode("true")`, Description: "Returns Ok(JBool(true))"},
				{Code: `_json_decode("null")`, Description: "Returns Ok(JNull)"},
				{Code: `_json_decode("{invalid}")`, Description: "Returns Err(\"invalid json: ...\")"},
			},
			SeeAlso:   []string{"std/json.encode", "std/json module"},
			Since:     "v0.2.0",
			Stability: StabilityStable,
			Tags:      []string{"json", "parsing", "deserialization", "data", "result"},
			Category:  "json",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _json_decode: %v", err))
	}
}

func makeJSONDecodeType() types.Type {
	T := types.NewBuilder()
	// Type signature: string -> Result[Json, string]
	jsonType := T.Con("Json")
	resultType := T.App("Result", jsonType, T.String())
	return T.Func(T.String()).Returns(resultType).Build()
}

// GetJSONDecodeImpl exports the implementation for legacy registry integration
func GetJSONDecodeImpl() EffectImpl {
	return jsonDecodeImpl
}

func jsonDecodeImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Handle already-structured AILANG values by converting to Json ADT directly
	switch v := args[0].(type) {
	case *eval.StringValue:
		// Standard path: parse JSON string
		return jsonDecodeString(v)

	case *eval.RecordValue:
		// Pre-parsed data (e.g., from serve-api HTTP response) → convert to JObject
		jsonVal := valueToJSON(v)
		return wrapOk(jsonVal), nil

	case *eval.ListValue:
		// Pre-parsed array → convert to JArray
		jsonVal := valueToJSON(v)
		return wrapOk(jsonVal), nil

	case *eval.TaggedValue:
		// Try to unwrap Result/Option wrappers containing a string
		if strVal, ok := unwrapToString(v); ok {
			return jsonDecodeString(strVal)
		}
		// If the TaggedValue is a Json ADT, return it directly wrapped in Ok
		if v.TypeName == "Json" {
			return wrapOk(v), nil
		}
		return nil, fmt.Errorf("_json_decode: expected string, got %s(%s) — unwrap the %s before calling json_decode",
			v.TypeName, v.CtorName, v.TypeName)

	case *eval.IntValue, *eval.FloatValue, *eval.BoolValue:
		// Primitive values → convert to Json ADT directly
		jsonVal := valueToJSON(v)
		return wrapOk(jsonVal), nil

	default:
		return nil, fmt.Errorf("_json_decode: expected string, got %T", args[0])
	}
}

// jsonDecodeString parses a JSON string into the Json ADT, returning Result[Json, string].
func jsonDecodeString(strVal *eval.StringValue) (eval.Value, error) {
	builder := newJSONBuilder(strVal.Value)
	jsonVal, err := builder.build()
	if err != nil {
		return wrapErr(err.Error()), nil
	}
	return wrapOk(jsonVal), nil
}

// valueToJSON converts an arbitrary AILANG value to the Json ADT.
func valueToJSON(v eval.Value) eval.Value {
	switch val := v.(type) {
	case *eval.StringValue:
		return makeJString(val.Value)
	case *eval.IntValue:
		return &eval.TaggedValue{
			ModulePath: "std/json",
			TypeName:   "Json",
			CtorName:   "JNumber",
			Fields:     []eval.Value{&eval.FloatValue{Value: float64(val.Value)}},
		}
	case *eval.FloatValue:
		return &eval.TaggedValue{
			ModulePath: "std/json",
			TypeName:   "Json",
			CtorName:   "JNumber",
			Fields:     []eval.Value{&eval.FloatValue{Value: val.Value}},
		}
	case *eval.BoolValue:
		return makeJBool(val.Value)
	case *eval.UnitValue:
		return makeJNull()
	case *eval.ListValue:
		elements := make([]eval.Value, len(val.Elements))
		for i, elem := range val.Elements {
			elements[i] = valueToJSON(elem)
		}
		return &eval.TaggedValue{
			ModulePath: "std/json",
			TypeName:   "Json",
			CtorName:   "JArray",
			Fields:     []eval.Value{&eval.ListValue{Elements: elements}},
		}
	case *eval.RecordValue:
		kvPairs := make([]eval.Value, 0, len(val.Fields))
		for key, fieldVal := range val.Fields {
			kvPairs = append(kvPairs, &eval.RecordValue{
				Fields: map[string]eval.Value{
					"key":   &eval.StringValue{Value: key},
					"value": valueToJSON(fieldVal),
				},
			})
		}
		return &eval.TaggedValue{
			ModulePath: "std/json",
			TypeName:   "Json",
			CtorName:   "JObject",
			Fields:     []eval.Value{&eval.ListValue{Elements: kvPairs}},
		}
	case *eval.TaggedValue:
		// If it's already a Json ADT value, return as-is
		if val.TypeName == "Json" {
			return val
		}
		// Otherwise represent as a JSON object with __type and __tag
		fields := make([]eval.Value, 0, 2+len(val.Fields))
		fields = append(fields, &eval.RecordValue{
			Fields: map[string]eval.Value{
				"key":   &eval.StringValue{Value: "__type"},
				"value": makeJString(val.TypeName),
			},
		})
		fields = append(fields, &eval.RecordValue{
			Fields: map[string]eval.Value{
				"key":   &eval.StringValue{Value: "__tag"},
				"value": makeJString(val.CtorName),
			},
		})
		if len(val.Fields) > 0 {
			fieldElements := make([]eval.Value, len(val.Fields))
			for i, f := range val.Fields {
				fieldElements[i] = valueToJSON(f)
			}
			fields = append(fields, &eval.RecordValue{
				Fields: map[string]eval.Value{
					"key": &eval.StringValue{Value: "fields"},
					"value": &eval.TaggedValue{
						ModulePath: "std/json",
						TypeName:   "Json",
						CtorName:   "JArray",
						Fields:     []eval.Value{&eval.ListValue{Elements: fieldElements}},
					},
				},
			})
		}
		return &eval.TaggedValue{
			ModulePath: "std/json",
			TypeName:   "Json",
			CtorName:   "JObject",
			Fields:     []eval.Value{&eval.ListValue{Elements: fields}},
		}
	default:
		// Fallback: represent as JString of the value's string representation
		return makeJString(fmt.Sprintf("%v", v))
	}
}

// wrapOk wraps a value in Ok(value) for Result[Json, string].
func wrapOk(v eval.Value) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Ok",
		Fields:     []eval.Value{v},
	}
}

// wrapErr wraps a string in Err(string) for Result[Json, string].
func wrapErr(msg string) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Err",
		Fields:     []eval.Value{&eval.StringValue{Value: msg}},
	}
}

// unwrapToString extracts a string from common TaggedValue wrappers.
// Handles Ok(string), Some(string), and single-field constructors wrapping a string.
func unwrapToString(tv *eval.TaggedValue) (*eval.StringValue, bool) {
	if len(tv.Fields) != 1 {
		return nil, false
	}
	// Direct string field: Ok("..."), Some("..."), etc.
	if sv, ok := tv.Fields[0].(*eval.StringValue); ok {
		return sv, true
	}
	// Nested wrapper: Ok(Some("...")) — one level of recursion
	if inner, ok := tv.Fields[0].(*eval.TaggedValue); ok {
		return unwrapToString(inner)
	}
	return nil, false
}

// Streaming builder implementation

type frameType int

const (
	frameArray frameType = iota
	frameObject
)

type buildFrame struct {
	typ     frameType
	values  []eval.Value // For arrays
	kvPairs []eval.Value // For objects: list of {key: string, value: Json} records
	lastKey string       // For objects: current key waiting for value
}

type JSONBuilder struct {
	decoder *json.Decoder
	stack   []buildFrame
	result  eval.Value
}

func newJSONBuilder(input string) *JSONBuilder {
	dec := json.NewDecoder(strings.NewReader(input))
	dec.UseNumber() // Preserve number precision, convert later
	return &JSONBuilder{
		decoder: dec,
		stack:   []buildFrame{},
	}
}

func (b *JSONBuilder) build() (eval.Value, error) {
	// Process all tokens
	for {
		tok, err := b.decoder.Token()
		if err == nil {
			// Process the token
			switch v := tok.(type) {
			case json.Delim:
				switch v {
				case '{':
					b.pushObject()
				case '}':
					obj, err := b.popObject()
					if err != nil {
						return nil, err
					}
					b.addValue(obj)
				case '[':
					b.pushArray()
				case ']':
					arr, err := b.popArray()
					if err != nil {
						return nil, err
					}
					b.addValue(arr)
				}
			case string:
				if b.inObject() && b.expectingKey() {
					b.setKey(v)
				} else {
					b.addValue(makeJString(v))
				}
			case json.Number:
				b.addValue(makeJNumber(v))
			case bool:
				b.addValue(makeJBool(v))
			case nil:
				b.addValue(makeJNull())
			}
		} else {
			// Check if we're done (EOF is expected)
			if err.Error() == "EOF" {
				break
			}
			return nil, b.normalizeError(err)
		}

		// Check if we've consumed all tokens for the top-level value
		if len(b.stack) == 0 && b.result != nil {
			break
		}
	}

	// Top-level value should be stored in result
	if len(b.stack) != 0 {
		return nil, fmt.Errorf("unexpected end of input")
	}

	if b.result == nil {
		return nil, fmt.Errorf("no JSON value found")
	}

	return b.result, nil
}

// Stack management

func (b *JSONBuilder) pushObject() {
	b.stack = append(b.stack, buildFrame{
		typ:     frameObject,
		kvPairs: []eval.Value{},
	})
}

func (b *JSONBuilder) popObject() (eval.Value, error) {
	if len(b.stack) == 0 {
		return nil, fmt.Errorf("unexpected '}'")
	}

	frame := b.stack[len(b.stack)-1]
	if frame.typ != frameObject {
		return nil, fmt.Errorf("mismatched brackets")
	}

	b.stack = b.stack[:len(b.stack)-1]

	// Build JObject constructor with list of {key, value} records
	listVal := &eval.ListValue{Elements: frame.kvPairs}
	return &eval.TaggedValue{
		ModulePath: "std/json",
		TypeName:   "Json",
		CtorName:   "JObject",
		Fields:     []eval.Value{listVal},
	}, nil
}

func (b *JSONBuilder) pushArray() {
	b.stack = append(b.stack, buildFrame{
		typ:    frameArray,
		values: []eval.Value{},
	})
}

func (b *JSONBuilder) popArray() (eval.Value, error) {
	if len(b.stack) == 0 {
		return nil, fmt.Errorf("unexpected ']'")
	}

	frame := b.stack[len(b.stack)-1]
	if frame.typ != frameArray {
		return nil, fmt.Errorf("mismatched brackets")
	}

	b.stack = b.stack[:len(b.stack)-1]

	// Build JArray constructor with list of values
	listVal := &eval.ListValue{Elements: frame.values}
	return &eval.TaggedValue{
		ModulePath: "std/json",
		TypeName:   "Json",
		CtorName:   "JArray",
		Fields:     []eval.Value{listVal},
	}, nil
}

func (b *JSONBuilder) addValue(val eval.Value) {
	if len(b.stack) == 0 {
		// Top-level value
		b.result = val
		return
	}

	frame := &b.stack[len(b.stack)-1]
	if frame.typ == frameArray {
		frame.values = append(frame.values, val)
	} else {
		// Object: this is the value for the current key
		if frame.lastKey == "" {
			// This shouldn't happen - encoding/json alternates keys and values
			return
		}

		// Create {key: string, value: Json} record
		kvRecord := &eval.RecordValue{
			Fields: map[string]eval.Value{
				"key":   &eval.StringValue{Value: frame.lastKey},
				"value": val,
			},
		}
		frame.kvPairs = append(frame.kvPairs, kvRecord)
		frame.lastKey = "" // Reset for next key
	}
}

func (b *JSONBuilder) setKey(key string) {
	if len(b.stack) == 0 {
		return
	}
	frame := &b.stack[len(b.stack)-1]
	if frame.typ == frameObject {
		frame.lastKey = key
	}
}

func (b *JSONBuilder) inObject() bool {
	if len(b.stack) == 0 {
		return false
	}
	return b.stack[len(b.stack)-1].typ == frameObject
}

func (b *JSONBuilder) expectingKey() bool {
	if len(b.stack) == 0 {
		return false
	}
	frame := &b.stack[len(b.stack)-1]
	return frame.typ == frameObject && frame.lastKey == ""
}

// Value constructors

func makeJString(s string) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/json",
		TypeName:   "Json",
		CtorName:   "JString",
		Fields:     []eval.Value{&eval.StringValue{Value: s}},
	}
}

func makeJNumber(n json.Number) eval.Value {
	str := string(n)

	// Check if float (contains . or e/E)
	if strings.ContainsAny(str, ".eE") {
		f, _ := n.Float64()
		return &eval.TaggedValue{
			ModulePath: "std/json",
			TypeName:   "Json",
			CtorName:   "JNumber",
			Fields:     []eval.Value{&eval.FloatValue{Value: f}},
		}
	}

	// Integer → convert to float for MVP simplicity
	i, _ := n.Int64()
	return &eval.TaggedValue{
		ModulePath: "std/json",
		TypeName:   "Json",
		CtorName:   "JNumber",
		Fields:     []eval.Value{&eval.FloatValue{Value: float64(i)}},
	}
}

func makeJBool(b bool) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/json",
		TypeName:   "Json",
		CtorName:   "JBool",
		Fields:     []eval.Value{&eval.BoolValue{Value: b}},
	}
}

func makeJNull() eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/json",
		TypeName:   "Json",
		CtorName:   "JNull",
		Fields:     []eval.Value{},
	}
}

// Error normalization

func (b *JSONBuilder) normalizeError(err error) error {
	// Normalize encoding/json errors to short, stable messages
	msg := err.Error()

	// Try to extract position info if available
	// encoding/json errors often include position like "json: error at offset X"
	// For now, keep it simple - just return a clean message
	if strings.Contains(msg, "unexpected") {
		return fmt.Errorf("invalid json: %s", msg)
	}

	return fmt.Errorf("invalid json: %s", msg)
}
