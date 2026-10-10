package effects

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/trace"
)

type ownedTestWorker struct {
	done        chan struct{}
	once        sync.Once
	failure     error
	cooperative bool
}

func (w *ownedTestWorker) RequestStop() error { w.once.Do(func() { close(w.done) }); return w.failure }
func (w *ownedTestWorker) Join(ctx context.Context) error {
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (w *ownedTestWorker) Done() <-chan struct{} { return w.done }
func (w *ownedTestWorker) StdinClosing() bool    { return w.cooperative }

func TestWorkerOwnerStopsAllAndRejectsNewAdmission(t *testing.T) {
	ctx := NewEffContext(nil)
	workers := make([]*ownedTestWorker, 20)
	for i := range workers {
		workers[i] = &ownedTestWorker{done: make(chan struct{})}
		a, err := ctx.BeginWorker()
		if err != nil {
			t.Fatal(err)
		}
		if err = a.Complete(workers[i]); err != nil {
			t.Fatal(err)
		}
	}
	if err := ctx.CloseWorkers(); err != nil {
		t.Fatal(err)
	}
	for _, w := range workers {
		select {
		case <-w.done:
		default:
			t.Fatal("worker still active")
		}
	}
	if _, err := ctx.BeginWorker(); err == nil {
		t.Fatal("shutdown admitted new worker")
	}
	if err := ctx.CloseWorkers(); err != nil {
		t.Fatal(err)
	}
}

func TestWorkerOwnerLateAdmissionIsJoined(t *testing.T) {
	ctx := NewEffContext(nil)
	a, err := ctx.BeginWorker()
	if err != nil {
		t.Fatal(err)
	}
	closed := make(chan error, 1)
	go func() { closed <- ctx.CloseWorkers() }()
	select {
	case <-a.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("pending child context not cancelled")
	}
	w := &ownedTestWorker{done: make(chan struct{})}
	if err = a.Complete(w); err == nil {
		t.Fatal("late admission accepted")
	}
	if err = <-closed; err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.done:
	default:
		t.Fatal("rejected child still active")
	}
}

func TestWorkerOwnerReportsStopFailure(t *testing.T) {
	ctx := NewEffContext(nil)
	a, _ := ctx.BeginWorker()
	if err := a.Complete(&ownedTestWorker{done: make(chan struct{}), failure: errors.New("injected kill failure")}); err != nil {
		t.Fatal(err)
	}
	if err := ctx.CloseWorkers(); err == nil || !strings.Contains(err.Error(), "injected kill failure") {
		t.Fatalf("cleanup failure hidden: %v", err)
	}
}

func TestWorkerOwnerBudgetViewsShareAndCloneIsIndependent(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Process = NewProcessContext()
	ctx.Process.Allowlist = map[string]string{"helper": "/pinned/helper"}
	ctx.Process.HasAllowlist = true
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowLocalhost = true
	scoped := ctx.WithBudget(nil)
	a, _ := scoped.BeginWorker()
	w := &ownedTestWorker{done: make(chan struct{})}
	if err := a.Complete(w); err != nil {
		t.Fatal(err)
	}
	child := ctx.Clone().(*EffContext)
	if child.Process == ctx.Process || child.Stream == ctx.Stream {
		t.Fatal("independent request shares registries")
	}
	if child.Process.Allowlist["helper"] != "/pinned/helper" || !child.Process.HasAllowlist || !child.Stream.AllowLocalhost {
		t.Fatal("clone lost policy")
	}
	if err := child.CloseWorkers(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.done:
		t.Fatal("closing request killed original worker")
	default:
	}
	if err := ctx.CloseWorkers(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.done:
	default:
		t.Fatal("budget view has different owner")
	}
}

func TestWorkerAdmissionParentContextIsPreserved(t *testing.T) {
	ctx := NewEffContext(nil)
	parent, cancel := context.WithCancel(context.Background())
	ctx.GoCtx = parent
	a, err := ctx.BeginWorker()
	if err != nil {
		t.Fatal(err)
	}
	defer a.Abort()
	cancel()
	select {
	case <-a.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("host context not propagated")
	}
}

func TestWorkerOwnerParentClosesActiveRequests(t *testing.T) {
	parent := NewEffContext(nil)
	child := parent.Clone().(*EffContext)
	a, _ := child.BeginWorker()
	w := &ownedTestWorker{done: make(chan struct{})}
	if err := a.Complete(w); err != nil {
		t.Fatal(err)
	}
	if err := parent.CloseWorkers(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.done:
	default:
		t.Fatal("active request escaped engine shutdown")
	}
	if _, err := child.BeginWorker(); err == nil {
		t.Fatal("closed request admitted worker")
	}
	if _, err := parent.Clone().(*EffContext).BeginWorker(); err == nil {
		t.Fatal("closed parent admitted new request")
	}
}

type stalledOwnedWorker struct{ done chan struct{} }

func (w *stalledOwnedWorker) RequestStop() error             { return nil }
func (w *stalledOwnedWorker) Join(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }
func (w *stalledOwnedWorker) Done() <-chan struct{}          { return w.done }
func (w *stalledOwnedWorker) StdinClosing() bool             { return false }

func TestWorkerOwnerOneDeadlineAcrossTwentyRequests(t *testing.T) {
	parent := NewEffContext(nil)
	for i := 0; i < 20; i++ {
		child := parent.Clone().(*EffContext)
		a, _ := child.BeginWorker()
		if err := a.Complete(&stalledOwnedWorker{done: make(chan struct{})}); err != nil {
			t.Fatal(err)
		}
	}
	start := time.Now()
	err := parent.CloseWorkers()
	if err == nil || !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("missing deadline receipt: %v", err)
	}
	if elapsed := time.Since(start); elapsed > WorkerShutdownTimeout+500*time.Millisecond {
		t.Fatalf("serial deadlines: %s", elapsed)
	}
}

func TestWorkerOwnerResetReplacesClosedScope(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Process = NewProcessContext()
	ctx.Process.HasAllowlist = true
	ctx.Process.Allowlist = map[string]string{"tool": "/pinned/tool"}
	a, _ := ctx.BeginWorker()
	w := &ownedTestWorker{done: make(chan struct{})}
	if err := a.Complete(w); err != nil {
		t.Fatal(err)
	}
	if err := ctx.ResetWorkerScope(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.done:
	default:
		t.Fatal("reset left old worker")
	}
	next, err := ctx.BeginWorker()
	if err != nil {
		t.Fatal(err)
	}
	next.Abort()
	if ctx.Process.Allowlist["tool"] != "/pinned/tool" {
		t.Fatal("reset lost pinned policy")
	}
	if err := ctx.CloseWorkers(); err != nil {
		t.Fatal(err)
	}
}

func TestWorkerOwnerStalledStopRemainsBoundedAndIdentified(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Stream = NewStreamContext()
	source := &stalledOwnedSource{EventSource: newMockSource("stalled", 0, 1), entered: make(chan struct{}), release: make(chan struct{}), completed: make(chan struct{})}
	id := ctx.Stream.AcquireSource(source)
	a, err := ctx.BeginWorker()
	if err != nil {
		t.Fatal(err)
	}
	a.SetResourceID(id)
	if err := a.Complete(source); err != nil {
		t.Fatal(err)
	}
	defer func() { close(source.release); <-source.completed }()
	start := time.Now()
	err = ctx.CloseWorkers()
	if err == nil {
		t.Fatal("stalled stop reported joined")
	}
	var failure WorkerCleanupFailure
	if !errors.As(err, &failure) || failure.Resource != id || !strings.Contains(failure.Phase, "deadline") {
		t.Fatalf("missing actual handle deadline: %v", err)
	}
	if time.Since(start) > WorkerShutdownTimeout+500*time.Millisecond {
		t.Fatal("post-deadline cleanup waited for stalled stop")
	}
	if _, ok := ctx.Stream.GetSource(id); ok {
		t.Fatal("post-deadline source data still registered")
	}
}

type failedJoinOwnedWorker struct{ ownedTestWorker }

func (w *failedJoinOwnedWorker) Join(ctx context.Context) error {
	if err := w.ownedTestWorker.Join(ctx); err != nil {
		return err
	}
	return errors.New("injected wait failure")
}
func TestWorkerOwnerReleasedNaturalFailureIsNotLost(t *testing.T) {
	ctx := NewEffContext(nil)
	a, _ := ctx.BeginWorker()
	w := &failedJoinOwnedWorker{ownedTestWorker{done: make(chan struct{})}}
	if err := a.Complete(w); err != nil {
		t.Fatal(err)
	}
	if err := w.RequestStop(); err != nil {
		t.Fatal(err)
	}
	a.ReleaseWorker(w)
	if err := ctx.CloseWorkers(); err == nil || !strings.Contains(err.Error(), "injected wait failure") {
		t.Fatalf("released failure disappeared: %v", err)
	}
}

func TestWorkerOwnerReceiptsPreserveNoWorkerTraces(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Trace = trace.NewCollector()
	child := ctx.Clone().(*EffContext)
	if err := child.CloseWorkers(); err != nil {
		t.Fatal(err)
	}
	if err := ctx.CloseWorkers(); err != nil {
		t.Fatal(err)
	}
	if events := ctx.Trace.Events(); len(events) != 0 {
		t.Fatalf("no-worker execution gained cleanup events: %v", events)
	}
}

func TestWorkerOwnerReceiptsRetainReleasedWorkerHistory(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Trace = trace.NewCollector()
	a, err := ctx.BeginWorker()
	if err != nil {
		t.Fatal(err)
	}
	w := &ownedTestWorker{done: make(chan struct{})}
	if err := a.Complete(w); err != nil {
		t.Fatal(err)
	}
	if err := w.RequestStop(); err != nil {
		t.Fatal(err)
	}
	a.ReleaseWorker(w)
	for i := 0; i < 2; i++ {
		if err := ctx.CloseWorkers(); err != nil {
			t.Fatal(err)
		}
	}
	if events := ctx.Trace.Events(); len(events) != 1 {
		t.Fatalf("completed worker must retain exactly one cleanup receipt: %v", events)
	}
}
