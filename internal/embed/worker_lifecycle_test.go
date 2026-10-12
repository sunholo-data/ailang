package embed

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/effects"
)

type embeddingOwnedWorker struct {
	done chan struct{}
	once sync.Once
}

func (w *embeddingOwnedWorker) RequestStop() error { w.once.Do(func() { close(w.done) }); return nil }
func (w *embeddingOwnedWorker) Join(ctx context.Context) error {
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (w *embeddingOwnedWorker) Done() <-chan struct{} { return w.done }
func (w *embeddingOwnedWorker) StdinClosing() bool    { return false }
func TestEngineWorkerLifecycle_CloseOwnsCloneButBorrowsBase(t *testing.T) {
	base := effects.NewEffContext(nil)
	engine := New(t.TempDir())
	engine.SetEffContext(base)
	owned := engine.runtime.GetEvaluator().GetEffContext().(*effects.EffContext)
	if owned == base {
		t.Fatal("engine retained caller context instead of owning a clone")
	}
	worker := &embeddingOwnedWorker{done: make(chan struct{})}
	admission, err := owned.BeginWorker()
	if err != nil {
		t.Fatal(err)
	}
	if err = admission.Complete(worker); err != nil {
		t.Fatal(err)
	}
	if err = engine.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-worker.done:
	default:
		t.Fatal("engine worker survived Close")
	}
	next, err := base.BeginWorker()
	if err != nil {
		t.Fatalf("engine closed borrowed base: %v", err)
	}
	next.Abort()
}
func TestEngineWorkerLifecycle_CloseDoesNotWaitForModuleMutex(t *testing.T) {
	engine := New(t.TempDir())
	engine.SetEffContext(effects.NewEffContext(nil))
	engine.mu.Lock()
	done := make(chan error, 1)
	go func() { done <- engine.Close() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(100 * time.Millisecond):
		engine.mu.Unlock()
		<-done
		t.Fatal("shutdown waited for an active initializer's module mutex")
	}
	engine.mu.Unlock()
}

func TestEngineWorkerLifecycle_CloseWhileCallBlocked(t *testing.T) {
	root := t.TempDir()
	source := "module worker_call\nimport std/ai (call)\nexport func work() -> string ! {AI} { call(\"blocked-host-stub\") }\n"
	if err := os.WriteFile(filepath.Join(root, "worker_call.ail"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	base := effects.NewEffContext(nil)
	base.Grant(effects.NewCapability("AI"))
	engine := New(root)
	engine.SetEffContext(base)
	defer func() { _ = engine.Close() }()
	started := make(chan struct{})
	release := make(chan struct{})
	returned := make(chan error, 1)
	worker := &embeddingOwnedWorker{done: make(chan struct{})}
	base.AI = effects.NewAIContext(&embeddingBlockedAI{AIHandler: effects.NewStubAIHandler(), entered: started, release: release})
	engine.SetEffContext(base)
	go func() {
		_, err := engine.CallPrepared(func(value interface{}) {
			ctx := value.(*effects.EffContext)
			admission, err := ctx.BeginWorker()
			if err != nil {
				panic(err)
			}
			if err = admission.Complete(worker); err != nil {
				panic(err)
			}
		}, "worker_call", "work")
		returned <- err
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("call did not reach barrier")
	}
	closed := make(chan error, 1)
	go func() { closed <- engine.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		close(release)
		t.Fatal("Close waited for blocked application callback")
	}
	select {
	case <-worker.Done():
	default:
		t.Fatal("active request worker survived engine close")
	}
	if _, err := engine.Call("worker_call", "work"); err == nil {
		t.Fatal("closed engine admitted a new call")
	}
	next, err := base.BeginWorker()
	if err != nil {
		t.Fatal("borrowed caller scope was closed")
	}
	next.Abort()
	close(release)
	select {
	case err := <-returned:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("released call did not finish")
	}
}

// Blocks through the real AI effect dispatch without a provider request.
// Engine.Close stops request-owned workers while this host callback remains
// blocked; in-process AI abort remains the separate #231 boundary.
type embeddingBlockedAI struct {
	effects.AIHandler
	entered chan struct{}
	release <-chan struct{}
}

func (h *embeddingBlockedAI) Call(string) (string, error) {
	close(h.entered)
	<-h.release
	return "completed", nil
}
