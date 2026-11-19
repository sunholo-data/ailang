package runner

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/runtime"
)

// vmTraceLabelGuard refuses the bytecode VM for a traced run of a program
// with labelled values (M-TRACE-LABEL-AWARE).
//
// The evaluator redacts the trace for the dynamic extent of every call the
// static IFC pass marked (core.App.TraceRedact). The bytecode compiler does
// not carry that mark, so on the VM those calls' effect arguments — a secret
// written to a file, sent in a request body — would be recorded verbatim.
// Refusing here makes --bytecode fall back to the evaluator, and makes
// --strict-bytecode fail loudly, rather than writing the value to the trace.
// Untraced runs, and traced programs with no labels, are unaffected.
func vmTraceLabelGuard(rt *runtime.ModuleRuntime, params ModuleExecParams) error {
	ctx, ok := rt.GetEvaluator().GetEffContext().(*effects.EffContext)
	if !ok || !ctx.HasTraceCollector() || params.PipelineResult == nil {
		return nil
	}
	for id, mod := range params.PipelineResult.Modules {
		if mod != nil && core.HasTraceRedactions(mod.Core) {
			return fmt.Errorf("module %s passes IFC-labelled values and a trace is being recorded; the bytecode VM does not redact them from the trace (M-TRACE-LABEL-AWARE)", id)
		}
	}
	if core.HasTraceRedactions(params.PipelineResult.Artifacts.Core) {
		return fmt.Errorf("the entry module passes IFC-labelled values and a trace is being recorded; the bytecode VM does not redact them from the trace (M-TRACE-LABEL-AWARE)")
	}
	return nil
}
