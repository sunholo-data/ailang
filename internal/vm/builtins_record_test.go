package vm

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

// _record_set is the by-name record update the compiler emits when a
// record's static type is unknown (ailang#1354, #1355). It must match the
// evaluator: replace the named field, or add it when absent.

func recXY(x, y int64) bytecode.Value {
	return bytecode.NewRecord([]bytecode.RecordField{
		{Name: "y", Value: bytecode.NewInt(y)},
		{Name: "x", Value: bytecode.NewInt(x)},
	})
}

func TestBuiltinRecordSet_ReplacesAndAdds(t *testing.T) {
	got, err := builtinRecordSet([]bytecode.Value{recXY(1, 2), bytecode.NewString("x"), bytecode.NewInt(99)})
	if err != nil {
		t.Fatal(err)
	}
	f := got.AsRecord()
	if len(f) != 2 || f[0].Name != "x" || f[0].Value.Int != 99 || f[1].Name != "y" || f[1].Value.Int != 2 {
		t.Errorf("replace: got %+v, want {x: 99, y: 2}", f)
	}

	got, err = builtinRecordSet([]bytecode.Value{recXY(1, 2), bytecode.NewString("a"), bytecode.NewInt(7)})
	if err != nil {
		t.Fatal(err)
	}
	f = got.AsRecord()
	if len(f) != 3 || f[0].Name != "a" || f[0].Value.Int != 7 {
		t.Errorf("add: got %+v, want sorted {a: 7, x: 1, y: 2}", f)
	}

	// The input record is not mutated.
	base := recXY(1, 2)
	if _, err := builtinRecordSet([]bytecode.Value{base, bytecode.NewString("x"), bytecode.NewInt(5)}); err != nil {
		t.Fatal(err)
	}
	if base.AsRecord()[0].Value.Int != 1 {
		t.Error("builtinRecordSet mutated its input record")
	}
}

func TestBuiltinRecordSet_RejectsBadArgs(t *testing.T) {
	for name, args := range map[string][]bytecode.Value{
		"arity":      {recXY(1, 2), bytecode.NewString("x")},
		"not record": {bytecode.NewInt(1), bytecode.NewString("x"), bytecode.NewInt(2)},
		"not string": {recXY(1, 2), bytecode.NewInt(0), bytecode.NewInt(2)},
	} {
		if _, err := builtinRecordSet(args); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
