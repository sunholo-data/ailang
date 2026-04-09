package vm

// VM-native JSON builtins: encode, decode, repair.
//
// These builtins operate on bytecode.Value directly, avoiding the eval bridge.
// The Json ADT tag ordering must match std/json.ail:
//
//   export type Json =
//     | JNull           → tag 0
//     | JBool(bool)     → tag 1
//     | JNumber(float)  → tag 2
//     | JString(string) → tag 3
//     | JArray(List[Json])                      → tag 4
//     | JObject(List[{key: string, value: Json}]) → tag 5
//
// Result ADT (std/result.ail): Ok=0, Err=1.

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/sunholo/ailang/internal/bytecode"
)

// Json ADT variant tags (must match std/json.ail declaration order).
const (
	jsonTagJNull   = 0
	jsonTagJBool   = 1
	jsonTagJNumber = 2
	jsonTagJString = 3
	jsonTagJArray  = 4
	jsonTagJObject = 5
)

// Result ADT variant tags (must match std/result.ail declaration order).
const (
	resultTagOk  = 0
	resultTagErr = 1
)

// --- json_encode: Json -> string ---------------------------------------------

func builtinJsonEncode(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__json_encode: expected 1 arg, got %d", len(args))
	}
	var buf strings.Builder
	if err := vmEncodeJson(args[0], &buf); err != nil {
		return bytecode.Value{}, err
	}
	return bytecode.NewString(buf.String()), nil
}

func vmEncodeJson(v bytecode.Value, buf *strings.Builder) error {
	if v.Tag != bytecode.TagADT {
		return fmt.Errorf("__json_encode: expected Json ADT, got tag %d", v.Tag)
	}
	adt := v.AsADT()

	switch adt.Tag {
	case jsonTagJNull:
		buf.WriteString("null")

	case jsonTagJBool:
		if len(adt.Fields) != 1 {
			return fmt.Errorf("JBool: expected 1 field, got %d", len(adt.Fields))
		}
		if adt.Fields[0].Bool {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}

	case jsonTagJNumber:
		if len(adt.Fields) != 1 {
			return fmt.Errorf("JNumber: expected 1 field, got %d", len(adt.Fields))
		}
		f := adt.Fields[0]
		switch f.Tag {
		case bytecode.TagFloat:
			buf.WriteString(vmFormatNumber(f.Flt))
		case bytecode.TagInt:
			buf.WriteString(strconv.FormatInt(f.Int, 10))
		default:
			return fmt.Errorf("JNumber: expected numeric, got tag %d", f.Tag)
		}

	case jsonTagJString:
		if len(adt.Fields) != 1 {
			return fmt.Errorf("JString: expected 1 field, got %d", len(adt.Fields))
		}
		buf.WriteByte('"')
		vmEscapeString(adt.Fields[0].AsString(), buf)
		buf.WriteByte('"')

	case jsonTagJArray:
		if len(adt.Fields) != 1 {
			return fmt.Errorf("JArray: expected 1 field, got %d", len(adt.Fields))
		}
		elems := adt.Fields[0].AsList()
		buf.WriteByte('[')
		for i, elem := range elems {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := vmEncodeJson(elem, buf); err != nil {
				return err
			}
		}
		buf.WriteByte(']')

	case jsonTagJObject:
		if len(adt.Fields) != 1 {
			return fmt.Errorf("JObject: expected 1 field, got %d", len(adt.Fields))
		}
		kvPairs := adt.Fields[0].AsList()
		buf.WriteByte('{')
		for i, kv := range kvPairs {
			if kv.Tag != bytecode.TagRecord {
				return fmt.Errorf("JObject entry: expected record, got tag %d", kv.Tag)
			}
			fields := kv.AsRecord()
			// Records are sorted alphabetically: key=0, value=1
			if len(fields) != 2 || fields[0].Name != "key" || fields[1].Name != "value" {
				return fmt.Errorf("JObject entry: expected {key, value} record")
			}
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteByte('"')
			vmEscapeString(fields[0].Value.AsString(), buf)
			buf.WriteByte('"')
			buf.WriteByte(':')
			if err := vmEncodeJson(fields[1].Value, buf); err != nil {
				return err
			}
		}
		buf.WriteByte('}')

	default:
		return fmt.Errorf("__json_encode: unknown Json tag %d", adt.Tag)
	}
	return nil
}

func vmFormatNumber(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "null"
	}
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func vmEscapeString(s string, buf *strings.Builder) {
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
			if r < 0x20 {
				buf.WriteString(fmt.Sprintf(`\u%04x`, r))
			} else {
				buf.WriteRune(r)
			}
		}
	}
}

// --- json_decode: string -> Result[Json, string] -----------------------------

func builtinJsonDecode(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__json_decode: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__json_decode: expected string, got tag %d", args[0].Tag)
	}
	input := args[0].AsString()

	b := vmJSONBuilder{decoder: json.NewDecoder(strings.NewReader(input))}
	b.decoder.UseNumber()

	jsonVal, err := b.build()
	if err != nil {
		return vmResultErr(err.Error()), nil
	}
	return vmResultOk(jsonVal), nil
}

type vmJSONFrameType int

const (
	vmFrameArray vmJSONFrameType = iota
	vmFrameObject
)

type vmJSONFrame struct {
	typ     vmJSONFrameType
	values  []bytecode.Value // arrays
	kvPairs []bytecode.Value // objects: list of {key, value} records
	lastKey string
}

type vmJSONBuilder struct {
	decoder *json.Decoder
	stack   []vmJSONFrame
	result  *bytecode.Value
}

func (b *vmJSONBuilder) build() (bytecode.Value, error) {
	for {
		tok, err := b.decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return bytecode.Value{}, fmt.Errorf("invalid json: %s", err.Error())
		}

		switch v := tok.(type) {
		case json.Delim:
			switch v {
			case '{':
				b.pushObject()
			case '}':
				obj := b.popObject()
				b.addValue(obj)
			case '[':
				b.pushArray()
			case ']':
				arr := b.popArray()
				b.addValue(arr)
			}
		case string:
			if b.inObject() && b.expectingKey() {
				b.setKey(v)
			} else {
				b.addValue(vmMakeJString(v))
			}
		case json.Number:
			b.addValue(vmMakeJNumber(v))
		case bool:
			b.addValue(vmMakeJBool(v))
		case nil:
			b.addValue(vmMakeJNull())
		}

		if len(b.stack) == 0 && b.result != nil {
			break
		}
	}

	if len(b.stack) != 0 {
		return bytecode.Value{}, fmt.Errorf("unexpected end of input")
	}
	if b.result == nil {
		return bytecode.Value{}, fmt.Errorf("no JSON value found")
	}
	return *b.result, nil
}

func (b *vmJSONBuilder) pushObject() {
	b.stack = append(b.stack, vmJSONFrame{typ: vmFrameObject})
}

func (b *vmJSONBuilder) popObject() bytecode.Value {
	frame := b.stack[len(b.stack)-1]
	b.stack = b.stack[:len(b.stack)-1]
	return bytecode.NewADT(jsonTagJObject, []bytecode.Value{bytecode.NewList(frame.kvPairs)})
}

func (b *vmJSONBuilder) pushArray() {
	b.stack = append(b.stack, vmJSONFrame{typ: vmFrameArray})
}

func (b *vmJSONBuilder) popArray() bytecode.Value {
	frame := b.stack[len(b.stack)-1]
	b.stack = b.stack[:len(b.stack)-1]
	return bytecode.NewADT(jsonTagJArray, []bytecode.Value{bytecode.NewList(frame.values)})
}

func (b *vmJSONBuilder) addValue(val bytecode.Value) {
	if len(b.stack) == 0 {
		b.result = &val
		return
	}
	frame := &b.stack[len(b.stack)-1]
	if frame.typ == vmFrameArray {
		frame.values = append(frame.values, val)
	} else {
		kv := bytecode.NewRecord([]bytecode.RecordField{
			{Name: "key", Value: bytecode.NewString(frame.lastKey)},
			{Name: "value", Value: val},
		})
		frame.kvPairs = append(frame.kvPairs, kv)
		frame.lastKey = ""
	}
}

func (b *vmJSONBuilder) setKey(key string) {
	if len(b.stack) > 0 {
		b.stack[len(b.stack)-1].lastKey = key
	}
}

func (b *vmJSONBuilder) inObject() bool {
	return len(b.stack) > 0 && b.stack[len(b.stack)-1].typ == vmFrameObject
}

func (b *vmJSONBuilder) expectingKey() bool {
	if len(b.stack) == 0 {
		return false
	}
	f := &b.stack[len(b.stack)-1]
	return f.typ == vmFrameObject && f.lastKey == ""
}

// Json ADT constructors for bytecode values.

func vmMakeJNull() bytecode.Value {
	return bytecode.NewADT(jsonTagJNull, nil)
}

func vmMakeJBool(b bool) bytecode.Value {
	return bytecode.NewADT(jsonTagJBool, []bytecode.Value{bytecode.NewBool(b)})
}

func vmMakeJNumber(n json.Number) bytecode.Value {
	str := string(n)
	if strings.ContainsAny(str, ".eE") {
		f, _ := n.Float64()
		return bytecode.NewADT(jsonTagJNumber, []bytecode.Value{bytecode.NewFloat(f)})
	}
	i, _ := n.Int64()
	return bytecode.NewADT(jsonTagJNumber, []bytecode.Value{bytecode.NewFloat(float64(i))})
}

func vmMakeJString(s string) bytecode.Value {
	return bytecode.NewADT(jsonTagJString, []bytecode.Value{bytecode.NewString(s)})
}

// Result ADT constructors.

func vmResultOk(v bytecode.Value) bytecode.Value {
	return bytecode.NewADT(resultTagOk, []bytecode.Value{v})
}

func vmResultErr(msg string) bytecode.Value {
	return bytecode.NewADT(resultTagErr, []bytecode.Value{bytecode.NewString(msg)})
}

// --- json_repair: string -> Result[string, string] ---------------------------

func builtinJsonRepair(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__json_repair: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__json_repair: expected string, got tag %d", args[0].Tag)
	}
	input := args[0].AsString()

	repaired, err := vmRepairJSON(input)
	if err != nil {
		return vmResultErr(err.Error()), nil
	}
	return vmResultOk(bytecode.NewString(repaired)), nil
}

// vmRepairJSON attempts to fix common JSON truncation issues.
func vmRepairJSON(input string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", fmt.Errorf("empty input")
	}

	// Try parsing as-is first.
	if json.Valid([]byte(s)) {
		return s, nil
	}

	// Scan tracking structural state.
	var (
		buf       strings.Builder
		stack     []byte
		inString  bool
		escaped   bool
		lastNonWS byte
	)
	buf.Grow(len(s) + 16)

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inString {
			if escaped {
				buf.WriteByte(ch)
				escaped = false
				continue
			}
			if ch == '\\' {
				buf.WriteByte(ch)
				escaped = true
				continue
			}
			if ch == '"' {
				buf.WriteByte(ch)
				inString = false
				lastNonWS = ch
				continue
			}
			if ch < 0x20 && ch != '\t' && ch != '\n' && ch != '\r' {
				continue
			}
			buf.WriteByte(ch)
			continue
		}

		switch ch {
		case '"':
			buf.WriteByte(ch)
			inString = true
			lastNonWS = ch
		case '{':
			buf.WriteByte(ch)
			stack = append(stack, '{')
			lastNonWS = ch
		case '}':
			vmTrimTrailingComma(&buf, &lastNonWS)
			buf.WriteByte(ch)
			if len(stack) > 0 && stack[len(stack)-1] == '{' {
				stack = stack[:len(stack)-1]
			}
			lastNonWS = ch
		case '[':
			buf.WriteByte(ch)
			stack = append(stack, '[')
			lastNonWS = ch
		case ']':
			vmTrimTrailingComma(&buf, &lastNonWS)
			buf.WriteByte(ch)
			if len(stack) > 0 && stack[len(stack)-1] == '[' {
				stack = stack[:len(stack)-1]
			}
			lastNonWS = ch
		case ' ', '\t', '\n', '\r':
			buf.WriteByte(ch)
		default:
			buf.WriteByte(ch)
			lastNonWS = ch
		}
	}

	result := buf.String()

	if inString {
		if escaped {
			result = result[:len(result)-1]
		}
		result += `"`
	}

	result = vmCompleteTruncatedKeyword(result)
	result = strings.TrimRight(result, " \t\n\r,")

	for i := len(stack) - 1; i >= 0; i-- {
		switch stack[i] {
		case '{':
			trimmed := strings.TrimRight(result, " \t\n\r")
			if len(trimmed) > 0 && trimmed[len(trimmed)-1] == ':' {
				result = trimmed + "null"
			}
			result = strings.TrimRight(result, " \t\n\r,") + "}"
		case '[':
			result = strings.TrimRight(result, " \t\n\r,") + "]"
		}
	}

	if !json.Valid([]byte(result)) {
		return result, fmt.Errorf("repair incomplete: structural damage too severe")
	}
	return result, nil
}

func vmTrimTrailingComma(buf *strings.Builder, lastNonWS *byte) {
	if *lastNonWS == ',' {
		s := buf.String()
		idx := strings.LastIndex(s, ",")
		if idx >= 0 {
			buf.Reset()
			buf.WriteString(s[:idx])
			buf.WriteString(s[idx+1:])
			trimmed := strings.TrimRight(buf.String(), " \t\n\r")
			if len(trimmed) > 0 {
				*lastNonWS = trimmed[len(trimmed)-1]
			}
		}
	}
}

func vmCompleteTruncatedKeyword(s string) string {
	trimmed := strings.TrimRight(s, " \t\n\r")
	if len(trimmed) == 0 {
		return s
	}
	keywords := []string{"true", "false", "null"}
	for _, kw := range keywords {
		for prefixLen := len(kw) - 1; prefixLen >= 2; prefixLen-- {
			prefix := kw[:prefixLen]
			if strings.HasSuffix(trimmed, prefix) {
				beforeIdx := len(trimmed) - prefixLen
				if beforeIdx == 0 || vmIsStructuralBefore(trimmed[beforeIdx-1]) {
					return trimmed[:len(trimmed)-prefixLen] + kw
				}
			}
		}
	}
	return s
}

func vmIsStructuralBefore(ch byte) bool {
	return ch == ':' || ch == ',' || ch == '[' || ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}
