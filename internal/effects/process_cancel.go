//go:build !js

package effects

import (
	"context"
	"errors"
	"fmt"

	"github.com/sunholo-data/ailang/internal/eval"
)

func init() { RegisterOp("Process", "cancelProcess", ProcessCancel) }

// ProcessCancel resolves authority in the current owner's registry, never a PID.
func ProcessCancel(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCap("Process"); err != nil {
		return nil, err
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("cancelProcess: expected 1 argument, got %d", len(args))
	}
	if !WorkerCancellationSupported() {
		return WorkerCancelFailure("WorkerCancelUnsupported", "process-group cancellation is unavailable on this platform"), nil
	}
	id, err := extractProcessHandleID(args[0])
	if err != nil {
		return WorkerCancelFailure("WorkerHandleInvalid", err.Error()), nil
	}
	if ctx.Process == nil {
		return WorkerCancelFailure("WorkerHandleInvalid", "process owner is unavailable"), nil
	}
	mp, ok := ctx.Process.GetManagedProcess(id)
	if !ok {
		return WorkerCancelFailure("WorkerHandleInvalid", "process handle is stale, foreign or unknown"), nil
	}
	joinCtx, cancel := context.WithTimeout(context.Background(), WorkerShutdownTimeout)
	defer cancel()
	stopErr := mp.RequestStop()
	joinErr := mp.Join(joinCtx)
	if errors.Is(joinErr, context.DeadlineExceeded) {
		return WorkerCancelTimeout(), nil
	}
	if stopErr != nil {
		return WorkerCancelFailure("WorkerCancelFailed", stopErr.Error()), nil
	}
	if joinErr != nil {
		return WorkerCancelFailure("WorkerCancelFailed", joinErr.Error()), nil
	}
	ctx.Process.ReleaseManagedProcess(id)
	ctx.ReleaseWorker(mp)
	return WorkerCancelOK(), nil
}
