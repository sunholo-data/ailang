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
//     | JInt(int)       → tag 6 (appended, so the older tags did not move)
//
// Result ADT (std/result.ail): Ok=0, Err=1.

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
)

// Json ADT variant tags (must match std/json.ail declaration order).
const (
	jsonTagJNull   = 0
	jsonTagJBool   = 1
	jsonTagJNumber = 2
	jsonTagJString = 3
	jsonTagJArray  = 4
	jsonTagJObject = 5
	jsonTagJInt    = 6
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

	case jsonTagJInt:
		if len(adt.Fields) != 1 {
			return fmt.Errorf("JInt: expected 1 field, got %d", len(adt.Fields))
		}
		if adt.Fields[0].Tag != bytecode.TagInt {
			return fmt.Errorf("JInt: expected int, got tag %d", adt.Fields[0].Tag)
		}
		buf.WriteString(strconv.FormatInt(adt.Fields[0].Int, 10))

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

// vmFormatNumber delegates to the canonical eval.FormatJSONNumber so the VM
// emits byte-identical number text to the interpreter.
func vmFormatNumber(f float64) string {
	return eval.FormatJSONNumber(f)
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
	jsonVal, err := eval.DecodeJSON[bytecode.Value](args[0].AsString(), vmJSONSink{})
	if err != nil {
		return vmResultErr(err.Error()), nil
	}
	return vmResultOk(jsonVal), nil
}

// vmJSONSink builds bytecode Json values for the shared eval.DecodeJSON walk,
// so the VM and the evaluator decode with one implementation (B1/B2/B3).
type vmJSONSink struct{}

var _ eval.JSONSink[bytecode.Value] = vmJSONSink{}

func (vmJSONSink) Null() bytecode.Value           { return vmMakeJNull() }
func (vmJSONSink) Bool(b bool) bytecode.Value     { return vmMakeJBool(b) }
func (vmJSONSink) String(s string) bytecode.Value { return vmMakeJString(s) }
func (vmJSONSink) Int(n int64) bytecode.Value {
	return bytecode.NewADT(jsonTagJInt, "JInt", []bytecode.Value{bytecode.NewInt(n)})
}
func (vmJSONSink) Float(f float64) bytecode.Value {
	return bytecode.NewADT(jsonTagJNumber, "JNumber", []bytecode.Value{bytecode.NewFloat(f)})
}
func (vmJSONSink) Array(elems []bytecode.Value) bytecode.Value {
	return bytecode.NewADT(jsonTagJArray, "JArray", []bytecode.Value{bytecode.NewList(elems)})
}

// Object builds JObject([{key, value}]) in source order.
func (vmJSONSink) Object(keys []string, vals []bytecode.Value) bytecode.Value {
	pairs := make([]bytecode.Value, len(keys))
	for i, k := range keys {
		pairs[i] = bytecode.NewRecord([]bytecode.RecordField{
			{Name: "key", Value: bytecode.NewString(k)},
			{Name: "value", Value: vals[i]},
		})
	}
	return bytecode.NewADT(jsonTagJObject, "JObject", []bytecode.Value{bytecode.NewList(pairs)})
}

// Json ADT constructors for bytecode values.

func vmMakeJNull() bytecode.Value {
	return bytecode.NewADT(jsonTagJNull, "JNull", nil)
}

func vmMakeJBool(b bool) bytecode.Value {
	return bytecode.NewADT(jsonTagJBool, "JBool", []bytecode.Value{bytecode.NewBool(b)})
}

func vmMakeJString(s string) bytecode.Value {
	return bytecode.NewADT(jsonTagJString, "JString", []bytecode.Value{bytecode.NewString(s)})
}

// Result ADT constructors.

func vmResultOk(v bytecode.Value) bytecode.Value {
	return bytecode.NewADT(resultTagOk, "Ok", []bytecode.Value{v})
}

func vmResultErr(msg string) bytecode.Value {
	return bytecode.NewADT(resultTagErr, "Err", []bytecode.Value{bytecode.NewString(msg)})
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
