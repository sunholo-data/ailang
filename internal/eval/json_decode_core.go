package eval

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// The ONE std/json decoder (M-V1-STDLIB-SURFACE-POLISH B1/B2/B3).
//
// There used to be three hand-copied decoders — the registry builtin
// (internal/builtins), the bytecode VM (internal/vm) and a legacy one here —
// and they had drifted: an empty-string object key was dropped by one engine
// and kept by the other, and none rejected trailing data. They now share this
// token walk; each engine only supplies a JSONSink that builds its own value
// representation, so the number rule, the key rule and the end-of-input rule
// cannot diverge again.

// JSONSink builds one engine's representation of the std/json Json ADT.
type JSONSink[V any] interface {
	Null() V
	Bool(b bool) V
	Int(n int64) V     // JInt
	Float(f float64) V // JNumber
	String(s string) V
	Array(elems []V) V
	Object(keys []string, vals []V) V
}

// JSONNumber classifies a grammar-validated JSON number token (the B3 rule):
//
//   - integer syntax (no '.', no exponent) that fits int64, except "-0" → JInt
//   - everything else (fractions, exponents, "-0", integers outside int64)
//     → JNumber with the nearest float64
//   - a literal outside float64 range (e.g. 1e400) → error, never ±Inf
func JSONNumber(n json.Number) (isInt bool, i int64, f float64, err error) {
	s := string(n)
	if s != "-0" && !strings.ContainsAny(s, ".eE") {
		if v, perr := strconv.ParseInt(s, 10, 64); perr == nil {
			return true, v, 0, nil
		}
	}
	f, err = ParseJSONNumber(n)
	return false, 0, f, err
}

// jsonFrame is one open array or object.
type jsonFrame[V any] struct {
	isObject  bool
	expectKey bool // objects only: the next string token is a key
	keys      []string
	vals      []V
}

// DecodeJSON parses input as exactly one JSON value, optionally surrounded by
// whitespace, and builds it with sink.
//
// Rules shared by both engines:
//   - B1: anything but whitespace after the value is
//     "decode: trailing data at offset N" (N = byte offset of the first
//     non-whitespace byte after the value);
//   - B2: an object key is a key by position (an explicit expecting-key state),
//     so "" is an ordinary key; keys keep source order, duplicates included;
//   - B3: numbers follow JSONNumber.
func DecodeJSON[V any](input string, sink JSONSink[V]) (V, error) {
	var zero V
	dec := json.NewDecoder(strings.NewReader(input))
	dec.UseNumber()

	var stack []*jsonFrame[V]
	var result V
	done := false

	// emit places a finished value into the enclosing frame, or finishes the
	// top-level value.
	emit := func(v V) {
		if len(stack) == 0 {
			result, done = v, true
			return
		}
		top := stack[len(stack)-1]
		top.vals = append(top.vals, v)
		if top.isObject {
			top.expectKey = true
		}
	}

	for !done {
		tok, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if len(stack) == 0 {
					return zero, fmt.Errorf("no JSON value found")
				}
				return zero, fmt.Errorf("unexpected end of input")
			}
			return zero, fmt.Errorf("invalid json: %s", err.Error())
		}

		// A string token in key position is the key, whatever its content.
		if len(stack) > 0 {
			if top := stack[len(stack)-1]; top.isObject && top.expectKey {
				if key, ok := tok.(string); ok {
					top.keys = append(top.keys, key)
					top.expectKey = false
					continue
				}
			}
		}

		switch v := tok.(type) {
		case json.Delim:
			switch v {
			case '{':
				stack = append(stack, &jsonFrame[V]{isObject: true, expectKey: true})
			case '[':
				stack = append(stack, &jsonFrame[V]{})
			case '}', ']':
				// encoding/json has already checked bracket balance and order.
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if top.isObject {
					emit(sink.Object(top.keys, top.vals))
				} else {
					emit(sink.Array(top.vals))
				}
			}
		case string:
			emit(sink.String(v))
		case json.Number:
			isInt, i, f, nerr := JSONNumber(v)
			if nerr != nil {
				return zero, nerr
			}
			if isInt {
				emit(sink.Int(i))
			} else {
				emit(sink.Float(f))
			}
		case bool:
			emit(sink.Bool(v))
		case nil:
			emit(sink.Null())
		}
	}

	// B1: only whitespace may follow the value.
	for off := int(dec.InputOffset()); off < len(input); off++ {
		switch input[off] {
		case ' ', '\t', '\n', '\r':
			continue
		}
		return zero, fmt.Errorf("decode: trailing data at offset %d", off)
	}
	return result, nil
}

// TaggedJSONSink builds the evaluator's TaggedValue form of the Json ADT. The
// registry builtin and the legacy evaluator builtin both decode through it.
type TaggedJSONSink struct{}

var _ JSONSink[Value] = TaggedJSONSink{}

func jsonTagged(ctor string, fields ...Value) Value {
	if fields == nil {
		fields = []Value{}
	}
	return &TaggedValue{ModulePath: "std/json", TypeName: "Json", CtorName: ctor, Fields: fields}
}

// Null builds JNull.
func (TaggedJSONSink) Null() Value { return jsonTagged("JNull") }

// Bool builds JBool(b).
func (TaggedJSONSink) Bool(b bool) Value { return jsonTagged("JBool", &BoolValue{Value: b}) }

// Int builds JInt(n).
func (TaggedJSONSink) Int(n int64) Value { return jsonTagged("JInt", &IntValue{Value: int(n)}) }

// Float builds JNumber(f).
func (TaggedJSONSink) Float(f float64) Value { return jsonTagged("JNumber", &FloatValue{Value: f}) }

// String builds JString(s).
func (TaggedJSONSink) String(s string) Value { return jsonTagged("JString", &StringValue{Value: s}) }

// Array builds JArray(elems).
func (TaggedJSONSink) Array(elems []Value) Value {
	if elems == nil {
		elems = []Value{}
	}
	return jsonTagged("JArray", &ListValue{Elements: elems})
}

// Object builds JObject([{key, value}]) in source order.
func (TaggedJSONSink) Object(keys []string, vals []Value) Value {
	pairs := make([]Value, len(keys))
	for i, k := range keys {
		pairs[i] = &RecordValue{Fields: map[string]Value{
			"key":   &StringValue{Value: k},
			"value": vals[i],
		}}
	}
	return jsonTagged("JObject", &ListValue{Elements: pairs})
}

// DecodeJSONString decodes s into the evaluator's Json ADT as
// Result[Json, string]: Ok(json), or Err(message) for any decode failure.
func DecodeJSONString(s string) Value {
	v, err := DecodeJSON[Value](s, TaggedJSONSink{})
	if err != nil {
		return legacyWrapErr(err.Error())
	}
	return legacyWrapOk(v)
}
