//go:build js

package effects

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func TestStreamCancelProcessSourceWASM_TypedUnsupportedAndAuthority(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Grant(NewCapability("Process"))
	result, err := Call(ctx, "Stream", "cancelProcessSource", []eval.Value{makeStreamSource(1)})
	if err != nil {
		t.Fatal(err)
	}
	outer, ok := result.(*eval.TaggedValue)
	if !ok || outer.CtorName != "Err" {
		t.Fatalf("unsupported result: %v", result)
	}
	inner := outer.Fields[0].(*eval.TaggedValue)
	if inner.CtorName != "WorkerCancelUnsupported" || inner.ModulePath != "std/process" {
		t.Fatalf("wrong error: %v", inner)
	}
	delete(ctx.Caps, "Process")
	if _, err := Call(ctx, "Stream", "cancelProcessSource", []eval.Value{makeStreamSource(1)}); err == nil {
		t.Fatal("missing Process capability accepted")
	}
}
