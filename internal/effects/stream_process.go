//go:build !js

package effects

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/proctree"
	"github.com/sunholo-data/ailang/internal/runlife"
)

// processSource owns its child process group, stdout pipe and reader. Selection
// borrows the source; only cancellation or the execution owner stops it.
type processSource struct {
	name         string
	priority     int
	ch           chan streamEvent
	done         chan struct{} // delivery/read cancellation, not completion
	completed    chan struct{}
	readerDone   chan struct{}
	once         sync.Once
	groupOnce    sync.Once
	killGroup    func(int) error
	cmd          *exec.Cmd
	stdout       *os.File
	cancel       context.CancelFunc
	mu           sync.Mutex
	stopErr      error
	joinErr      error
	lifetime     *runlife.Child
	stopDeadline time.Time
}

// NewProcessSource starts one owned process group and one Wait goroutine.
// A private pipe keeps Wait from closing stdout before its final bytes drain.
func NewProcessSource(parentCtx context.Context, cmdPath string, args []string, name string, priority int, chunkSize int) (EventSource, error) {
	return newProcessSource(parentCtx, cmdPath, args, name, priority, chunkSize, proctree.KillGroup)
}

func newProcessSource(parentCtx context.Context, cmdPath string, args []string, name string, priority int, chunkSize int, killGroup func(int) error) (EventSource, error) {
	if chunkSize <= 0 {
		return nil, fmt.Errorf("chunkSize must be positive, got %d", chunkSize)
	}
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithCancel(parentCtx)
	stdout, childStdout, err := os.Pipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd := exec.CommandContext(ctx, cmdPath, args...)
	proctree.Configure(cmd)
	ps := &processSource{
		name: name, priority: priority, ch: make(chan streamEvent, 100),
		done: make(chan struct{}), completed: make(chan struct{}), readerDone: make(chan struct{}),
		cmd: cmd, stdout: stdout, cancel: cancel, killGroup: killGroup,
	}
	lifetime, err := runlife.Prepare(cmd, nil)
	if err != nil {
		stdout.Close()
		childStdout.Close()
		cancel()
		return nil, fmt.Errorf("prepare runtime lifetime: %w", err)
	}
	ps.lifetime = lifetime
	cmd.Stdout = childStdout
	cmd.Cancel = ps.RequestStop // raw termination errors remain owned until checked Join
	if err := cmd.Start(); err != nil {
		if lifetime != nil {
			lifetime.Abort()
		}
		_ = stdout.Close()
		_ = childStdout.Close()
		cancel()
		return nil, fmt.Errorf("start process: %w", err)
	}
	if lifetime != nil {
		lifetime.AfterStart()
	}
	_ = childStdout.Close() // only child writers now keep the reader open
	go ps.readLoop(chunkSize)
	go ps.waitLoop()
	return ps, nil
}

func (ps *processSource) Name() string               { return ps.name }
func (ps *processSource) Priority() int              { return ps.priority }
func (ps *processSource) Events() <-chan streamEvent { return ps.ch }
func (ps *processSource) Done() <-chan struct{}      { return ps.completed }
func (ps *processSource) StdinClosing() bool         { return false }

// RequestStop interrupts pipe reads and blocked delivery. Group termination
// has one shared attempt, so natural completion cannot trigger later signals.
func (ps *processSource) RequestStop() error {
	bounded, cancel := context.WithTimeout(context.Background(), WorkerShutdownTimeout)
	defer cancel()
	return ps.RequestStopContext(bounded)
}

func (ps *processSource) RequestStopContext(deadline context.Context) error {
	ps.once.Do(func() {
		close(ps.done)
		ps.mu.Lock()
		ps.stopDeadline, _ = deadline.Deadline()
		ps.mu.Unlock()
		if ps.lifetime != nil {
			ps.lifetime.RequestStop(deadline, func() error { ps.finishGroup(false); ps.mu.Lock(); defer ps.mu.Unlock(); return ps.stopErr })
		} else {
			ps.finishGroup(false)
			ps.cancel()
		}
		if ps.lifetime == nil {
			_ = ps.stdout.Close()
		}
	})
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return immediateWorkerStopError(ps.stopErr)
}

// finishGroup performs one checked termination attempt shared by context
// cancellation, explicit cancellation and natural leader completion.
func (ps *processSource) finishGroup(natural bool) {
	ps.groupOnce.Do(func() {
		var err error
		if WorkerCancellationSupported() {
			err = ps.killGroup(ps.cmd.Process.Pid)
		} else if !natural {
			err = ps.cmd.Process.Kill()
			if errors.Is(err, os.ErrProcessDone) {
				err = nil
			}
		}
		ps.mu.Lock()
		ps.stopErr = err
		ps.mu.Unlock()
	})
}

// Join succeeds only after the direct child is reaped and the reader has joined.
func (ps *processSource) Join(ctx context.Context) error {
	select {
	case <-ps.completed:
		if err := ctx.Err(); err != nil {
			return err
		}
		ps.mu.Lock()
		defer ps.mu.Unlock()
		return errors.Join(proctree.ResolveGroupError(ps.cmd.Process.Pid, ps.stopErr), ps.joinErr)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close follows EventSource's signal-only contract. The execution supervisor
// and public cancellation operation perform the checked, deadline-bound join.
func (ps *processSource) Close() { _ = ps.RequestStop() }

func (ps *processSource) waitLoop() {
	err := ps.cmd.Wait() // exactly one owner, including natural EOF
	if ps.lifetime != nil {
		ps.lifetime.AfterWait()
		ctx, cancel := ps.receiptContext()
		receiptErr := ps.lifetime.Join(ctx)
		cancel()
		ps.mu.Lock()
		ps.joinErr = errors.Join(ps.joinErr, receiptErr)
		ps.mu.Unlock()
	}
	// Finish inherited descendants before publishing completion, sharing the
	// checked attempt so explicit cancellation never causes a second signal.
	ps.finishGroup(true)
	ps.mu.Lock()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) && !errors.Is(err, context.Canceled) {
		ps.joinErr = errors.Join(ps.joinErr, err)
	}
	ps.mu.Unlock()
	if ps.lifetime != nil {
		select {
		case <-ps.readerDone:
		case <-ps.done:
			// Keep registered runtime output readable until its sole Wait
			// finishes. Closing it during cleanup could SIGPIPE the child
			// before its private receipt. After Wait, close to bound drain
			// even when a detached process retained an output descriptor.
			_ = ps.stdout.Close()
			<-ps.readerDone
		}
	} else {
		<-ps.readerDone
	}
	ps.cancel()
	if ps.lifetime != nil {
		ps.lifetime.Abort()
	}
	close(ps.completed)
}

func (ps *processSource) readLoop(chunkSize int) {
	defer close(ps.readerDone)
	defer close(ps.ch)
	defer ps.stdout.Close()
	buf := make([]byte, chunkSize)
	draining := false
	for {
		select {
		case <-ps.done:
			if ps.lifetime == nil {
				return
			}
			draining = true
		default:
		}
		n, err := io.ReadFull(ps.stdout, buf)
		if n > 0 && !draining {
			chunk := append([]byte(nil), buf[:n]...)
			evt := streamEvent{kind: "source_bytes", sourceName: ps.name, data: chunk}
			select {
			case ps.ch <- evt:
			case <-ps.done:
				if ps.lifetime == nil {
					return
				}
				draining = true
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				select {
				case <-ps.done:
				default:
					ps.mu.Lock()
					ps.joinErr = errors.Join(ps.joinErr, fmt.Errorf("worker stdout read: %w", err))
					ps.mu.Unlock()
				}
			}
			return
		}
	}
}

func (ps *processSource) receiptContext() (context.Context, context.CancelFunc) {
	ps.mu.Lock()
	deadline := ps.stopDeadline
	ps.mu.Unlock()
	if !deadline.IsZero() {
		return context.WithDeadline(context.Background(), deadline)
	}
	return context.WithTimeout(context.Background(), WorkerShutdownTimeout)
}
