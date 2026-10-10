//go:build js

package effects

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/eval"
)

func init() { RegisterOp("Process", "cancelProcess", ProcessCancel) }

// ProcessCancel is explicit about the unavailable JS/WASM subprocess boundary.
func ProcessCancel(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCap("Process"); err != nil {
		return nil, err
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("cancelProcess: expected 1 argument, got %d", len(args))
	}
	return WorkerCancelFailure("WorkerCancelUnsupported", "subprocess cancellation is unavailable on JS/WASM"), nil
}
