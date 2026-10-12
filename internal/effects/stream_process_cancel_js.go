//go:build js

package effects

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/eval"
)

func init() { RegisterOp("Stream", "cancelProcessSource", StreamCancelProcessSource) }

func StreamCancelProcessSource(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("_stream_cancel_process_source: expected one source")
	}
	if err := ctx.RequireCap("Stream"); err != nil {
		return nil, err
	}
	// Stream's evaluator charge scope must not suppress the second effect.
	previous := ctx.SaveAndResetBudgetChargeScope()
	err := ctx.RequireCapWithBudget("Process", "stream.cancelProcessSource")
	ctx.RestoreBudgetChargeScope(previous)
	if err != nil {
		return nil, err
	}
	return WorkerCancelFailure("WorkerCancelUnsupported", "process cancellation is unavailable on JS/WASM"), nil
}
