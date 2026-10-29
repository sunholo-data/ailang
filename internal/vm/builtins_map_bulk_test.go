package vm

import (
	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
	"testing"
)

func TestMapNativeBulkParity(t *testing.T) {
	pairs := bytecode.NewList([]bytecode.Value{pair(2, i64(20)), pair(-1, i64(10)), pair(10, i64(100)), pair(2, i64(22))})
	for _, input := range []bytecode.Value{pairs, bytecode.NewList(nil)} {
		spec, _ := builtins.GetSpec("_map_from_list")
		evInput, _ := BytecodeToEval(input)
		want, err := spec.Impl(nil, []eval.Value{evInput})
		if err != nil {
			t.Fatal(err)
		}
		m, err := builtinMapFromList([]bytecode.Value{input})
		if err != nil {
			t.Fatal(err)
		}
		wantBC, _ := EvalToBytecode(want)
		if !m.Equal(wantBC) {
			t.Fatalf("fromList %s != %s", m, wantBC)
		}
		for _, name := range []string{"__map_keys", "__map_values", "__map_to_list"} {
			got, err := nativeBuiltin(t, name)([]bytecode.Value{m})
			if err != nil {
				t.Fatal(err)
			}
			spec, _ := builtins.GetSpec(name[1:])
			expected, err := spec.Impl(nil, []eval.Value{want})
			if err != nil {
				t.Fatal(err)
			}
			bc, _ := EvalToBytecode(expected)
			if !got.Equal(bc) {
				t.Fatalf("%s %s != %s", name, got, bc)
			}
		}
	}
	for _, bad := range []bytecode.Value{i64(1), bytecode.NewList([]bytecode.Value{i64(1)}), bytecode.NewList([]bytecode.Value{bytecode.NewTuple(nil)}), bytecode.NewList([]bytecode.Value{bytecode.NewTuple([]bytecode.Value{bytecode.NewList(nil), i64(1)})})} {
		if _, err := builtinMapFromList([]bytecode.Value{bad}); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}
