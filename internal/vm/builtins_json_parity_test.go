package vm

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
)

// The VM decodes through the shared eval.DecodeJSON walk with its own sink.
// This pins the seam: for every edge case the VM's value, converted to the
// evaluator's form, equals the evaluator's, and the errors are the same text.
func TestVMJSONDecodeMatchesEvaluator(t *testing.T) {
	inputs := []string{
		`1`, `-7`, `0`, `9007199254740993`, `9223372036854775807`, `-9223372036854775808`,
		`9223372036854775808`, `123456789012345678901234567890`, `1.0`, `1e3`, `-0`, `-0.0`,
		`1E400`, `[1, 2.5, 1234567890123456789]`, `[01]`, `01`,
		`1 2`, `{"a":1} xx`, `[1]]`, `{} x`, " \t\n1\r\n ", ``, `   `, `[1,2`,
		`{"": 1, "b": 2}`, `{"": "x", "b": "y"}`, `{"a": {"": [true, null]}, "": ""}`,
		`{"k": 1, "k": 2}`, `[]`, `{}`, `"s"`, `false`, `null`,
	}
	for _, in := range inputs {
		vmRes, err := builtinJsonDecode([]bytecode.Value{bytecode.NewString(in)})
		if err != nil {
			t.Fatalf("decode(%q): %v", in, err)
		}
		evRes := eval.DecodeJSONString(in).(*eval.TaggedValue)
		adt := vmRes.AsADT()
		if (adt.Tag == resultTagOk) != (evRes.CtorName == "Ok") {
			t.Errorf("decode(%q): VM tag %d vs evaluator %s", in, adt.Tag, evRes.CtorName)
			continue
		}
		if evRes.CtorName == "Err" {
			if got, want := adt.Fields[0].AsString(), evRes.Fields[0].(*eval.StringValue).Value; got != want {
				t.Errorf("decode(%q): VM error %q vs evaluator %q", in, got, want)
			}
			continue
		}
		conv, err := vmJsonToEval(adt.Fields[0])
		if err != nil {
			t.Fatalf("decode(%q): convert: %v", in, err)
		}
		if got, want := conv.String(), evRes.Fields[0].String(); got != want {
			t.Errorf("decode(%q): VM %s vs evaluator %s", in, got, want)
		}
		// And the encoders agree byte for byte.
		enc, err := builtinJsonEncode([]bytecode.Value{adt.Fields[0]})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := eval.DecodeJSON[bytecode.Value](enc.AsString(), vmJSONSink{}); err != nil {
			t.Errorf("decode(%q): VM encode %q does not decode: %v", in, enc.AsString(), err)
		}
	}
}

func TestVMJSONEncodeJInt(t *testing.T) {
	v := bytecode.NewADT(jsonTagJInt, "JInt", []bytecode.Value{bytecode.NewInt(-9223372036854775808)})
	enc, err := builtinJsonEncode([]bytecode.Value{v})
	if err != nil || enc.AsString() != "-9223372036854775808" {
		t.Fatalf("got %v %v", enc, err)
	}
	bad := bytecode.NewADT(jsonTagJInt, "JInt", []bytecode.Value{bytecode.NewFloat(1)})
	if _, err := builtinJsonEncode([]bytecode.Value{bad}); err == nil {
		t.Fatal("JInt with a float field must be an error")
	}
	if _, err := builtinJsonEncode([]bytecode.Value{bytecode.NewADT(jsonTagJInt, "JInt", nil)}); err == nil {
		t.Fatal("JInt with no field must be an error")
	}
	ev, err := vmJsonToEval(v)
	if err != nil || ev.(*eval.TaggedValue).CtorName != "JInt" {
		t.Fatalf("vmJsonToEval: %v %v", ev, err)
	}
	if _, err := vmJsonToEval(bad); err == nil {
		t.Fatal("vmJsonToEval must reject a JInt with a float field")
	}
}
