package builtins

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

func TestProcessSpawnResultRegistration(t *testing.T) {
	spec, ok := GetSpec("_process_spawn_result")
	if !ok {
		t.Fatal("missing spawn result builtin")
	}
	if spec.NumArgs != 2 || spec.IsPure || spec.Effect != "Process" || spec.Module != "std/process" || spec.Metadata == nil {
		t.Fatal("invalid builtin metadata")
	}
	typ := spec.Type().(*types.TFunc2)
	if _, ok := typ.EffectRow.Labels["Process"]; !ok {
		t.Fatal("Process effect absent")
	}
	for _, name := range []string{"Result", "ProcessHandle", "ProcessError"} {
		if !strings.Contains(typ.Return.String(), name) {
			t.Fatalf("return type missing %s: %v", name, typ)
		}
	}
}
