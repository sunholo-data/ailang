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
	// Extract string argument
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_json_decode: expected string, got %T", args[0])
	}

	// Build JSON value from string
	builder := newJSONBuilder(strVal.Value)
	jsonVal, err := builder.build()
	if err != nil {
		// Return Err(string)
		return &eval.TaggedValue{
			ModulePath: "std/result",
			TypeName:   "Result",
			CtorName:   "Err",
			Fields:     []eval.Value{&eval.StringValue{Value: err.Error()}},
		}, nil
	}

	// Return Ok(Json)
	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Ok",
		Fields:     []eval.Value{jsonVal},
	}, nil
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
