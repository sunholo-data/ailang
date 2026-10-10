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

	"github.com/sunholo-data/ailang/internal/proctree"
)

// processSource owns its child process group, stdout pipe and reader. Selection
// borrows the source; only cancellation or the execution owner stops it.
type processSource struct {
	name       string
	priority   int
	ch         chan streamEvent
	done       chan struct{} // delivery/read cancellation, not completion
	completed  chan struct{}
	readerDone chan struct{}
	once       sync.Once
	groupOnce  sync.Once
	cmd        *exec.Cmd
	stdout     *os.File
	cancel     context.CancelFunc
	mu         sync.Mutex
	stopErr    error
	joinErr    error
}

// NewProcessSource starts one owned process group and one Wait goroutine.
// A private pipe keeps Wait from closing stdout before its final bytes drain.
func NewProcessSource(parentCtx context.Context, cmdPath string, args []string, name string, priority int, chunkSize int) (EventSource, error) {
	if chunkSize <= 0 {
		return nil, fmt.Errorf("chunkSize must be positive, got %d", chunkSize)
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
		cmd: cmd, stdout: stdout, cancel: cancel,
	}
	cmd.Stdout = childStdout
	cmd.Cancel = ps.RequestStop // checked termination; never discard a kill error
	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		_ = childStdout.Close()
		cancel()
		return nil, fmt.Errorf("start process: %w", err)
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
	ps.once.Do(func() {
		close(ps.done)
		ps.finishGroup(false)
		_ = ps.stdout.Close()
		ps.cancel()
	})
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.stopErr
}

// finishGroup performs one checked termination attempt shared by context
// cancellation, explicit cancellation and natural leader completion.
func (ps *processSource) finishGroup(natural bool) {
	ps.groupOnce.Do(func() {
		var err error
		if WorkerCancellationSupported() {
			err = proctree.KillGroup(ps.cmd.Process.Pid)
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
		ps.mu.Lock()
		defer ps.mu.Unlock()
		return errors.Join(ps.stopErr, ps.joinErr)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close follows EventSource's signal-only contract. The execution supervisor
// and public cancellation operation perform the checked, deadline-bound join.
func (ps *processSource) Close() { _ = ps.RequestStop() }

func (ps *processSource) waitLoop() {
	err := ps.cmd.Wait() // exactly one owner, including natural EOF
	// Finish inherited descendants before publishing completion, sharing the
	// checked attempt so explicit cancellation never causes a second signal.
	ps.finishGroup(true)
	ps.mu.Lock()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) && !errors.Is(err, context.Canceled) {
		ps.joinErr = errors.Join(ps.joinErr, err)
	}
	ps.mu.Unlock()
	<-ps.readerDone
	ps.cancel()
	close(ps.completed)
}

func (ps *processSource) readLoop(chunkSize int) {
	defer close(ps.readerDone)
	defer close(ps.ch)
	defer ps.stdout.Close()
	buf := make([]byte, chunkSize)
	for {
		select {
		case <-ps.done:
			return
		default:
		}
		n, err := io.ReadFull(ps.stdout, buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			evt := streamEvent{kind: "source_bytes", sourceName: ps.name, data: chunk}
			select {
			case ps.ch <- evt:
			case <-ps.done:
				return
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
