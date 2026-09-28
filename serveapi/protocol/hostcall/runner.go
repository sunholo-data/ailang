// Package hostcall bounds calls from AILANG protocol handlers into host code:
// each call gets a deadline, and at most a fixed number run at once. It is
// protocol-neutral (A2A and MCP handlers share one Runner) and imports only the
// standard library and serveapi/protocol, so a zero-cloud consumer can link it.
package hostcall

import (
	"context"
	"fmt"
	"time"

	"github.com/sunholo-data/ailang/serveapi/protocol"
)

// Runner bounds handler wait time and concurrently started host calls.
// A slot remains occupied until host code actually returns, even after timeout.
type Runner struct {
	timeout time.Duration
	slots   chan struct{}
}

// New constructs a Runner from effective positive limits.
func New(timeout time.Duration, maxConcurrent int) (*Runner, error) {
	if timeout <= 0 {
		return nil, fmt.Errorf("callback timeout must be positive")
	}
	if maxConcurrent <= 0 {
		return nil, fmt.Errorf("maximum concurrent callbacks must be positive")
	}
	return &Runner{timeout: timeout, slots: make(chan struct{}, maxConcurrent)}, nil
}

type result[T any] struct {
	value T
	err   error
}

// Run executes callback with a bounded context. In-process callbacks cannot be
// forcibly terminated; a non-cooperative callback keeps its slot.
func Run[T any](ctx context.Context, runner *Runner, callback func(context.Context) (T, error)) (T, error) {
	var zero T
	callCtx, cancel := context.WithTimeout(ctx, runner.timeout)
	defer cancel()

	select {
	case runner.slots <- struct{}{}:
	case <-callCtx.Done():
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		return zero, protocol.ErrCallbackCapacity
	}

	done := make(chan result[T], 1)
	go func() {
		defer func() { <-runner.slots }()
		value, err := callback(callCtx)
		done <- result[T]{value: value, err: err}
	}()

	select {
	case completed := <-done:
		return completed.value, completed.err
	case <-callCtx.Done():
		return zero, callCtx.Err()
	}
}
