package repl

import (
	"bytes"
	"context"
	"sync"
	"testing"
)

type replOwnedWorker struct {
	done chan struct{}
	once sync.Once
}

func (w *replOwnedWorker) RequestStop() error { w.once.Do(func() { close(w.done) }); return nil }
func (w *replOwnedWorker) Join(ctx context.Context) error {
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (w *replOwnedWorker) Done() <-chan struct{} { return w.done }
func (w *replOwnedWorker) StdinClosing() bool    { return false }
func TestREPLWorkerLifecycle_ResetAndQuit(t *testing.T) {
	r := New()
	var output bytes.Buffer
	for _, command := range []string{":reset", ":quit"} {
		admission, err := r.effContext.BeginWorker()
		if err != nil {
			t.Fatal(err)
		}
		worker := &replOwnedWorker{done: make(chan struct{})}
		if err = admission.Complete(worker); err != nil {
			t.Fatal(err)
		}
		r.HandleCommand(command, &output)
		select {
		case <-worker.done:
		default:
			t.Fatalf("worker survived %s", command)
		}
	}
}
