package effects

import (
	"context"
	"os"
	"sync"
	"testing"
)

type terminalSignalWorker struct {
	done chan struct{}
	once sync.Once
}

func (w *terminalSignalWorker) RequestStop() error { w.once.Do(func() { close(w.done) }); return nil }
func (w *terminalSignalWorker) Join(ctx context.Context) error {
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (w *terminalSignalWorker) Done() <-chan struct{} { return w.done }
func (w *terminalSignalWorker) StdinClosing() bool    { return false }
func TestTerminalWorkerSignal_CleanupBeforeExitHook(t *testing.T) {
	ctx := NewEffContext(nil)
	worker := &terminalSignalWorker{done: make(chan struct{})}
	admission, err := ctx.BeginWorker()
	if err != nil {
		t.Fatal(err)
	}
	if err = admission.Complete(worker); err != nil {
		t.Fatal(err)
	}
	called := false
	ctx.TerminalSignalExit = func(code int) {
		called = true
		if code != 130 {
			t.Error(code)
		}
		select {
		case <-worker.done:
		default:
			t.Error("signal exit hook ran before worker cleanup")
		}
	}
	(&terminalSession{ctx: ctx}).terminateSignal(os.Interrupt)
	if !called {
		t.Fatal("exit hook was not called")
	}
}
