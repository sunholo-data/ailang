package compiler

import (
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"testing"
)

func TestCompileTerminalEffectsNative(t *testing.T) {
	for _, name := range []string{"__terminal_info", "__terminal_withTerminal", "__terminal_readEvent", "__io_readLineOpt", "__io_println"} {
		img, err := Compile(&stmt.Program{FuncDecls: []stmt.FuncDecl{{Name: "f", Exported: true, Return: stmt.BuiltinCall{Name: name, Args: []stmt.Expr{stmt.LitInt{Value: 1}}}}}})
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range img.Prototypes {
			if p.Name == "f" && p.EvalOnly {
				t.Errorf("%s became evaluator-only: %s", name, p.EvalReason)
			}
		}
	}
}
