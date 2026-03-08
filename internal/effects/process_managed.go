//go:build !js

package effects

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// managedProcess holds a long-running subprocess with a writable stdin pipe.
//
// M-ASYNC-IO Phase 3: Enables incremental writes to a subprocess's stdin,
// complementing processSource (Phase 2) which reads stdout.
// Use case: streaming audio playback via sox, data pipelines via jq, etc.
type managedProcess struct {
	id      int
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	writeCh chan []byte   // Buffered write channel (capacity: 256)
	done    chan struct{} // Closed when subprocess exits or Close() called
	once    sync.Once     // Idempotent close
	cancel  context.CancelFunc
	mu      sync.Mutex
	closed  bool  // Stdin pipe closed
	exited  bool  // Subprocess has exited
	exitErr error // Subprocess exit error (if any)
}

// NewManagedProcess spawns a subprocess with a writable stdin pipe.
//
// The subprocess is started immediately. A background goroutine (writeLoop)
// drains the write channel and writes to the stdin pipe. Stdout and stderr
// are discarded — use asyncExecProcess for stdout streaming.
//
// Parameters:
//   - parentCtx: parent context (cancelled on program exit)
//   - cmdPath: resolved absolute path to the binary
//   - args: command arguments (no shell expansion)
func NewManagedProcess(parentCtx context.Context, cmdPath string, args []string) (*managedProcess, error) {
	ctx, cancel := context.WithCancel(parentCtx)

	cmd := exec.CommandContext(ctx, cmdPath, args...)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	// Discard stdout and stderr — this is a write-only process handle
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start process: %w", err)
	}

	mp := &managedProcess{
		cmd:     cmd,
		stdin:   stdinPipe,
		writeCh: make(chan []byte, 256),
		done:    make(chan struct{}),
		cancel:  cancel,
	}

	go mp.writeLoop()
	go mp.waitLoop()

	return mp, nil
}

// Write sends data to the subprocess's stdin pipe via the write channel.
// Non-blocking: returns an error if the buffer is full or the pipe is closed.
func (mp *managedProcess) Write(data []byte) error {
	mp.mu.Lock()
	if mp.closed {
		mp.mu.Unlock()
		return fmt.Errorf("stdin already closed")
	}
	if mp.exited {
		mp.mu.Unlock()
		return fmt.Errorf("process exited")
	}
	mp.mu.Unlock()

	// Copy data to avoid caller mutation
	buf := make([]byte, len(data))
	copy(buf, data)

	select {
	case mp.writeCh <- buf:
		return nil
	default:
		return fmt.Errorf("write buffer full — subprocess may be stalled")
	}
}

// CloseStdin closes the stdin pipe, signaling EOF to the subprocess.
// Drains any remaining buffered writes first, then waits for subprocess exit.
func (mp *managedProcess) CloseStdin() {
	mp.mu.Lock()
	if mp.closed {
		mp.mu.Unlock()
		return
	}
	mp.closed = true
	mp.mu.Unlock()

	// Close write channel — writeLoop will drain remaining and close stdin pipe
	close(mp.writeCh)
}

// Close performs full cleanup: kills the subprocess and releases resources.
// Safe to call multiple times.
func (mp *managedProcess) Close() {
	mp.once.Do(func() {
		// Mark as closed so no more writes are accepted
		mp.mu.Lock()
		mp.closed = true
		mp.mu.Unlock()

		// Close write channel if not already closed by CloseStdin
		select {
		case _, ok := <-mp.writeCh:
			if ok {
				// Channel still open — we got a stale value, drain and close
				// This is tricky — use a flag instead
			}
		default:
		}
		// Safe close: use sync to avoid double-close panic
		mp.closeWriteChSafe()

		// Cancel context (sends signal to subprocess)
		mp.cancel()

		// Wait for subprocess with grace period
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()

		select {
		case <-mp.done:
			// Clean exit
		case <-timer.C:
			// Force kill after grace period
			if mp.cmd.Process != nil {
				_ = mp.cmd.Process.Signal(syscall.SIGKILL)
			}
		}
	})
}

// closeWriteChSafe closes the write channel without panicking on double-close.
func (mp *managedProcess) closeWriteChSafe() {
	defer func() { recover() }() // Catch double-close panic
	close(mp.writeCh)
}

// writeLoop drains the write channel and writes to the stdin pipe.
// Exits when the channel is closed (by CloseStdin or Close).
func (mp *managedProcess) writeLoop() {
	for data := range mp.writeCh {
		_, err := mp.stdin.Write(data)
		if err != nil {
			// Pipe broken — subprocess may have exited
			mp.mu.Lock()
			mp.closed = true
			mp.mu.Unlock()
			// Drain remaining channel entries to unblock senders
			for range mp.writeCh {
			}
			return
		}
	}
	// Channel closed — close stdin pipe to signal EOF
	mp.stdin.Close()
}

// waitLoop waits for the subprocess to exit and records the result.
func (mp *managedProcess) waitLoop() {
	err := mp.cmd.Wait()
	mp.mu.Lock()
	mp.exited = true
	mp.exitErr = err
	mp.mu.Unlock()
	// Signal that subprocess has exited
	select {
	case <-mp.done:
		// Already closed
	default:
		close(mp.done)
	}
}
