//go:build js

package effects

import (
	"github.com/sunholo-data/ailang/internal/eval"
	"testing"
)

func TestProcessCancelWASM_TypedUnsupportedAndAuthority(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Process"))
	handle := &eval.TaggedValue{ModulePath: "std/process", TypeName: "ProcessHandle", CtorName: "ProcessHandle", Fields: []eval.Value{&eval.IntValue{Value: 1}}}
	result, err := Call(ctx, "Process", "cancelProcess", []eval.Value{handle})
	if err != nil {
		t.Fatal(err)
	}
	outer, ok := result.(*eval.TaggedValue)
	if !ok || outer.CtorName != "Err" || outer.ModulePath != "std/result" {
		t.Fatalf("unsupported result: %v", result)
	}
	inner := outer.Fields[0].(*eval.TaggedValue)
	if inner.CtorName != "WorkerCancelUnsupported" || inner.TypeName != "WorkerCancelError" {
		t.Fatalf("error identity: %v", inner)
	}
	delete(ctx.Caps, "Process")
	if _, err := Call(ctx, "Process", "cancelProcess", []eval.Value{handle}); err == nil {
		t.Fatal("missing Process capability accepted")
	}
}
