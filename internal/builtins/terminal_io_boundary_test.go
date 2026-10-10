package builtins

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestEOFReaderBuiltinContract(t *testing.T) {
	spec, ok := GetSpec("_io_readLineOpt")
	if !ok {
		t.Fatal("missing EOF reader")
	}
	fn := spec.Type().(*types.TFunc2)
	if len(fn.Params) != 1 || fn.Params[0].String() != "()" || fn.Return.String() != "Option[string]" {
		t.Fatalf("EOF reader signature: %s", fn)
	}
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.Capability{Name: "IO"})
	ctx.IOReader = strings.NewReader("\nlast")
	for _, want := range []struct{ ctor, text string }{{"Some", ""}, {"Some", "last"}, {"None", ""}, {"None", ""}} {
		result, err := spec.Impl(ctx, []eval.Value{&eval.UnitValue{}})
		if err != nil {
			t.Fatal(err)
		}
		value, ok := result.(*eval.TaggedValue)
		if !ok || value.ModulePath != "std/option" || value.TypeName != "Option" || value.CtorName != want.ctor {
			t.Fatalf("invalid EOF result identity: %#v", result)
		}
		if want.ctor == "Some" && value.Fields[0].(*eval.StringValue).Value != want.text {
			t.Fatalf("line lost: %s", value)
		}
	}
	for _, args := range [][]eval.Value{nil, {&eval.StringValue{Value: "not unit"}}, {&eval.UnitValue{}, &eval.UnitValue{}}} {
		if _, err := spec.Impl(ctx, args); err == nil || !strings.Contains(err.Error(), "expected unit") {
			t.Fatalf("malformed EOF invocation accepted: %v", err)
		}
	}
	ctx = effects.NewEffContext(nil)
	if _, err := spec.Impl(ctx, []eval.Value{&eval.UnitValue{}}); err == nil {
		t.Fatal("EOF read bypassed IO")
	}
	ctx.Grant(effects.Capability{Name: "IO"})
	ctx.IOReader = terminalBuiltinFailReader{}
	if _, err := spec.Impl(ctx, []eval.Value{&eval.UnitValue{}}); err == nil || !strings.Contains(err.Error(), "read failure") {
		t.Fatalf("input failure must remain an error: %v", err)
	}
}

type terminalBuiltinFailReader struct{}

func (terminalBuiltinFailReader) Read([]byte) (int, error) { return 0, fmt.Errorf("read failure") }

func TestTerminalInfoBuiltinRejectsInvalidUnit(t *testing.T) {
	spec, _ := GetSpec("_terminal_info")
	ctx := effects.NewEffContext(nil)
	for _, args := range [][]eval.Value{nil, {&eval.IntValue{Value: 0}}} {
		if _, err := spec.Impl(ctx, args); err == nil || !strings.Contains(err.Error(), "expected unit") {
			t.Fatalf("malformed query accepted: %v", err)
		}
	}
}
