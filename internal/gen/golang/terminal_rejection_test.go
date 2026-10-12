package golang

import (
	"github.com/sunholo-data/ailang/internal/core"
	"strings"
	"testing"
)

func TestTerminalSurfaceRejectedBeforeExecution(t *testing.T) {
	for _, name := range []string{"_terminal_info", "_terminal_withTerminal", "_terminal_readEvent", "_io_readLineOpt"} {
		prog := &core.Program{Decls: []core.CoreExpr{&core.Let{Name: "main", Value: &core.Lambda{Params: []string{"u"}, Body: &core.App{Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: name}}, Args: []core.CoreExpr{&core.Var{Name: "u"}}}}, Body: &core.Var{Name: "main"}}}}
		_, err := New("terminal").Generate(prog)
		if err == nil || !strings.Contains(err.Error(), "not supported by Go codegen") {
			t.Errorf("%s: got %v", name, err)
		}
	}
}
