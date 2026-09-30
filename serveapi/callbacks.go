package serveapi

import (
	"context"
	"time"

	"github.com/sunholo-data/ailang/serveapi/protocol/hostcall"
)

// callbackRunner is the one host-call runner, shared by the A2A and MCP handlers.
// It lives in serveapi/protocol/hostcall so the stdlib-only MCP dispatcher can use it.
type callbackRunner = hostcall.Runner

func newCallbackRunner(timeout time.Duration, maxConcurrent int) (*callbackRunner, error) {
	return hostcall.New(timeout, maxConcurrent)
}

func runCallback[T any](ctx context.Context, runner *callbackRunner, callback func(context.Context) (T, error)) (T, error) {
	return hostcall.Run(ctx, runner, callback)
}
