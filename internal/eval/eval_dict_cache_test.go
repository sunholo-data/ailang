package eval

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

func dictRecord(t *testing.T, e *CoreEvaluator, class, typ string) *RecordValue {
	t.Helper()
	v, err := e.evalDictRef(&core.DictRef{ClassName: class, TypeName: typ})
	if err != nil {
		t.Fatalf("evalDictRef(%s, %s): %v", class, typ, err)
	}
	rec, ok := v.(*RecordValue)
	if !ok {
		t.Fatalf("evalDictRef(%s, %s) = %T, want *RecordValue", class, typ, v)
	}
	return rec
}

// The hot path: the same class/type evaluated again must reuse the dictionary,
// not rebuild 6-10 Sprintf keys, lookups and wrappers per operator call.
func TestEvalDictRef_ReusesDictionaryForSameClassAndType(t *testing.T) {
	e := NewCoreEvaluatorWithRegistry(types.NewDictionaryRegistry())
	first := dictRecord(t, e, "Num", "Int")
	if again := dictRecord(t, e, "Num", "Int"); again != first {
		t.Fatal("second Num/Int lookup rebuilt the dictionary")
	}
	if other := dictRecord(t, e, "Num", "Float"); other == first {
		t.Fatal("Num/Float returned the Num/Int dictionary")
	}
	if other := dictRecord(t, e, "Ord", "Int"); other == first {
		t.Fatal("Ord/Int returned the Num/Int dictionary")
	}
}

// A Register after the dictionary was built (derived instances can land after
// evaluation starts) must be visible on the next lookup.
func TestEvalDictRef_RegisterInvalidates(t *testing.T) {
	reg := types.NewDictionaryRegistry()
	e := NewCoreEvaluatorWithRegistry(reg)
	before := dictRecord(t, e, "Num", "Int")

	custom := &BuiltinFunction{Name: "custom_add"}
	reg.Register("prelude", "Num", "int", "add", custom)

	after := dictRecord(t, e, "Num", "Int")
	if after == before {
		t.Fatal("dictionary not rebuilt after Register")
	}
	if after.Fields["add"] != custom {
		t.Fatalf("add = %v, want the newly registered implementation", after.Fields["add"])
	}
}

func TestEvalDictRef_SetDictionaryRegistryInvalidates(t *testing.T) {
	e := NewCoreEvaluatorWithRegistry(types.NewDictionaryRegistry())
	before := dictRecord(t, e, "Eq", "Int")
	e.SetDictionaryRegistry(types.NewDictionaryRegistry())
	if after := dictRecord(t, e, "Eq", "Int"); after == before {
		t.Fatal("dictionary from the old registry reused after SetDictionaryRegistry")
	}
}

// Errors are not cached: an unknown class fails every time.
func TestEvalDictRef_UnknownClassStillFails(t *testing.T) {
	e := NewCoreEvaluatorWithRegistry(types.NewDictionaryRegistry())
	for i := 0; i < 2; i++ {
		if _, err := e.evalDictRef(&core.DictRef{ClassName: "Monoid", TypeName: "Int"}); err == nil {
			t.Fatalf("call %d: unknown class succeeded", i+1)
		}
	}
}
