//go:build js

package effects

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/eval"
)

// StreamAsyncExecProcess is unavailable on js/wasm (no subprocess support).
func StreamAsyncExecProcess(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	return nil, fmt.Errorf("_stream_async_exec_process: subprocess streaming not available on js/wasm")
}
