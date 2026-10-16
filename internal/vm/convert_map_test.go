package vm

import (
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
	"testing"
)

func TestMapBridgeRoundTrip(t *testing.T) {
	inner := &eval.MapValue{Entries: map[string]*eval.MapEntry{}}
	original := &eval.MapValue{Entries: map[string]*eval.MapEntry{
		"i:-2":   {Key: &eval.IntValue{Value: -2}, Value: inner},
		"s:x":    {Key: &eval.StringValue{Value: "x"}, Value: &eval.ListValue{Elements: []eval.Value{&eval.BoolValue{Value: true}}}},
		"b:true": {Key: &eval.BoolValue{Value: true}, Value: &eval.IntValue{Value: 3}},
	}}
	bc, err := EvalToBytecode(original)
	if err != nil {
		t.Fatal(err)
	}
	back, err := BytecodeToEval(bc)
	if err != nil {
		t.Fatal(err)
	}
	if back.String() != original.String() {
		t.Fatalf("%s != %s", back, original)
	}
	again, err := EvalToBytecode(back)
	if err != nil || !bc.Equal(again) {
		t.Fatalf("round trip: %v", err)
	}
	if bytecode.NewString("x").Equal(bc) {
		t.Fatal("map equals string")
	}
}
