package builtins

import (
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
	"testing"
)

func TestTerminalRegistrationAndEffectRow(t *testing.T) {
	for _, name := range []string{"_terminal_info", "_terminal_withTerminal", "_terminal_readEvent", "_io_readLineOpt"} {
		spec, ok := GetSpec(name)
		if !ok {
			t.Errorf("missing builtin %s", name)
			continue
		}
		if spec.Effect != "IO" || spec.IsPure || spec.Metadata == nil {
			t.Errorf("bad effect/metadata: %s", name)
		}
	}
	spec, ok := GetSpec("_terminal_withTerminal")
	if !ok {
		return
	}
	typ := spec.Type().(*types.TFunc2)
	callback := typ.Params[1].(*types.TFunc2)
	if typ.EffectRow.Tail == nil || callback.EffectRow.Tail == nil || typ.EffectRow.Tail.Name != callback.EffectRow.Tail.Name {
		t.Fatal("callback effects must propagate into scope")
	}
	if _, ok := typ.EffectRow.Labels["IO"]; !ok {
		t.Fatal("scope missing IO")
	}
}

func TestTerminalCapabilityDeniedBeforeHost(t *testing.T) {
	spec, ok := GetSpec("_terminal_info")
	if !ok {
		t.Fatal("missing terminal info builtin")
	}
	ctx := effects.NewEffContext(nil)
	if _, err := spec.Impl(ctx, []eval.Value{&eval.UnitValue{}}); err == nil {
		t.Fatal("terminal info without IO capability succeeded")
	}
}

func TestTerminalInfoChargesIOBudgetOnce(t *testing.T) {
	spec, _ := GetSpec("_terminal_info")
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.Capability{Name: "IO"})
	limit := 1
	ctx.SetBudget(effects.NewBudgetContext(map[string]*int{"IO": &limit}))
	if _, err := spec.Impl(ctx, []eval.Value{&eval.UnitValue{}}); err != nil {
		t.Fatal(err)
	}
	if ctx.Budget.Used("IO") != 1 {
		t.Fatalf("charged %d; want exactly once", ctx.Budget.Used("IO"))
	}
	if _, err := spec.Impl(ctx, []eval.Value{&eval.UnitValue{}}); err == nil {
		t.Fatal("exhausted terminal query budget succeeded")
	}
}
