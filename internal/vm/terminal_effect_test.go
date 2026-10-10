package vm

import (
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/bytecode/compiler"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"strings"
	"testing"
)

func TestTerminalEOFNativeVM(t *testing.T) {
	img, err := compiler.Compile(&stmt.Program{FuncDecls: []stmt.FuncDecl{{Name: "read", Exported: true, Return: stmt.BuiltinCall{Name: "__io_readLineOpt", Args: []stmt.Expr{stmt.LitUnit{}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	var proto *bytecode.FuncPrototype
	for _, p := range img.Prototypes {
		if p.Name == "read" {
			proto = p
		}
	}
	if proto == nil || proto.EvalOnly {
		t.Fatal("EOF reader did not compile natively")
	}
	m := NewVM(img)
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.Capability{Name: "IO"})
	ctx.IOReader = strings.NewReader("\nlast")
	m.Effects = ctx
	for _, want := range []struct{ ctor, text string }{{"Some", ""}, {"Some", "last"}, {"None", ""}, {"None", ""}} {
		got, err := m.Run(proto, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got.Tag != bytecode.TagADT || got.AsADT().Ctor != want.ctor {
			t.Fatalf("got %v, want %s", got, want.ctor)
		}
		if want.ctor == "Some" && got.AsADT().Fields[0].AsString() != want.text {
			t.Fatalf("got %v, want %q", got, want.text)
		}
	}
	// A new VM with the same context cannot bypass revoked IO authority.
	m.Effects = effects.NewEffContext(nil)
	if _, err := m.Run(proto, nil); err == nil {
		t.Fatal("IO capability bypassed")
	}
	m.Effects = nil
	if _, err := m.Run(proto, nil); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("unconfigured host: %v", err)
	}
}

func TestTerminalADTConversion(t *testing.T) {
	for _, pair := range []struct {
		typ, ctor string
		tag       int
	}{{"TerminalSession", "TerminalSession", 0}, {"TerminalEvent", "Resize", 1}, {"TerminalError", "InvalidTimeout", 4}, {"TerminalKey", "PageDown", 12}} {
		got, ok := bytecode.StdADTTag("std/terminal", pair.typ, pair.ctor)
		if !ok || got != pair.tag {
			t.Errorf("%s.%s: tag=%d ok=%v", pair.typ, pair.ctor, got, ok)
		}
	}
}

// A test host operation isolates the VM callback seam; real terminal ownership
// and cleanup are exercised by the PTY integration tests.
func TestTerminalNativeCallbackPreservesCustomADTAndFnCaller(t *testing.T) {
	previousOp := effects.Registry["IO"]["withTerminal"]
	defer func() { effects.Registry["IO"]["withTerminal"] = previousOp }()
	effects.RegisterOp("IO", "withTerminal", func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		value, err := ctx.FnCaller(args[1], &eval.TaggedValue{ModulePath: "std/terminal", TypeName: "TerminalSession", CtorName: "TerminalSession", Fields: []eval.Value{&eval.IntValue{Value: 7}}})
		if err != nil {
			return nil, err
		}
		return &eval.TaggedValue{ModulePath: "std/result", TypeName: "Result", CtorName: "Ok", Fields: []eval.Value{value}}, nil
	})
	img := bytecode.NewImage()
	body := &bytecode.FuncPrototype{Name: "nativeBody", NumRegs: 2, NumParams: 1}
	addConstants(img, body, bytecode.NewADT(1, "CustomChoice", []bytecode.Value{bytecode.NewInt(42)}))
	body.Instructions = []bytecode.Instruction{bytecode.EncodeABx(bytecode.OpLoadConst, 1, 0), bytecode.EncodeABC(bytecode.OpReturn, 1, 0, 0)}
	img.AddPrototype(body)
	m := NewVM(img)
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.Capability{Name: "IO"})
	ctx.FnCaller = func(eval.Value, eval.Value) (eval.Value, error) { return &eval.IntValue{Value: 99}, nil }
	m.Effects = ctx
	args := []bytecode.Value{bytecode.NewRecord([]bytecode.RecordField{{Name: "alternate_screen", Value: bytecode.NewBool(false)}, {Name: "hide_cursor", Value: bytecode.NewBool(false)}}), bytecode.NewClosure(body, nil)}
	index := -1
	for i, name := range bytecode.EffectBuiltinNames {
		if name == "__terminal_withTerminal" {
			index = i
		}
	}
	got, err := m.callEffectBuiltin(index, args)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tag != bytecode.TagADT || got.AsADT().Ctor != "Ok" {
		t.Fatalf("got %v", got)
	}
	choice := got.AsADT().Fields[0]
	if choice.Tag != bytecode.TagADT || choice.AsADT().Ctor != "CustomChoice" || choice.AsADT().Tag != 1 || choice.AsADT().Fields[0].Int != 42 {
		t.Fatalf("callback result changed: %v", choice)
	}
	restored, err := ctx.FnCaller(nil, nil)
	if err != nil || restored.(*eval.IntValue).Value != 99 {
		t.Fatal("FnCaller not restored")
	}

	effects.RegisterOp("IO", "withTerminal", func(*effects.EffContext, []eval.Value) (eval.Value, error) { panic("test scope panic") })
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected host panic")
			}
		}()
		_, _ = m.callEffectBuiltin(index, args)
	}()
	restored, err = ctx.FnCaller(nil, nil)
	if err != nil || restored.(*eval.IntValue).Value != 99 {
		t.Fatal("FnCaller not restored after panic")
	}
}

func TestTerminalDemoArgsNativeVM(t *testing.T) {
	img, err := compiler.Compile(&stmt.Program{FuncDecls: []stmt.FuncDecl{{Name: "args", Exported: true, Return: stmt.BuiltinCall{Name: "__env_getArgs", Args: []stmt.Expr{stmt.LitUnit{}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	var proto *bytecode.FuncPrototype
	for _, p := range img.Prototypes {
		if p.Name == "args" {
			proto = p
		}
	}
	if proto == nil || proto.EvalOnly {
		t.Fatal("demo args did not compile natively")
	}
	m := NewVM(img)
	ctx := effects.NewEffContext([]string{"--native", "colour=青"})
	m.Effects = ctx
	if _, err := m.Run(proto, nil); err == nil {
		t.Fatal("Env capability bypassed")
	}
	ctx.Grant(effects.Capability{Name: "Env"})
	got, err := m.Run(proto, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tag != bytecode.TagList || len(got.AsList()) != 2 || got.AsList()[0].AsString() != "--native" || got.AsList()[1].AsString() != "colour=青" {
		t.Fatalf("args changed: %v", got)
	}
}

func TestTerminalCallbackDiagnosticBounded(t *testing.T) {
	payload := strings.Repeat("x", 1<<20)
	wrapper := &effectCallbackValue{value: bytecode.NewString(payload)}
	if rendered := wrapper.String(); len(rendered) > 80 || strings.Contains(rendered, payload[:100]) {
		t.Fatalf("callback diagnostic materialized payload: %d bytes", len(rendered))
	}
	actual, err := EvalToBytecode(wrapper)
	if err != nil || actual.Tag != bytecode.TagString || actual.AsString() != payload {
		t.Fatal("bounded trace changed callback result")
	}
}
