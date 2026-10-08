package vm

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

func vmJObject(pairs ...bytecode.Value) bytecode.Value {
	return bytecode.NewADT(jsonTagJObject, "JObject", []bytecode.Value{bytecode.NewList(pairs)})
}

func vmKV(k string, v bytecode.Value) bytecode.Value {
	return bytecode.NewRecord([]bytecode.RecordField{
		{Name: "key", Value: bytecode.NewString(k)},
		{Name: "value", Value: v},
	})
}

func TestBuiltinYamlEncode_MatchesEvaluatorEmitter(t *testing.T) {
	j := vmJObject(
		vmKV("z", bytecode.NewADT(jsonTagJNumber, "JNumber", []bytecode.Value{bytecode.NewFloat(1)})),
		vmKV("a", bytecode.NewADT(jsonTagJArray, "JArray", []bytecode.Value{bytecode.NewList([]bytecode.Value{
			vmMakeJString("x"),
			bytecode.NewADT(jsonTagJBool, "JBool", []bytecode.Value{bytecode.NewBool(true)}),
		})})),
		vmKV("n", bytecode.NewADT(jsonTagJNull, "JNull", nil)),
	)
	got, err := builtinYamlEncode([]bytecode.Value{j})
	if err != nil {
		t.Fatal(err)
	}
	adt := got.AsADT()
	if adt.Tag != resultTagOk {
		t.Fatalf("expected Ok, got %s", adt.Fields[0].AsString())
	}
	want := "\"z\": 1\n\"a\":\n  - \"x\"\n  - true\n\"n\": null\n"
	if s := adt.Fields[0].AsString(); s != want {
		t.Errorf("yaml mismatch\n got: %q\nwant: %q", s, want)
	}
}
