package vm

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

// --- json_encode tests -------------------------------------------------------

func TestBuiltinJsonEncode_Null(t *testing.T) {
	v := vmMakeJNull()
	got, err := builtinJsonEncode([]bytecode.Value{v})
	if err != nil {
		t.Fatal(err)
	}
	if got.AsString() != "null" {
		t.Fatalf("expected null, got %q", got.AsString())
	}
}

func TestBuiltinJsonEncode_Bool(t *testing.T) {
	for _, tc := range []struct {
		val  bool
		want string
	}{
		{true, "true"},
		{false, "false"},
	} {
		v := vmMakeJBool(tc.val)
		got, err := builtinJsonEncode([]bytecode.Value{v})
		if err != nil {
			t.Fatal(err)
		}
		if got.AsString() != tc.want {
			t.Fatalf("expected %q, got %q", tc.want, got.AsString())
		}
	}
}

func TestBuiltinJsonEncode_Number(t *testing.T) {
	for _, tc := range []struct {
		flt  float64
		want string
	}{
		{42, "42"},
		{3.14, "3.14"},
		{0, "0"},
		{-1, "-1"},
	} {
		v := bytecode.NewADT(jsonTagJNumber, []bytecode.Value{bytecode.NewFloat(tc.flt)})
		got, err := builtinJsonEncode([]bytecode.Value{v})
		if err != nil {
			t.Fatal(err)
		}
		if got.AsString() != tc.want {
			t.Fatalf("for %v: expected %q, got %q", tc.flt, tc.want, got.AsString())
		}
	}
}

func TestBuiltinJsonEncode_String(t *testing.T) {
	v := vmMakeJString("hello \"world\"\nnewline")
	got, err := builtinJsonEncode([]bytecode.Value{v})
	if err != nil {
		t.Fatal(err)
	}
	want := `"hello \"world\"\nnewline"`
	if got.AsString() != want {
		t.Fatalf("expected %q, got %q", want, got.AsString())
	}
}

func TestBuiltinJsonEncode_Array(t *testing.T) {
	arr := bytecode.NewADT(jsonTagJArray, []bytecode.Value{
		bytecode.NewList([]bytecode.Value{
			vmMakeJNull(),
			vmMakeJBool(true),
			vmMakeJString("x"),
		}),
	})
	got, err := builtinJsonEncode([]bytecode.Value{arr})
	if err != nil {
		t.Fatal(err)
	}
	want := `[null,true,"x"]`
	if got.AsString() != want {
		t.Fatalf("expected %s, got %s", want, got.AsString())
	}
}

func TestBuiltinJsonEncode_Object(t *testing.T) {
	obj := bytecode.NewADT(jsonTagJObject, []bytecode.Value{
		bytecode.NewList([]bytecode.Value{
			bytecode.NewRecord([]bytecode.RecordField{
				{Name: "key", Value: bytecode.NewString("name")},
				{Name: "value", Value: vmMakeJString("alice")},
			}),
			bytecode.NewRecord([]bytecode.RecordField{
				{Name: "key", Value: bytecode.NewString("age")},
				{Name: "value", Value: bytecode.NewADT(jsonTagJNumber, []bytecode.Value{bytecode.NewFloat(30)})},
			}),
		}),
	})
	got, err := builtinJsonEncode([]bytecode.Value{obj})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"name":"alice","age":30}`
	if got.AsString() != want {
		t.Fatalf("expected %s, got %s", want, got.AsString())
	}
}

func TestBuiltinJsonEncode_Nested(t *testing.T) {
	inner := bytecode.NewADT(jsonTagJObject, []bytecode.Value{
		bytecode.NewList([]bytecode.Value{
			bytecode.NewRecord([]bytecode.RecordField{
				{Name: "key", Value: bytecode.NewString("x")},
				{Name: "value", Value: bytecode.NewADT(jsonTagJNumber, []bytecode.Value{bytecode.NewFloat(1)})},
			}),
		}),
	})
	outer := bytecode.NewADT(jsonTagJArray, []bytecode.Value{
		bytecode.NewList([]bytecode.Value{inner, vmMakeJNull()}),
	})
	got, err := builtinJsonEncode([]bytecode.Value{outer})
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"x":1},null]`
	if got.AsString() != want {
		t.Fatalf("expected %s, got %s", want, got.AsString())
	}
}

// --- json_decode tests -------------------------------------------------------

func TestBuiltinJsonDecode_Null(t *testing.T) {
	got, err := builtinJsonDecode([]bytecode.Value{bytecode.NewString("null")})
	if err != nil {
		t.Fatal(err)
	}
	assertResultOk(t, got)
	inner := got.AsADT().Fields[0]
	if inner.Tag != bytecode.TagADT || inner.AsADT().Tag != jsonTagJNull {
		t.Fatalf("expected JNull, got %v", inner)
	}
}

func TestBuiltinJsonDecode_Bool(t *testing.T) {
	got, err := builtinJsonDecode([]bytecode.Value{bytecode.NewString("true")})
	if err != nil {
		t.Fatal(err)
	}
	assertResultOk(t, got)
	inner := got.AsADT().Fields[0]
	if inner.AsADT().Tag != jsonTagJBool {
		t.Fatalf("expected JBool, got tag %d", inner.AsADT().Tag)
	}
	if !inner.AsADT().Fields[0].Bool {
		t.Fatalf("expected true")
	}
}

func TestBuiltinJsonDecode_Number(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  float64
	}{
		{"42", 42},
		{"3.14", 3.14},
		{"-1", -1},
	} {
		got, err := builtinJsonDecode([]bytecode.Value{bytecode.NewString(tc.input)})
		if err != nil {
			t.Fatal(err)
		}
		assertResultOk(t, got)
		inner := got.AsADT().Fields[0]
		if inner.AsADT().Tag != jsonTagJNumber {
			t.Fatalf("expected JNumber for %q", tc.input)
		}
		f := inner.AsADT().Fields[0].Flt
		if f != tc.want {
			t.Fatalf("for %q: expected %v, got %v", tc.input, tc.want, f)
		}
	}
}

func TestBuiltinJsonDecode_String(t *testing.T) {
	got, err := builtinJsonDecode([]bytecode.Value{bytecode.NewString(`"hello"`)})
	if err != nil {
		t.Fatal(err)
	}
	assertResultOk(t, got)
	inner := got.AsADT().Fields[0]
	if inner.AsADT().Tag != jsonTagJString {
		t.Fatalf("expected JString")
	}
	if inner.AsADT().Fields[0].AsString() != "hello" {
		t.Fatalf("expected hello, got %q", inner.AsADT().Fields[0].AsString())
	}
}

func TestBuiltinJsonDecode_Array(t *testing.T) {
	got, err := builtinJsonDecode([]bytecode.Value{bytecode.NewString(`[1, "two", null]`)})
	if err != nil {
		t.Fatal(err)
	}
	assertResultOk(t, got)
	inner := got.AsADT().Fields[0]
	if inner.AsADT().Tag != jsonTagJArray {
		t.Fatalf("expected JArray, got tag %d", inner.AsADT().Tag)
	}
	elems := inner.AsADT().Fields[0].AsList()
	if len(elems) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elems))
	}
}

func TestBuiltinJsonDecode_Object(t *testing.T) {
	got, err := builtinJsonDecode([]bytecode.Value{bytecode.NewString(`{"a":1,"b":"two"}`)})
	if err != nil {
		t.Fatal(err)
	}
	assertResultOk(t, got)
	inner := got.AsADT().Fields[0]
	if inner.AsADT().Tag != jsonTagJObject {
		t.Fatalf("expected JObject, got tag %d", inner.AsADT().Tag)
	}
	kvPairs := inner.AsADT().Fields[0].AsList()
	if len(kvPairs) != 2 {
		t.Fatalf("expected 2 kv pairs, got %d", len(kvPairs))
	}
}

func TestBuiltinJsonDecode_InvalidJSON(t *testing.T) {
	got, err := builtinJsonDecode([]bytecode.Value{bytecode.NewString(`{bad`)})
	if err != nil {
		t.Fatal(err)
	}
	// Should be Err result, not a Go error
	adt := got.AsADT()
	if adt.Tag != resultTagErr {
		t.Fatalf("expected Err result for invalid JSON, got tag %d", adt.Tag)
	}
}

func TestBuiltinJsonDecode_NestedObject(t *testing.T) {
	input := `{"users":[{"name":"alice","age":30},{"name":"bob","age":25}]}`
	got, err := builtinJsonDecode([]bytecode.Value{bytecode.NewString(input)})
	if err != nil {
		t.Fatal(err)
	}
	assertResultOk(t, got)
	// Roundtrip: decode then encode should produce equivalent JSON
	inner := got.AsADT().Fields[0]
	encoded, err := builtinJsonEncode([]bytecode.Value{inner})
	if err != nil {
		t.Fatal(err)
	}
	if encoded.AsString() != input {
		t.Fatalf("roundtrip mismatch:\n  input:   %s\n  output:  %s", input, encoded.AsString())
	}
}

// --- json_repair tests -------------------------------------------------------

func TestBuiltinJsonRepair_ValidJSON(t *testing.T) {
	input := `{"key": "value"}`
	got, err := builtinJsonRepair([]bytecode.Value{bytecode.NewString(input)})
	if err != nil {
		t.Fatal(err)
	}
	assertResultOk(t, got)
	if got.AsADT().Fields[0].AsString() != input {
		t.Fatalf("expected unchanged valid JSON")
	}
}

func TestBuiltinJsonRepair_UnclosedBrace(t *testing.T) {
	input := `{"key": "value"`
	got, err := builtinJsonRepair([]bytecode.Value{bytecode.NewString(input)})
	if err != nil {
		t.Fatal(err)
	}
	assertResultOk(t, got)
	repaired := got.AsADT().Fields[0].AsString()
	if !strings.HasSuffix(repaired, "}") {
		t.Fatalf("expected closing brace, got %q", repaired)
	}
}

func TestBuiltinJsonRepair_UnclosedBracket(t *testing.T) {
	input := `[1, 2, 3`
	got, err := builtinJsonRepair([]bytecode.Value{bytecode.NewString(input)})
	if err != nil {
		t.Fatal(err)
	}
	assertResultOk(t, got)
	repaired := got.AsADT().Fields[0].AsString()
	if !strings.HasSuffix(repaired, "]") {
		t.Fatalf("expected closing bracket, got %q", repaired)
	}
}

func TestBuiltinJsonRepair_UnclosedString(t *testing.T) {
	input := `{"key": "truncated`
	got, err := builtinJsonRepair([]bytecode.Value{bytecode.NewString(input)})
	if err != nil {
		t.Fatal(err)
	}
	assertResultOk(t, got)
	repaired := got.AsADT().Fields[0].AsString()
	if !strings.Contains(repaired, `"truncated"`) {
		t.Fatalf("expected string to be closed, got %q", repaired)
	}
}

func TestBuiltinJsonRepair_TrailingComma(t *testing.T) {
	input := `[1, 2, 3,]`
	got, err := builtinJsonRepair([]bytecode.Value{bytecode.NewString(input)})
	if err != nil {
		t.Fatal(err)
	}
	assertResultOk(t, got)
}

func TestBuiltinJsonRepair_EmptyInput(t *testing.T) {
	got, err := builtinJsonRepair([]bytecode.Value{bytecode.NewString("")})
	if err != nil {
		t.Fatal(err)
	}
	// Should be Err result
	if got.AsADT().Tag != resultTagErr {
		t.Fatalf("expected Err for empty input, got tag %d", got.AsADT().Tag)
	}
}

func TestBuiltinJsonRepair_TruncatedKeyword(t *testing.T) {
	input := `[1, tru`
	got, err := builtinJsonRepair([]bytecode.Value{bytecode.NewString(input)})
	if err != nil {
		t.Fatal(err)
	}
	assertResultOk(t, got)
	repaired := got.AsADT().Fields[0].AsString()
	if !strings.Contains(repaired, "true") {
		t.Fatalf("expected 'tru' to be completed to 'true', got %q", repaired)
	}
}

// --- roundtrip test (determinism) --------------------------------------------

func TestBuiltinJsonRoundtrip(t *testing.T) {
	inputs := []string{
		`null`,
		`true`,
		`false`,
		`42`,
		`3.14`,
		`"hello"`,
		`[1,2,3]`,
		`{"a":1,"b":"two"}`,
		`{"nested":{"x":[1,null,true]}}`,
	}
	for _, input := range inputs {
		for i := 0; i < 20; i++ {
			decoded, err := builtinJsonDecode([]bytecode.Value{bytecode.NewString(input)})
			if err != nil {
				t.Fatalf("decode %q: %v", input, err)
			}
			assertResultOk(t, decoded)
			encoded, err := builtinJsonEncode([]bytecode.Value{decoded.AsADT().Fields[0]})
			if err != nil {
				t.Fatalf("encode %q: %v", input, err)
			}
			if encoded.AsString() != input {
				t.Fatalf("roundtrip[%d] %q: got %q", i, input, encoded.AsString())
			}
		}
	}
}

// --- helpers -----------------------------------------------------------------

func assertResultOk(t *testing.T, v bytecode.Value) {
	t.Helper()
	if v.Tag != bytecode.TagADT {
		t.Fatalf("expected ADT, got %v", v.Tag)
	}
	adt := v.AsADT()
	if adt.Tag != resultTagOk {
		msg := ""
		if len(adt.Fields) > 0 {
			msg = adt.Fields[0].AsString()
		}
		t.Fatalf("expected Ok result, got Err: %s", msg)
	}
}
