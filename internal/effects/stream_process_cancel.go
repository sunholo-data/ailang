//go:build !js

package effects

import (
	"context"
	"errors"
	"fmt"

	"github.com/sunholo-data/ailang/internal/eval"
)

func init() { RegisterOp("Stream", "cancelProcessSource", StreamCancelProcessSource) }

// StreamCancelProcessSource cancels only an owned process source. Borrowed stdin
// readers and connection adapters are deliberately outside this operation.
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
	if !WorkerCancellationSupported() {
		return WorkerCancelFailure("WorkerCancelUnsupported", "process-tree cancellation is supported on macOS and Linux"), nil
	}
	id, err := extractSourceID(args[0])
	if err != nil {
		return WorkerCancelFailure("WorkerHandleInvalid", err.Error()), nil
	}
	if ctx.Stream == nil {
		return WorkerCancelFailure("WorkerHandleInvalid", "no source registry"), nil
	}
	source, ok := ctx.Stream.GetSource(id)
	if !ok {
		return WorkerCancelFailure("WorkerHandleInvalid", "source is stale or belongs to another execution"), nil
	}
	worker, ok := source.(*processSource)
	if !ok {
		return WorkerCancelFailure("WorkerCancelUnsupported", "source is not an owned process"), nil
	}
	stopErr := worker.RequestStop()
	deadline, cancel := context.WithTimeout(context.Background(), WorkerShutdownTimeout)
	defer cancel()
	joinErr := worker.Join(deadline)
	if errors.Is(joinErr, context.DeadlineExceeded) {
		return WorkerCancelTimeout(), nil
	}
	if err := errors.Join(stopErr, joinErr); err != nil {
		return WorkerCancelFailure("WorkerCancelFailed", err.Error()), nil
	}
	ctx.Stream.ReleaseSource(id)
	ctx.ReleaseWorker(worker)
	return WorkerCancelOK(), nil
}
