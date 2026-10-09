package builtins

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

func TestIOReadLineOpt_Spec(t *testing.T) {
	spec, ok := GetSpec("_io_readLineOpt")
	if !ok {
		t.Fatal("readLineOpt not registered")
	}
	if spec.NumArgs != 1 || spec.IsPure || spec.Effect != "IO" || spec.Module != "std/io" {
		t.Fatalf("invalid spec: %+v", spec)
	}
	if got := spec.Type().String(); !strings.Contains(got, "Option") || !strings.Contains(got, "string") || !strings.Contains(got, "IO") {
		t.Fatalf("wrong type: %s", got)
	}
	if spec.Metadata == nil || spec.Metadata.Description == "" || spec.Metadata.Since == "" {
		t.Fatal("missing metadata")
	}
	ctx := effects.NewEffContext(nil)
	ctx.IOReader = strings.NewReader("\n")
	if _, err := spec.Impl(ctx, []eval.Value{&eval.UnitValue{}}); err == nil {
		t.Fatal("missing capability accepted")
	}
	ctx.Grant(effects.NewCapability("IO"))
	v, err := spec.Impl(ctx, []eval.Value{&eval.UnitValue{}})
	if err != nil {
		t.Fatal(err)
	}
	if opt := v.(*eval.TaggedValue); opt.CtorName != "Some" || opt.Fields[0].(*eval.StringValue).Value != "" {
		t.Fatalf("expected blank line, got %v", v)
	}
	for _, args := range [][]eval.Value{nil, {&eval.IntValue{Value: 1}}, {&eval.UnitValue{}, &eval.UnitValue{}}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("invalid unit arguments accepted")
				}
			}()
			_, _ = spec.Impl(ctx, args)
		}()
	}
}
