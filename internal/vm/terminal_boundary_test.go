package vm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

func terminalEffectIndex(t *testing.T, name string) int {
	t.Helper()
	for i, n := range bytecode.EffectBuiltinNames {
		if n == name {
			return i
		}
	}
	t.Fatalf("native effect %s missing", name)
	return -1
}

func TestNativeEffectRejectsMalformedBoundaryBeforeDispatch(t *testing.T) {
	m := NewVM(bytecode.NewImage())
	m.Effects = effects.NewEffContext(nil)
	m.Effects.Grant(effects.Capability{Name: "IO"})
	for _, tc := range []struct {
		name  string
		index int
		args  []bytecode.Value
		want  string
	}{
		{"negative operation", -1, nil, "unknown effect builtin"},
		{"outside table", len(bytecode.EffectBuiltinNames), nil, "unknown effect builtin"},
		{"missing unit", terminalEffectIndex(t, "__io_readLineOpt"), nil, "expected 1 args"},
		{"callback passed to IO", terminalEffectIndex(t, "__io_println"), []bytecode.Value{bytecode.NewClosure(&bytecode.FuncPrototype{Name: "fake"}, nil)}, "argument 0"},
		{"wrong session tag", terminalEffectIndex(t, "__terminal_readEvent"), []bytecode.Value{bytecode.NewADT(1, "TerminalSession", []bytecode.Value{bytecode.NewInt(1)}), bytecode.NewInt(0)}, "invalid terminal handle representation"},
		{"missing session token", terminalEffectIndex(t, "__terminal_readEvent"), []bytecode.Value{bytecode.NewADT(0, "TerminalSession", nil), bytecode.NewInt(0)}, "invalid terminal handle representation"},
		{"wrong session token", terminalEffectIndex(t, "__terminal_readEvent"), []bytecode.Value{bytecode.NewADT(0, "TerminalSession", []bytecode.Value{bytecode.NewString("token")}), bytecode.NewInt(0)}, "invalid terminal handle representation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := m.callEffectBuiltin(tc.index, tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %s", err, tc.want)
			}
			if m.EffectCalls != 0 {
				t.Fatal("malformed program reached host dispatch")
			}
		})
	}
	// A representationally valid fabricated token reaches the authority check,
	// which returns typed InvalidSession without ever touching stdin.
	got, err := m.callEffectBuiltin(terminalEffectIndex(t, "__terminal_readEvent"), []bytecode.Value{bytecode.NewADT(0, "TerminalSession", []bytecode.Value{bytecode.NewInt(77)}), bytecode.NewInt(0)})
	if err != nil {
		t.Fatal(err)
	}
	if got.AsADT().Ctor != "Err" || got.AsADT().Fields[0].AsADT().Ctor != "InvalidSession" {
		t.Fatalf("fabricated handle gained authority: %v", got)
	}
}

// Faulty embedding operations cannot leak the native callback router. Both
// host argument corruption and callback runtime errors must unwind it.
func TestNativeCallbackBoundaryErrorsRestoreCaller(t *testing.T) {
	previousOp := effects.Registry["IO"]["withTerminal"]
	defer func() { effects.Registry["IO"]["withTerminal"] = previousOp }()
	m := NewVM(bytecode.NewImage())
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.Capability{Name: "IO"})
	ctx.FnCaller = func(eval.Value, eval.Value) (eval.Value, error) { return &eval.IntValue{Value: 91}, nil }
	m.Effects = ctx
	index := terminalEffectIndex(t, "__terminal_withTerminal")
	args := []bytecode.Value{bytecode.NewRecord(nil), bytecode.NewInt(0)}
	for _, tc := range []struct {
		name, want string
		invoke     func(*effects.EffContext, []eval.Value) (eval.Value, error)
	}{
		{"host arity corruption", "expected session", func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			return args[1].(*eval.BuiltinFunction).Fn(nil)
		}},
		{"host session corruption", "TaggedValue", func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			return ctx.FnCaller(args[1], &eval.TaggedValue{ModulePath: "app", TypeName: "Unknown", CtorName: "Unknown"})
		}},
		{"not callable", "CallClosure", func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			return ctx.FnCaller(args[1], &eval.TaggedValue{ModulePath: "std/terminal", TypeName: "TerminalSession", CtorName: "TerminalSession", Fields: []eval.Value{&eval.IntValue{Value: 1}}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			effects.RegisterOp("IO", "withTerminal", tc.invoke)
			_, err := m.callEffectBuiltin(index, args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %s", err, tc.want)
			}
			restored, err := ctx.FnCaller(nil, nil)
			if err != nil || restored.(*eval.IntValue).Value != 91 {
				t.Fatal("previous caller not restored after runtime failure")
			}
		})
	}
	// A second host callback must still reach the embedding's original caller.
	effects.RegisterOp("IO", "withTerminal", func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return ctx.FnCaller(&eval.BuiltinFunction{Name: "embedding_callback"}, &eval.UnitValue{})
	})
	got, err := m.callEffectBuiltin(index, args)
	if err != nil || got.Tag != bytecode.TagInt || got.Int != 91 {
		t.Fatalf("embedding callback routing changed: %v %v", got, err)
	}
	ctx.FnCaller = nil
	if _, err = m.callEffectBuiltin(index, args); err == nil || !strings.Contains(err.Error(), "unconfigured callback") {
		t.Fatalf("unexpected callback acquired missing authority: %v", err)
	}
	if ctx.FnCaller != nil {
		t.Fatal("missing caller not restored")
	}
	// Host result conversion also fails loudly for unknown foreign ADTs.
	effects.RegisterOp("IO", "withTerminal", func(*effects.EffContext, []eval.Value) (eval.Value, error) {
		return &eval.TaggedValue{ModulePath: "app", TypeName: "Foreign", CtorName: "Foreign"}, nil
	})
	if _, err = m.callEffectBuiltin(index, args); err == nil || !strings.Contains(err.Error(), "TaggedValue") {
		t.Fatalf("foreign host result silently converted: %v", err)
	}
}

func TestTerminalResizeAndKeyResultConversion(t *testing.T) {
	wrap := func(typ, ctor string, fields ...eval.Value) *eval.TaggedValue {
		return &eval.TaggedValue{ModulePath: "std/terminal", TypeName: typ, CtorName: ctor, Fields: fields}
	}
	for _, event := range []*eval.TaggedValue{
		wrap("TerminalEvent", "Key", wrap("TerminalKey", "Text", &eval.StringValue{Value: "青"})),
		wrap("TerminalEvent", "Resize", &eval.RecordValue{Fields: map[string]eval.Value{"columns": &eval.IntValue{Value: 2}, "rows": &eval.IntValue{Value: 1}}}),
		wrap("TerminalEvent", "EndOfInput"), wrap("TerminalEvent", "Interrupted"), wrap("TerminalEvent", "Idle"),
		wrap("TerminalError", "CleanupFailure", &eval.StringValue{Value: "restore failed"}),
	} {
		result := &eval.TaggedValue{ModulePath: "std/result", TypeName: "Result", CtorName: "Ok", Fields: []eval.Value{event}}
		got, err := EvalToBytecode(result)
		if err != nil {
			t.Fatal(err)
		}
		converted := got.AsADT().Fields[0].AsADT()
		if converted.Ctor != event.CtorName || len(converted.Fields) != len(event.Fields) {
			t.Fatalf("event shape changed: %v", got)
		}
		if event.CtorName == "Key" && converted.Fields[0].AsADT().Fields[0].AsString() != "青" {
			t.Fatal("unicode key lost")
		}
		if event.CtorName == "Resize" && fmt.Sprint(converted.Fields[0]) != "{columns: 2, rows: 1}" {
			t.Fatalf("tiny physical size changed: %v", got)
		}
	}
	wrapper := &effectCallbackValue{value: bytecode.Unit()}
	if wrapper.Type() != "vm_callback_result" {
		t.Fatal("callback result diagnostic type changed")
	}
}
