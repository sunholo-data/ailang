package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
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
Returns Result[Json, string] - Ok(json) on success, Err(message) on parse error.
An integer-syntax literal (no '.', no exponent) that fits int64 decodes to JInt(int), exactly; every other
number literal (fractions, exponents, "-0", integers outside int64) decodes to JNumber with its nearest float64.
A literal outside float64 range (e.g. 1e400) is an Err, never a silent Infinity. The input must hold exactly one
JSON value: anything but whitespace after it is Err("decode: trailing data at offset N"). An empty-string
object key is an ordinary key.`,
			Params: []ParamDoc{
				{Name: "input", Description: "JSON string to parse"},
			},
			Returns: "Result[Json, string] - Ok(Json) on success, Err(error message) on invalid JSON",
			Examples: []Example{
				{Code: `_json_decode("{\"name\":\"Alice\",\"age\":30}")`, Description: "Returns Ok(JObject(...))"},
				{Code: `_json_decode("[1,2,3]")`, Description: "Returns Ok(JArray([JInt(1), JInt(2), JInt(3)]))"},
				{Code: `_json_decode("\"hello\"")`, Description: "Returns Ok(JString(\"hello\"))"},
				{Code: `_json_decode("42")`, Description: "Returns Ok(JInt(42))"},
				{Code: `_json_decode("42.0")`, Description: "Returns Ok(JNumber(42.0))"},
				{Code: `_json_decode("1 2")`, Description: "Returns Err(\"decode: trailing data at offset 2\")"},
				{Code: `_json_decode("10000000000000000000")`, Description: "Returns Ok(JNumber(1.0e19)) exactly"},
				{Code: `_json_decode("1e400")`, Description: "Returns Err(\"json: number 1e400 out of float64 range\")"},
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
// It is the shared eval.DecodeJSON walk, so both engines agree on numbers
// (JInt vs JNumber), empty-string keys and trailing data.
func jsonDecodeString(strVal *eval.StringValue) (eval.Value, error) {
	return eval.DecodeJSONString(strVal.Value), nil
}

// valueToJSON converts an arbitrary AILANG value to the Json ADT.
func valueToJSON(v eval.Value) eval.Value {
	switch val := v.(type) {
	case *eval.StringValue:
		return makeJString(val.Value)
	case *eval.IntValue:
		// An AILANG int is exact: JInt, not a float that rounds above 2^53.
		return &eval.TaggedValue{
			ModulePath: "std/json",
			TypeName:   "Json",
			CtorName:   "JInt",
			Fields:     []eval.Value{&eval.IntValue{Value: val.Value}},
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

// Value constructors

func makeJString(s string) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/json",
		TypeName:   "Json",
		CtorName:   "JString",
		Fields:     []eval.Value{&eval.StringValue{Value: s}},
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
