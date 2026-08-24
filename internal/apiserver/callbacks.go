package apiserver

import (
	"context"
	"fmt"
	"time"
)

// CallbackRunner bounds handler wait time and concurrently started host calls.
// A slot remains occupied until host code actually returns, even after timeout.
type CallbackRunner struct {
	timeout time.Duration
	slots   chan struct{}
}

// NewCallbackRunner constructs a callback runner from effective positive limits.
func NewCallbackRunner(timeout time.Duration, maxConcurrent int) (*CallbackRunner, error) {
	if timeout <= 0 {
		return nil, fmt.Errorf("callback timeout must be positive")
	}
	if maxConcurrent <= 0 {
		return nil, fmt.Errorf("maximum concurrent callbacks must be positive")
	}
	return &CallbackRunner{timeout: timeout, slots: make(chan struct{}, maxConcurrent)}, nil
}

type callbackResult[T any] struct {
	value T
	err   error
}

// RunCallback executes callback with a bounded context. In-process callbacks
// cannot be forcibly terminated; a non-cooperative callback keeps its slot.
func RunCallback[T any](ctx context.Context, runner *CallbackRunner, callback func(context.Context) (T, error)) (T, error) {
	var zero T
	callCtx, cancel := context.WithTimeout(ctx, runner.timeout)
	defer cancel()

	select {
	case runner.slots <- struct{}{}:
	case <-callCtx.Done():
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		return zero, ErrCallbackCapacity
	}

	result := make(chan callbackResult[T], 1)
	go func() {
		defer func() { <-runner.slots }()
		value, err := callback(callCtx)
		result <- callbackResult[T]{value: value, err: err}
	}()

	select {
	case completed := <-result:
		return completed.value, completed.err
	case <-callCtx.Done():
		return zero, callCtx.Err()
	}
}
