package effects

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const WorkerShutdownTimeout = 2 * time.Second
const WorkerCooperativeGrace = 250 * time.Millisecond

// OwnedWorker separates termination from joining; shutdown never closes borrowed
// transports or stdin readers. Done closes only after Wait and owned IO tasks join.
type OwnedWorker interface {
	RequestStop() error
	Join(context.Context) error
	Done() <-chan struct{}
	StdinClosing() bool
}

type WorkerCleanupFailure struct {
	Resource int
	Phase    string
	Err      error
}

func (f WorkerCleanupFailure) Error() string {
	return fmt.Sprintf("worker %d %s: %v", f.Resource, f.Phase, f.Err)
}
func (f WorkerCleanupFailure) Unwrap() error { return f.Err }

// WorkerOwner has an admission barrier so a concurrent Start cannot escape the
// shutdown snapshot. Locks protect bookkeeping only, never external cleanup.
type WorkerOwner struct {
	parent         *WorkerOwner
	children       map[*WorkerOwner]struct{}
	mu             sync.Mutex
	closing        bool
	pending        map[*WorkerAdmission]struct{}
	workers        map[OwnedWorker]*WorkerAdmission
	pendingDone    chan struct{}
	closeOnce      sync.Once
	closed         chan struct{}
	closeErr       error
	cleanupSources func() int
	recordReceipt  func(error, time.Duration, int)
	pendingReaders int
	receiptOnce    sync.Once
	elapsed        time.Duration
	closedChildren []*WorkerOwner
}

func newWorkerOwner() *WorkerOwner {
	done := make(chan struct{})
	close(done)
	return &WorkerOwner{children: make(map[*WorkerOwner]struct{}), pending: make(map[*WorkerAdmission]struct{}), workers: make(map[OwnedWorker]*WorkerAdmission), pendingDone: done, closed: make(chan struct{})}
}

// child gives independent requests their own registry while letting host shutdown
// stop all currently active requests. Closing a request never closes its parent.
func (owner *WorkerOwner) child() *WorkerOwner {
	child := newWorkerOwner()
	owner.mu.Lock()
	child.parent = owner
	if owner.closing {
		child.closing = true
	} else {
		owner.children[child] = struct{}{}
	}
	owner.mu.Unlock()
	return child
}

var workerOwnerInit sync.Mutex
var nextWorkerHandle atomic.Int64

// NextWorkerHandleID is process-global and never reuses an ID across registries.
// Handles are not OS PIDs. Exhaustion fails instead of wrapping into authority.
func NextWorkerHandleID() int {
	id := nextWorkerHandle.Add(1)
	if id <= 0 || int64(int(id)) != id {
		panic("worker handle ID space exhausted")
	}
	return int(id)
}

func (ctx *EffContext) executionWorkerOwner() *WorkerOwner {
	workerOwnerInit.Lock()
	defer workerOwnerInit.Unlock()
	if ctx.workerOwner == nil {
		ctx.workerOwner = newWorkerOwner()
	}
	return ctx.workerOwner
}

// WorkerAdmission must be completed or aborted exactly once by the spawning
// handler. The child inherits host cancellation before it is registered.
type WorkerAdmission struct {
	owner  *WorkerOwner
	id     int
	ctx    context.Context
	cancel context.CancelFunc
	once   sync.Once
	err    error
}

func (a *WorkerAdmission) Context() context.Context { return a.ctx }

// SetResourceID correlates shutdown failures with the public owner-local handle.
// Spawn handlers call this after registry insertion and before Complete.
func (a *WorkerAdmission) SetResourceID(id int) {
	a.owner.mu.Lock()
	a.id = id
	a.owner.mu.Unlock()
}

func (ctx *EffContext) BeginWorker() (*WorkerAdmission, error) {
	owner := ctx.executionWorkerOwner()
	parent := ctx.GoCtx
	if parent == nil {
		parent = context.Background()
	}
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.closing {
		return nil, fmt.Errorf("execution owner is shutting down")
	}
	if err := parent.Err(); err != nil {
		return nil, err
	}
	child, cancel := context.WithCancel(parent)
	a := &WorkerAdmission{owner: owner, id: NextWorkerHandleID(), ctx: child, cancel: cancel}
	if len(owner.pending) == 0 {
		owner.pendingDone = make(chan struct{})
	}
	owner.pending[a] = struct{}{}
	return a, nil
}
func (a *WorkerAdmission) finishPendingLocked() {
	delete(a.owner.pending, a)
	if len(a.owner.pending) == 0 {
		close(a.owner.pendingDone)
	}
}
func (a *WorkerAdmission) Abort() {
	a.once.Do(func() { a.cancel(); a.owner.mu.Lock(); a.finishPendingLocked(); a.owner.mu.Unlock() })
}
func (a *WorkerAdmission) Complete(worker OwnedWorker) error {
	a.once.Do(func() {
		if worker == nil {
			a.cancel()
			a.owner.mu.Lock()
			a.finishPendingLocked()
			a.owner.mu.Unlock()
			a.err = fmt.Errorf("nil owned worker")
			return
		}
		a.owner.mu.Lock()
		if !a.owner.closing {
			a.owner.workers[worker] = a
			a.finishPendingLocked()
			a.owner.mu.Unlock()
			return
		}
		a.owner.mu.Unlock()
		stopErr := worker.RequestStop()
		a.cancel()
		bounded, cancel := context.WithTimeout(context.Background(), WorkerShutdownTimeout)
		joinErr := worker.Join(bounded)
		cancel()
		a.err = errors.Join(fmt.Errorf("execution owner is shutting down"), stopErr, joinErr)
		a.owner.mu.Lock()
		if stopErr != nil || joinErr != nil {
			a.owner.closeErr = errors.Join(a.owner.closeErr, WorkerCleanupFailure{a.id, "late admission", errors.Join(stopErr, joinErr)})
		}
		a.finishPendingLocked()
		a.owner.mu.Unlock()
	})
	return a.err
}

// ReleaseWorker is called after all worker tasks have joined, never at stdin EOF.
func (ctx *EffContext) ReleaseWorker(worker OwnedWorker) {
	ctx.executionWorkerOwner().releaseWorker(worker)
}

func (a *WorkerAdmission) ReleaseWorker(worker OwnedWorker) { a.owner.releaseWorker(worker) }

func (owner *WorkerOwner) releaseWorker(worker OwnedWorker) {
	select {
	case <-worker.Done():
	default:
		return
	}
	bounded, cancel := context.WithTimeout(context.Background(), WorkerShutdownTimeout)
	joinErr := worker.Join(bounded)
	cancel()
	owner.mu.Lock()
	a := owner.workers[worker]
	if a != nil {
		if joinErr != nil {
			owner.closeErr = errors.Join(owner.closeErr, WorkerCleanupFailure{a.id, "natural completion", joinErr})
		}
		delete(owner.workers, worker)
	}
	owner.mu.Unlock()
	if a != nil {
		a.cancel()
	}
}

func (owner *WorkerOwner) close(parentDeadline context.Context) error {
	owner.closeOnce.Do(func() {
		started := time.Now()
		bounded, cancel := context.WithTimeout(parentDeadline, WorkerShutdownTimeout)
		defer cancel()
		owner.mu.Lock()
		owner.closing = true
		workers := make(map[OwnedWorker]*WorkerAdmission, len(owner.workers))
		for w, a := range owner.workers {
			workers[w] = a
		}
		pending := make([]*WorkerAdmission, 0, len(owner.pending))
		for a := range owner.pending {
			pending = append(pending, a)
		}
		pendingDone := owner.pendingDone
		children := make([]*WorkerOwner, 0, len(owner.children))
		for child := range owner.children {
			children = append(children, child)
		}
		owner.mu.Unlock()
		for _, a := range pending {
			a.cancel()
		}
		type closeResult struct {
			id  int
			err error
		}
		results := make(chan closeResult, len(workers)+len(children))
		remaining := make(map[int]bool, len(workers))
		for _, a := range workers {
			remaining[a.id] = true
		}
		for _, child := range children {
			go func(child *WorkerOwner) { results <- closeResult{err: child.close(bounded)} }(child)
		}
		for w, a := range workers {
			go func(w OwnedWorker, a *WorkerAdmission) {
				if w.StdinClosing() && a.ctx.Err() == nil {
					timer := time.NewTimer(WorkerCooperativeGrace)
					select {
					case <-w.Done():
					case <-timer.C:
					case <-bounded.Done():
					}
					timer.Stop()
				}
				stopErr := w.RequestStop()
				joinErr := w.Join(bounded)
				var failures []error
				if stopErr != nil {
					failures = append(failures, WorkerCleanupFailure{a.id, "stop", stopErr})
				}
				if joinErr != nil {
					failures = append(failures, WorkerCleanupFailure{a.id, "join", joinErr})
				}
				results <- closeResult{a.id, errors.Join(failures...)}
			}(w, a)
		}
		var failures []error
		for i := 0; i < len(workers)+len(children); i++ {
			select {
			case result := <-results:
				delete(remaining, result.id)
				if result.err != nil {
					failures = append(failures, result.err)
				}
			case <-bounded.Done():
				for id := range remaining {
					failures = append(failures, WorkerCleanupFailure{id, "shutdown deadline", bounded.Err()})
				}
				if len(remaining) == 0 {
					failures = append(failures, WorkerCleanupFailure{0, "owner deadline", bounded.Err()})
				}
				goto finished
			}
		}
		select {
		case <-pendingDone:
		case <-bounded.Done():
			failures = append(failures, WorkerCleanupFailure{0, "admission deadline", bounded.Err()})
		}
	finished:
		owner.mu.Lock()
		owner.closeErr = errors.Join(owner.closeErr, errors.Join(failures...))
		for w, a := range owner.workers {
			select {
			case <-w.Done():
				delete(owner.workers, w)
				a.cancel()
			default:
			}
		}
		cleanupSources := owner.cleanupSources
		owner.mu.Unlock()
		pendingReaders := 0
		if cleanupSources != nil {
			pendingReaders = cleanupSources()
		}
		owner.mu.Lock()
		owner.pendingReaders = pendingReaders
		owner.elapsed = time.Since(started)
		owner.closedChildren = children
		owner.mu.Unlock()
		if owner.parent != nil {
			owner.parent.mu.Lock()
			delete(owner.parent.children, owner)
			owner.parent.mu.Unlock()
		}
		close(owner.closed)
	})
	<-owner.closed
	owner.mu.Lock()
	defer owner.mu.Unlock()
	return owner.closeErr
}

// BindWorkerScope connects source cleanup and receipts before evaluation begins.
// Request preparation may replace Stream, so hosts bind after their prepare hook.
func (ctx *EffContext) BindWorkerScope() {
	workerOwnerInit.Lock()
	if ctx.workerOwner == nil {
		ctx.workerOwner = newWorkerOwner()
	}
	owner, stream, collector := ctx.workerOwner, ctx.Stream, ctx.Trace
	workerOwnerInit.Unlock()
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.closing {
		return
	}
	if stream != nil {
		owner.cleanupSources = stream.CloseSourcesAfterWorkers
	}
	if collector != nil && collector.Enabled() {
		owner.recordReceipt = func(err error, elapsed time.Duration, pending int) {
			scope := "leader-only"
			if WorkerCancellationSupported() {
				scope = "posix-process-group"
			}
			outcome := "joined"
			if err != nil {
				outcome = err.Error()
			}
			collector.RecordEffect("Process", "shutdownWorkers", []string{scope, fmt.Sprintf("elapsed_ms=%d", elapsed.Milliseconds()), fmt.Sprintf("pending_borrowed_readers=%d", pending)}, outcome)
		}
	}
}

// CloseWorkers is mandatory host cleanup, independent of user capability budgets.
// Calling it repeatedly returns the same cleanup outcome and emits one receipt.
func (ctx *EffContext) CloseWorkers() error {
	ctx.BindWorkerScope()
	owner := ctx.executionWorkerOwner()
	err := owner.close(context.Background())
	owner.flushReceipts()
	return err
}

// ResetWorkerScope ends a REPL session scope and creates fresh runtime registries.
// Hosts must serialize reset with evaluation; policy and caller-owned IO remain.
func (ctx *EffContext) ResetWorkerScope() error {
	err := ctx.CloseWorkers()
	workerOwnerInit.Lock()
	ctx.workerOwner = newWorkerOwner()
	if ctx.Process != nil {
		ctx.Process = ctx.Process.Child()
	}
	if ctx.Stream != nil {
		ctx.Stream = ctx.Stream.Child()
	}
	workerOwnerInit.Unlock()
	ctx.BindWorkerScope()
	return err
}

// Record completed receipts on the host cleanup goroutine. The collector
// synchronizes with any evaluator still running during host cancellation.
func (owner *WorkerOwner) flushReceipts() {
	select {
	case <-owner.closed:
	default:
		return
	}
	owner.receiptOnce.Do(func() {
		owner.mu.Lock()
		children := owner.closedChildren
		record, err, elapsed, pending := owner.recordReceipt, owner.closeErr, owner.elapsed, owner.pendingReaders
		owner.mu.Unlock()
		for _, child := range children {
			child.flushReceipts()
		}
		if record != nil {
			record(err, elapsed, pending)
		}
	})
}
