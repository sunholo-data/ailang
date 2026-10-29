package vm

import (
	"math"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
)

// The evaluator and the VM each walk their own value type into the one
// grammar in internal/pcoll. Maps cross the engine bridge carrying their
// canonical keys, so the two walkers must agree byte for byte on every key
// kind, nested ones included (M-PERSISTENT-COLLECTIONS 1a).

func ei(n int) eval.Value            { return &eval.IntValue{Value: n} }
func es(s string) eval.Value         { return &eval.StringValue{Value: s} }
func ef(f float64) eval.Value        { return &eval.FloatValue{Value: f} }
func eb(b bool) eval.Value           { return &eval.BoolValue{Value: b} }
func et(xs ...eval.Value) eval.Value { return &eval.TupleValue{Elements: xs} }
func er(fs map[string]eval.Value) eval.Value {
	return &eval.RecordValue{Fields: fs}
}
func ector(name string, fs ...eval.Value) eval.Value {
	return &eval.TaggedValue{ModulePath: "user/m", TypeName: "Dir", CtorName: name, Fields: fs}
}
func esome(v eval.Value) eval.Value {
	return &eval.TaggedValue{ModulePath: "std/option", TypeName: "Option", CtorName: "Some", Fields: []eval.Value{v}}
}

// bcMirror converts through the production bridge where it can; user ADTs
// (which the bridge refuses without the compiler's type table) are rebuilt
// the way the VM's MAKE_ADT builds them: a local ordinal plus the ctor name.
func bcMirror(t *testing.T, v eval.Value) bytecode.Value {
	t.Helper()
	switch x := v.(type) {
	case *eval.TaggedValue:
		if x.ModulePath == "user/m" {
			fields := make([]bytecode.Value, len(x.Fields))
			for i, f := range x.Fields {
				fields[i] = bcMirror(t, f)
			}
			return bytecode.NewADT(int(x.CtorName[0]), x.CtorName, fields)
		}
	case *eval.TupleValue:
		elems := make([]bytecode.Value, len(x.Elements))
		for i, e := range x.Elements {
			elems[i] = bcMirror(t, e)
		}
		return bytecode.NewTuple(elems)
	case *eval.RecordValue:
		fields := make([]bytecode.RecordField, 0, len(x.Fields))
		for name, f := range x.Fields {
			fields = append(fields, bytecode.RecordField{Name: name, Value: bcMirror(t, f)})
		}
		return bytecode.NewRecord(fields)
	}
	bv, err := EvalToBytecode(v)
	if err != nil {
		t.Fatalf("bridge %s: %v", v, err)
	}
	return bv
}

func TestMapKeyCrossEngineGolden(t *testing.T) {
	cases := []struct {
		name string
		v    eval.Value
		want string // canonical bytes; legacy kinds frozen (D2)
	}{
		{"int", ei(-12), "i:-12"},
		{"string", es("x\x00y"), "s:x\x00y"},
		{"bool", eb(false), "b:false"},
		{"unit", &eval.UnitValue{}, "u:"},
		{"float", ef(2.5), "f:c004000000000000"},
		{"negative zero folds", ef(math.Copysign(0, -1)), "f:8000000000000000"},
		{"tuple", et(ei(1), ei(2), ei(3)), "t:i:1\x00i:2\x00i:3\x00"},
		{"record", er(map[string]eval.Value{"y": ei(2), "x": ei(1)}), "r:x\x00i:1\x00y\x00i:2\x00"},
		{"nullary ctor", ector("North"), "c:North\x00"},
		{"std ctor", esome(ei(4)), "c:Some\x00i:4\x00"},
		{"nested", et(ector("East"), er(map[string]eval.Value{"p": et(ei(1), es("a\x00"))}), ef(-1)),
			"t:c:East\x00\xff\x00r:p\x00\xfft:i:1\x00\xff\xffs:a\x00\xff\xff\xff\x00\xff\xff\x00\xff\x00f:400fffffffffffff\x00"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ek, err := eval.MapKey(c.v)
			if err != nil {
				t.Fatalf("eval.MapKey: %v", err)
			}
			bk, err := bytecode.MapKey(bcMirror(t, c.v))
			if err != nil {
				t.Fatalf("bytecode.MapKey: %v", err)
			}
			if ek != bk {
				t.Fatalf("engines disagree:\n eval %q\n   vm %q", ek, bk)
			}
			if ek != c.want {
				t.Fatalf("golden: got %q want %q", ek, c.want)
			}
		})
	}
}

// Every non-encodable value is an error naming its type, in both engines,
// and the message is the same in both.
func TestMapKeyUnsupportedSameErrorBothEngines(t *testing.T) {
	cases := []struct {
		kind string
		ev   eval.Value
		bv   bytecode.Value
	}{
		{"function", &eval.FunctionValue{}, bytecode.Value{Tag: bytecode.TagClosure, Obj: &bytecode.ClosureObj{}}},
		{"list", &eval.ListValue{}, bytecode.NewList(nil)},
		{"map", &eval.MapValue{Entries: map[string]*eval.MapEntry{}}, bytecode.NewMap(nil)},
	}
	for _, c := range cases {
		_, eerr := eval.MapKey(et(ei(1), c.ev))
		_, berr := bytecode.MapKey(bytecode.NewTuple([]bytecode.Value{bytecode.NewInt(1), c.bv}))
		if eerr == nil || berr == nil {
			t.Fatalf("%s: want errors from both engines, got eval=%v vm=%v", c.kind, eerr, berr)
		}
		if !strings.Contains(eerr.Error(), "unsupported map key type: "+c.kind) || eerr.Error() != berr.Error() {
			t.Fatalf("%s: messages differ or do not name the type:\n eval %v\n   vm %v", c.kind, eerr, berr)
		}
	}
	if _, err := eval.MapKey(ef(math.NaN())); err == nil {
		t.Fatal("eval NaN key: want error")
	}
	if _, err := bytecode.MapKey(bytecode.NewFloat(math.NaN())); err == nil {
		t.Fatal("vm NaN key: want error")
	}
}

// A map with structured keys survives the bridge both ways, keys intact.
func TestStructuredKeyMapCrossesBridge(t *testing.T) {
	m := &eval.MapValue{Entries: map[string]*eval.MapEntry{}}
	for _, k := range []eval.Value{et(ei(1), ei(2)), et(ei(10), ei(0)), et(ei(2), ei(-1))} {
		var err error
		m, err = m.Insert(k, eb(true))
		if err != nil {
			t.Fatal(err)
		}
	}
	bc, err := EvalToBytecode(m)
	if err != nil {
		t.Fatal(err)
	}
	for k, e := range bc.AsMap().Entries {
		recomputed, err := bytecode.MapKey(e.Key)
		if err != nil || recomputed != k {
			t.Fatalf("VM recomputes a different key for %s: %q vs carried %q (%v)", e.Key, recomputed, k, err)
		}
	}
	if _, ok, err := bc.AsMap().Lookup(bytecode.NewTuple([]bytecode.Value{bytecode.NewInt(10), bytecode.NewInt(0)})); err != nil || !ok {
		t.Fatalf("VM lookup of an evaluator-built tuple key: ok=%v err=%v", ok, err)
	}
	back, err := BytecodeToEval(bc)
	if err != nil {
		t.Fatal(err)
	}
	if back.String() != m.String() {
		t.Fatalf("round trip: %s != %s", back, m)
	}
}
