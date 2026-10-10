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
	"syscall"

	"github.com/sunholo-data/ailang/internal/proctree"
)

// managedProcess owns its child, process group, stdin pipe and writer task.
// The sole waitLoop reaps the command and joins the writer before publishing Done.
type managedProcess struct {
	id         int
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	writeCh    chan []byte
	done       chan struct{}
	writerDone chan struct{}
	stop       chan struct{}
	once       sync.Once
	cancel     context.CancelFunc
	kill       func() error
	mu         sync.Mutex
	stopMu     sync.Mutex
	closed     bool
	exited     bool
	exitErr    error
	writeErr   error
	stopErr    error
}

func NewManagedProcess(parentCtx context.Context, cmdPath string, args []string) (*managedProcess, error) {
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithCancel(parentCtx)
	cmd := exec.CommandContext(ctx, cmdPath, args...)
	proctree.Configure(cmd)
	// Context, explicit cancellation and natural completion share one checked
	// termination attempt. Never re-signal a group after its members were killed.
	var killOnce sync.Once
	var killErr error
	kill := func() error {
		killOnce.Do(func() {
			if WorkerCancellationSupported() {
				killErr = proctree.KillGroup(cmd.Process.Pid)
			} else {
				killErr = cmd.Process.Kill()
				if errors.Is(killErr, os.ErrProcessDone) {
					killErr = nil
				}
			}
		})
		return killErr
	}
	cmd.Cancel = kill
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		cancel()
		_ = stdin.Close()
		return nil, fmt.Errorf("start process: %w", err)
	}
	mp := &managedProcess{cmd: cmd, stdin: stdin, writeCh: make(chan []byte, 256), done: make(chan struct{}), writerDone: make(chan struct{}), stop: make(chan struct{}), cancel: cancel, kill: kill}
	go mp.writeLoop()
	go mp.waitLoop()
	return mp, nil
}

// Write accepts a copy without blocking. Channel sends and closure share mu.
func (mp *managedProcess) Write(data []byte) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	if mp.closed {
		return fmt.Errorf("stdin already closed")
	}
	if mp.exited {
		return fmt.Errorf("process exited")
	}
	select {
	case mp.writeCh <- append([]byte(nil), data...):
		return nil
	default:
		return fmt.Errorf("write buffer full — subprocess may be stalled")
	}
}

// CloseStdin requests cooperative EOF after queued bytes drain. It does not
// relinquish ownership or wait for a child that may ignore its input forever.
func (mp *managedProcess) CloseStdin() {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	if !mp.closed {
		mp.closed = true
		close(mp.writeCh)
	}
}

func (mp *managedProcess) StdinClosing() bool {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return mp.closed
}

// RequestStop force-stops this owned group. Repeated calls share one result.
func (mp *managedProcess) RequestStop() error {
	mp.once.Do(func() {
		mp.stopMu.Lock()
		defer mp.stopMu.Unlock()
		select {
		case <-mp.done:
			return
		default:
		}
		mp.CloseStdin()
		close(mp.stop)
		_ = mp.stdin.Close() // owned pipe; unblock any in-progress writer syscall
		mp.stopErr = mp.kill()
		mp.cancel()
	})
	mp.stopMu.Lock()
	defer mp.stopMu.Unlock()
	return mp.stopErr
}

// Join waits for the sole command waiter and owned writer, bounded by ctx.
func (mp *managedProcess) Join(ctx context.Context) error {
	select {
	case <-mp.done:
		mp.mu.Lock()
		err := mp.exitErr
		writeErr := mp.writeErr
		mp.mu.Unlock()
		var exited *exec.ExitError
		if err != nil && !errors.As(err, &exited) && !errors.Is(err, context.Canceled) {
			return err
		}
		mp.stopMu.Lock()
		defer mp.stopMu.Unlock()
		return errors.Join(mp.stopErr, writeErr)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (mp *managedProcess) Done() <-chan struct{} { return mp.done }

// Close is the compatibility host helper; public handlers use checked methods.
func (mp *managedProcess) Close() {
	_ = mp.RequestStop()
	ctx, cancel := context.WithTimeout(context.Background(), WorkerShutdownTimeout)
	defer cancel()
	_ = mp.Join(ctx)
}

func (mp *managedProcess) writeLoop() {
	defer close(mp.writerDone)
	defer func() { _ = mp.stdin.Close() }()
	for {
		select {
		case <-mp.stop:
			return
		default:
		}
		select {
		case <-mp.stop:
			return
		case data, ok := <-mp.writeCh:
			if !ok {
				return
			}
			if _, err := mp.stdin.Write(data); err != nil {
				mp.recordWriterError(err)
				mp.CloseStdin()
				return
			}
		}
	}
}

// Pipe closure during cancellation and peer EOF are expected. Other writer
// failures must reach cleanup receipts instead of disappearing with the task.
func (mp *managedProcess) recordWriterError(err error) {
	select {
	case <-mp.stop:
		return
	default:
	}
	if errors.Is(err, os.ErrClosed) || errors.Is(err, syscall.EPIPE) {
		return
	}
	mp.mu.Lock()
	mp.writeErr = errors.Join(mp.writeErr, err)
	mp.mu.Unlock()
}

func (mp *managedProcess) waitLoop() {
	err := mp.cmd.Wait() // exactly one owner, including natural exit
	mp.mu.Lock()
	mp.exited = true
	mp.exitErr = err
	mp.mu.Unlock()
	mp.CloseStdin()
	_ = mp.stdin.Close()
	<-mp.writerDone
	// A descendant can hold the group after its leader exits. Finish group cleanup
	// before Done permits registry release; never signal after Done is published.
	mp.stopMu.Lock()
	var groupErr error
	if WorkerCancellationSupported() {
		groupErr = mp.kill()
	}
	if mp.stopErr == nil {
		mp.stopErr = groupErr
	}
	close(mp.done)
	mp.stopMu.Unlock()
	mp.cancel()
}

// Compile-time ownership interface check also catches accidental missing joins.
var _ OwnedWorker = (*managedProcess)(nil)
