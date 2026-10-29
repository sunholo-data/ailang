package vm

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
)

// #1864: getEnv returns Err(NotFound(name)) from the evaluator. EnvError was
// not in bytecode.StdADTTag, so getEnvOr failed on the VM instead of falling
// back. Any std ADT now converts at its declaration-order ordinal.
func TestEvalToBytecodeConvertsUnlistedStdADT(t *testing.T) {
	for ctor, want := range map[string]int{"NotFound": 0, "NotAllowed": 1} {
		v := &eval.TaggedValue{ModulePath: "std/env", TypeName: "EnvError", CtorName: ctor,
			Fields: []eval.Value{&eval.StringValue{Value: "X"}}}
		bv, err := EvalToBytecode(v)
		if err != nil {
			t.Fatalf("%s: %v", ctor, err)
		}
		if bv.Tag != bytecode.TagADT || bv.Obj.(*bytecode.ADTObj).Tag != want {
			t.Errorf("%s: got %v, want ADT tag %d", ctor, bv, want)
		}
	}
	if _, err := EvalToBytecode(&eval.TaggedValue{ModulePath: "user/mod", TypeName: "EnvError", CtorName: "NotFound"}); err == nil {
		t.Error("a user ADT must not borrow a std type's ordinals")
	}
}

// The hand-kept table and the stdlib declarations must agree wherever both
// know a constructor, or the bridge would hand the VM the wrong variant.
func TestStdADTTagAgreesWithStdlibDeclarations(t *testing.T) {
	table := stdADTOrdinals()
	if len(table) < 20 {
		t.Fatalf("parsed only %d std ADTs from the embedded stdlib", len(table))
	}
	checked := 0
	for key, ords := range table {
		dot := strings.LastIndexByte(key, '.')
		mod, typ := key[:dot], key[dot+1:]
		for ctor, ord := range ords {
			if got, ok := bytecode.StdADTTag(mod, typ, ctor); ok {
				checked++
				if got != ord {
					t.Errorf("%s.%s: StdADTTag=%d, declaration order=%d", key, ctor, got, ord)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no constructor is known to both tables")
	}
}
