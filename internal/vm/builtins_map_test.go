package vm

import (
	"github.com/sunholo-data/ailang/internal/bytecode"
	"testing"
)

func TestMapNativePointOperations(t *testing.T) {
	for _, key := range []bytecode.Value{bytecode.NewInt(-2), bytecode.NewString("x"), bytecode.NewBool(true)} {
		m, err := builtinMapEmpty(nil)
		if err != nil {
			t.Fatal(err)
		}
		// Native containers must retain opaque VM values without an evaluator bridge.
		val := bytecode.NewADT(9, "Opaque", []bytecode.Value{bytecode.NewInt(3)})
		inserted, err := builtinMapInsert([]bytecode.Value{m, key, val})
		if err != nil {
			t.Fatal(err)
		}
		got, err := builtinMapLookup([]bytecode.Value{inserted, key})
		if err != nil || got.AsADT().Ctor != "Some" || !got.AsADT().Fields[0].Equal(val) {
			t.Fatalf("lookup: %s %v", got, err)
		}
		absent, _ := builtinMapLookup([]bytecode.Value{m, key})
		if absent.AsADT().Ctor != "None" {
			t.Fatal("insert mutated source")
		}
		member, _ := builtinMapMember([]bytecode.Value{inserted, key})
		if !member.Bool {
			t.Fatal("not member")
		}
		size, _ := builtinMapSize([]bytecode.Value{inserted})
		if size.Int != 1 {
			t.Fatal(size)
		}
		removed, err := builtinMapRemove([]bytecode.Value{inserted, key})
		if err != nil {
			t.Fatal(err)
		}
		size, _ = builtinMapSize([]bytecode.Value{removed})
		if size.Int != 0 {
			t.Fatal(size)
		}
		size, _ = builtinMapSize([]bytecode.Value{inserted})
		if size.Int != 1 {
			t.Fatal("remove mutated source")
		}
	}
}
