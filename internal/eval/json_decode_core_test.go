package eval

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

// The shared std/json decode walk (B1 trailing data, B2 empty keys, B3 JInt).

// jsonShape renders a decoded Json compactly: JInt(n), JNumber(f),
// {"k":v,...} with keys in order, [..], JString, JBool, JNull.
func jsonShape(v Value) string {
	tv := v.(*TaggedValue)
	switch tv.CtorName {
	case "JNull":
		return "null"
	case "JBool":
		if tv.Fields[0].(*BoolValue).Value {
			return "true"
		}
		return "false"
	case "JInt":
		return "JInt(" + tv.Fields[0].String() + ")"
	case "JNumber":
		f := tv.Fields[0].(*FloatValue).Value
		if f == 0 && math.Signbit(f) {
			return "JNumber(-0)"
		}
		return "JNumber(" + FormatJSONNumber(f) + ")"
	case "JString":
		return "S(" + tv.Fields[0].(*StringValue).Value + ")"
	case "JArray":
		var parts []string
		for _, e := range tv.Fields[0].(*ListValue).Elements {
			parts = append(parts, jsonShape(e))
		}
		return "[" + strings.Join(parts, ",") + "]"
	case "JObject":
		var parts []string
		for _, e := range tv.Fields[0].(*ListValue).Elements {
			r := e.(*RecordValue)
			parts = append(parts, "\""+r.Fields["key"].(*StringValue).Value+"\":"+jsonShape(r.Fields["value"]))
		}
		return "{" + strings.Join(parts, ",") + "}"
	}
	return "?" + tv.CtorName
}

// JSONDecodeCases is the edge-case table every decoder must agree on. The VM
// package runs the same table through its sink (builtins_json_parity_test.go).
var jsonDecodeCases = []struct{ in, want string }{
	// B3: integer syntax that fits int64 is JInt; everything else JNumber.
	{`1`, `JInt(1)`},
	{`-7`, `JInt(-7)`},
	{`0`, `JInt(0)`},
	{`9007199254740993`, `JInt(9007199254740993)`},
	{`9223372036854775807`, `JInt(9223372036854775807)`},
	{`-9223372036854775808`, `JInt(-9223372036854775808)`},
	{`9223372036854775808`, `JNumber(9223372036854776000)`},
	{`-9223372036854775809`, `JNumber(-9223372036854776000)`},
	{`123456789012345678901234567890`, `JNumber(1.2345678901234568e+29)`},
	{`1.0`, `JNumber(1)`},
	{`1e3`, `JNumber(1000)`},
	{`1E3`, `JNumber(1000)`},
	{`1.5e3`, `JNumber(1500)`},
	{`-0`, `JNumber(-0)`},
	{`-0.0`, `JNumber(-0)`},
	{`1E400`, `ERR json: number 1E400 out of float64 range`},
	{`[1, 2.5, 1234567890123456789]`, `[JInt(1),JNumber(2.5),JInt(1234567890123456789)]`},
	{`[01]`, `ERR invalid json`},
	{`01`, `ERR decode: trailing data at offset 1`},
	// B1: one value, whitespace allowed around it.
	{`1 2`, `ERR decode: trailing data at offset 2`},
	{`{"a":1} xx`, `ERR decode: trailing data at offset 8`},
	{`[1]]`, `ERR decode: trailing data at offset 3`},
	{`{} x`, `ERR decode: trailing data at offset 3`},
	{`"a""b"`, `ERR decode: trailing data at offset 3`},
	{" \t\n1\r\n ", `JInt(1)`},
	{`null `, `null`},
	{``, `ERR no JSON value found`},
	{`   `, `ERR no JSON value found`},
	{`[1,2`, `ERR unexpected end of input`},
	{`{"a":}`, `ERR invalid json`},
	// B2: "" is a key; keys keep source order and duplicates.
	{`{"": 1, "b": 2}`, `{"":JInt(1),"b":JInt(2)}`},
	{`{"": "x", "b": "y"}`, `{"":S(x),"b":S(y)}`},
	{`{"a": {"": [true, null]}, "": ""}`, `{"a":{"":[true,null]},"":S()}`},
	{`{"k": 1, "k": 2}`, `{"k":JInt(1),"k":JInt(2)}`},
	{`{"x": "", "": "x"}`, `{"x":S(),"":S(x)}`},
	{`[]`, `[]`},
	{`{}`, `{}`},
}

func TestDecodeJSONEdgeCases(t *testing.T) {
	for _, c := range jsonDecodeCases {
		v, err := DecodeJSON[Value](c.in, TaggedJSONSink{})
		got := ""
		if err != nil {
			got = "ERR " + err.Error()
		} else {
			got = jsonShape(v)
		}
		if strings.HasPrefix(c.want, "ERR ") {
			if !strings.HasPrefix(got, c.want) {
				t.Errorf("decode(%q) = %s, want prefix %s", c.in, got, c.want)
			}
			continue
		}
		if got != c.want {
			t.Errorf("decode(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

// DecodeJSONString and the legacy evaluator builtin wrap the same walk.
func TestDecodeJSONStringWraps(t *testing.T) {
	ok := DecodeJSONString(`{"":1}`).(*TaggedValue)
	if ok.CtorName != "Ok" || jsonShape(ok.Fields[0]) != `{"":JInt(1)}` {
		t.Fatalf("got %s", ok)
	}
	bad := DecodeJSONString(`1 2`).(*TaggedValue)
	if bad.CtorName != "Err" || bad.Fields[0].(*StringValue).Value != "decode: trailing data at offset 2" {
		t.Fatalf("got %s", bad)
	}
	legacy, err := legacyDecodeString(&StringValue{Value: `[1, 1.5]`})
	if err != nil || jsonShape(legacy.(*TaggedValue).Fields[0]) != `[JInt(1),JNumber(1.5)]` {
		t.Fatalf("legacy: %v %v", legacy, err)
	}
}

func TestJSONNumberClassification(t *testing.T) {
	for in, wantInt := range map[string]bool{
		"0": true, "-1": true, "42": true, "9223372036854775807": true,
		"-0": false, "1.0": false, "1e0": false, "1E2": false, "9223372036854775808": false,
	} {
		isInt, _, _, err := JSONNumber(json.Number(in))
		if err != nil || isInt != wantInt {
			t.Errorf("JSONNumber(%s) = int:%v err:%v, want int:%v", in, isInt, err, wantInt)
		}
	}
}

// encode of the legacy evaluator prints JInt exactly and rejects a malformed one.
func TestLegacyEncodeJInt(t *testing.T) {
	s, err := encodeJSON(&TaggedValue{TypeName: "Json", CtorName: "JInt", Fields: []Value{&IntValue{Value: 1234567890123456789}}})
	if err != nil || s != "1234567890123456789" {
		t.Fatalf("got %q %v", s, err)
	}
	if _, err := encodeJSON(&TaggedValue{TypeName: "Json", CtorName: "JInt", Fields: []Value{&FloatValue{Value: 1}}}); err == nil {
		t.Fatal("JInt with a float field must be an error")
	}
	if _, err := encodeJSON(&TaggedValue{TypeName: "Json", CtorName: "JInt"}); err == nil {
		t.Fatal("JInt with no field must be an error")
	}
	if v := legacyValueToJSON(&IntValue{Value: 1 << 60}).(*TaggedValue); v.CtorName != "JInt" {
		t.Fatalf("an AILANG int converts to %s, want JInt", v.CtorName)
	}
}
