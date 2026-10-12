package golang

import (
	"github.com/sunholo-data/ailang/internal/core"
	"strings"
	"testing"
)

func TestWorkerCancellationRejectedBeforeExecution(t *testing.T) {
	for _, ref := range []core.GlobalRef{
		{Module: "$builtin", Name: "_process_cancel"},
		{Module: "$builtin", Name: "_stream_cancel_process_source"},
		{Module: "std/process", Name: "cancelProcess"},
		{Module: "std/stream", Name: "cancelProcessSource"},
	} {
		prog := &core.Program{Decls: []core.CoreExpr{&core.Let{Name: "main", Value: &core.Lambda{Params: []string{"u"}, Body: &core.App{Func: &core.VarGlobal{Ref: ref}, Args: []core.CoreExpr{&core.Var{Name: "u"}}}}, Body: &core.Var{Name: "main"}}}}
		if _, err := New("worker").Generate(prog); err == nil || !strings.Contains(err.Error(), "not supported by Go codegen") {
			t.Errorf("%s.%s: got %v", ref.Module, ref.Name, err)
		}
	}
}
