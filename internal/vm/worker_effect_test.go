package vm

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/bytecode/compiler"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

func TestWorkerEffectsCompileNatively(t *testing.T) {
	for _, name := range []string{"__process_spawn_process", "__process_write_stdin", "__process_close_stdin", "__process_cancel", "__stream_async_exec_process", "__stream_cancel_process_source"} {
		t.Run(name, func(t *testing.T) {
			img, err := compiler.Compile(&stmt.Program{FuncDecls: []stmt.FuncDecl{{Name: "worker", Return: stmt.BuiltinCall{Name: name, Args: []stmt.Expr{stmt.LitUnit{}}}}}})
			if err != nil {
				t.Fatal(err)
			}
			if len(img.Prototypes) != 1 || img.Prototypes[0].EvalOnly {
				t.Fatalf("%s requires evaluator fallback", name)
			}
		})
	}
}

func TestWorkerADTConversionScoped(t *testing.T) {
	for _, pair := range []struct {
		module, typ, ctor string
		tag               int
	}{
		{"std/process", "ProcessHandle", "ProcessHandle", 0},
		{"std/stream", "StreamSource", "StreamSource", 0},
		{"std/process", "WorkerCancelError", "WorkerHandleInvalid", 0},
		{"std/process", "WorkerCancelError", "WorkerCancelUnsupported", 1},
		{"std/process", "WorkerCancelError", "WorkerCancelTimedOut", 2},
		{"std/process", "WorkerCancelError", "WorkerCancelFailed", 3},
	} {
		value := &eval.TaggedValue{ModulePath: pair.module, TypeName: pair.typ, CtorName: pair.ctor, Fields: []eval.Value{&eval.IntValue{Value: 2000}}}
		got, err := EvalToBytecode(value)
		if err != nil {
			t.Errorf("%s.%s: %v", pair.typ, pair.ctor, err)
			continue
		}
		if got.AsADT().Tag != pair.tag {
			t.Errorf("%s: got ordinal %d want %d", pair.ctor, got.AsADT().Tag, pair.tag)
		}
		value.ModulePath = "application"
		if _, err := EvalToBytecode(value); err == nil {
			t.Errorf("foreign %s admitted as stdlib type", pair.ctor)
		}
	}
}

func TestWorkerNativeCancellationAuthority(t *testing.T) {
	for _, tc := range []struct {
		name, ctor string
		caps       []string
	}{
		{"__process_cancel", "ProcessHandle", []string{"Process"}},
		{"__stream_cancel_process_source", "StreamSource", []string{"Stream", "Process"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := NewVM(bytecode.NewImage())
			m.Effects = effects.NewEffContext(nil)
			index := terminalEffectIndex(t, tc.name)
			args := []bytecode.Value{bytecode.NewADT(0, tc.ctor, []bytecode.Value{bytecode.NewInt(987654)})}
			if _, err := m.callEffectBuiltin(index, args); err == nil {
				t.Fatal("missing capability gained cancellation authority")
			}
			for _, cap := range tc.caps {
				m.Effects.Grant(effects.Capability{Name: cap})
			}
			got, err := m.callEffectBuiltin(index, args)
			if err != nil {
				t.Fatal(err)
			}
			if got.AsADT().Ctor != "Err" {
				t.Fatalf("fabricated handle accepted: %v", got)
			}
			ctor := got.AsADT().Fields[0].AsADT().Ctor
			if ctor != "WorkerHandleInvalid" && !(ctor == "WorkerCancelUnsupported" && !effects.WorkerCancellationSupported()) {
				t.Fatalf("unexpected failure %s", ctor)
			}
			before := m.EffectCalls
			args[0] = bytecode.NewADT(0, tc.ctor, []bytecode.Value{bytecode.NewString("token")})
			if _, err := m.callEffectBuiltin(index, args); err == nil || !strings.Contains(err.Error(), "handle representation") {
				t.Fatalf("malformed handle: %v", err)
			}
			if m.EffectCalls != before {
				t.Fatal("malformed handle reached host")
			}
		})
	}
}

func TestWorkerNativeCancellationBudgets(t *testing.T) {
	for _, tc := range []struct{ name, ctor, effect string }{
		{"__process_cancel", "ProcessHandle", "Process"},
		{"__stream_cancel_process_source", "StreamSource", "Stream"},
		{"__stream_cancel_process_source", "StreamSource", "Process"},
	} {
		t.Run(tc.name+"_"+tc.effect, func(t *testing.T) {
			m := NewVM(bytecode.NewImage())
			m.Effects = effects.NewEffContext(nil)
			m.Effects.Grant(effects.Capability{Name: "Stream"})
			m.Effects.Grant(effects.Capability{Name: "Process"})
			limit := 0
			m.Effects.SetBudget(effects.NewBudgetContext(map[string]*int{tc.effect: &limit}))
			if _, err := m.callEffectBuiltin(terminalEffectIndex(t, tc.name), []bytecode.Value{bytecode.NewADT(0, tc.ctor, []bytecode.Value{bytecode.NewInt(42)})}); err == nil {
				t.Fatal("exhausted cancellation budget accepted")
			}
		})
	}
}

func TestWorkerNativeSpawnWriteCloseAndCancel(t *testing.T) {
	if !effects.WorkerCancellationSupported() {
		t.Skip("POSIX process-group support required")
	}
	m := NewVM(bytecode.NewImage())
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.Capability{Name: "Process"})
	ctx.Grant(effects.Capability{Name: "Stream"})
	ctx.Process = effects.NewProcessContext()
	ctx.Stream = effects.NewStreamContext()
	m.Effects = ctx
	t.Cleanup(func() {
		if err := ctx.CloseWorkers(); err != nil {
			t.Error(err)
		}
	})
	dispatch := func(name string, args ...bytecode.Value) bytecode.Value {
		t.Helper()
		got, err := m.callEffectBuiltin(terminalEffectIndex(t, name), args)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return got
	}
	handle := dispatch("__process_spawn_process", bytecode.NewString("sleep"), bytecode.NewList([]bytecode.Value{bytecode.NewString("60")}))
	if handle.AsADT().Ctor != "ProcessHandle" {
		t.Fatalf("wrong process handle %v", handle)
	}
	data := bytecode.NewBytes(&bytecode.BytesObj{B: []byte("accepted input\n")})
	if got := dispatch("__process_write_stdin", handle, data); got.AsADT().Ctor != "Ok" {
		t.Fatalf("stdin write: %v", got)
	}
	dispatch("__process_close_stdin", handle)
	if got := dispatch("__process_cancel", handle); got.AsADT().Ctor != "Ok" {
		t.Fatalf("cancel after stdin close: %v", got)
	}
	if got := dispatch("__process_cancel", handle); got.AsADT().Fields[0].AsADT().Ctor != "WorkerHandleInvalid" {
		t.Fatalf("released handle retained: %v", got)
	}
	source := dispatch("__stream_async_exec_process", bytecode.NewString("sleep"), bytecode.NewList([]bytecode.Value{bytecode.NewString("60")}), bytecode.NewString("worker"), bytecode.NewInt(1), bytecode.NewInt(1024))
	if source.AsADT().Ctor != "StreamSource" {
		t.Fatalf("wrong source handle %v", source)
	}
	if got := dispatch("__stream_cancel_process_source", source); got.AsADT().Ctor != "Ok" {
		t.Fatalf("source cancel: %v", got)
	}
	if got := dispatch("__stream_cancel_process_source", source); got.AsADT().Fields[0].AsADT().Ctor != "WorkerHandleInvalid" {
		t.Fatalf("released source retained: %v", got)
	}
}
