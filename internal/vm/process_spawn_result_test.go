package vm

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/bytecode/compiler"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

func TestProcessSpawnResultNative(t *testing.T) {
	const name = "__process_spawn_result"
	img, err := compiler.Compile(&stmt.Program{FuncDecls: []stmt.FuncDecl{{Name: "spawn", Return: stmt.BuiltinCall{Name: name, Args: []stmt.Expr{stmt.LitUnit{}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	if img.Prototypes[0].EvalOnly {
		t.Fatal("spawn result requires evaluator fallback")
	}
	m := NewVM(bytecode.NewImage())
	m.Effects = effects.NewEffContext(nil)
	index := terminalEffectIndex(t, name)
	args := []bytecode.Value{bytecode.NewString("absent"), bytecode.NewList(nil)}
	if _, err := m.callEffectBuiltin(index, args); err == nil {
		t.Fatal("missing capability accepted")
	}
	m.Effects.Grant(effects.NewCapability("Process"))
	m.Effects.Process = effects.NewProcessContext()
	m.Effects.Process.HasAllowlist = true
	v, err := m.callEffectBuiltin(index, args)
	if err != nil {
		t.Fatal(err)
	}
	if v.AsADT().Ctor != "Err" || v.AsADT().Fields[0].AsADT().Ctor != "NotAllowed" {
		t.Fatalf("unexpected result %v", v)
	}
	m.Effects.Process.Allowlist = map[string]string{"absent": ""}
	v, err = m.callEffectBuiltin(index, args)
	if err != nil {
		t.Fatal(err)
	}
	inner := v.AsADT().Fields[0].AsADT()
	if inner.Ctor != "NotFound" || inner.Tag != 1 {
		t.Fatalf("unexpected NotFound %v", inner)
	}
}
