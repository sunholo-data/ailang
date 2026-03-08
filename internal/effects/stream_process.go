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

// processSource implements EventSource for streaming subprocess stdout.
//
// M-ASYNC-IO Phase 2: Spawns a subprocess and delivers its stdout as SourceBytes
// events into selectEventsLoop. The subprocess is killed when the source is closed.
type processSource struct {
	name     string
	priority int
	ch       chan streamEvent
	done     chan struct{}
	once     sync.Once
	cmd      *exec.Cmd
	cancel   context.CancelFunc
}

// NewProcessSource creates an EventSource that reads subprocess stdout in fixed-size chunks.
//
// The subprocess is started immediately. A background goroutine reads stdout in
// chunkSize-byte increments and sends SourceBytes events to the channel.
//
// Parameters:
//   - parentCtx: parent context (cancelled when selectEvents loop exits)
//   - cmdPath: resolved absolute path to the binary
//   - args: command arguments (no shell expansion)
//   - name: source name for SourceBytes(name, data) matching
//   - priority: dispatch priority in selectEvents (higher = checked first)
//   - chunkSize: bytes per SourceBytes event (determines streaming latency)
//
// The subprocess is killed (SIGTERM → 5s grace → SIGKILL) when Close() is called
// or when the parent context is cancelled.
func NewProcessSource(parentCtx context.Context, cmdPath string, args []string, name string, priority int, chunkSize int) (EventSource, error) {
	if chunkSize <= 0 {
		return nil, fmt.Errorf("chunkSize must be positive, got %d", chunkSize)
	}

	ctx, cancel := context.WithCancel(parentCtx)

	cmd := exec.CommandContext(ctx, cmdPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	// Discard stderr — streaming mode doesn't capture it
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start process: %w", err)
	}

	src := &processSource{
		name:     name,
		priority: priority,
		ch:       make(chan streamEvent, 100),
		done:     make(chan struct{}),
		cmd:      cmd,
		cancel:   cancel,
	}

	go src.readLoop(stdout, chunkSize)
	return src, nil
}

func (ps *processSource) Name() string               { return ps.name }
func (ps *processSource) Priority() int              { return ps.priority }
func (ps *processSource) Events() <-chan streamEvent { return ps.ch }

// Close stops the subprocess and cleans up resources.
// Safe to call multiple times.
func (ps *processSource) Close() {
	ps.once.Do(func() {
		close(ps.done)
		ps.cancel() // sends signal to subprocess via context

		// Give subprocess time to exit gracefully, then force kill
		go func() {
			timer := time.NewTimer(5 * time.Second)
			defer timer.Stop()

			waitDone := make(chan struct{})
			go func() {
				_ = ps.cmd.Wait()
				close(waitDone)
			}()

			select {
			case <-waitDone:
				// Clean exit
			case <-timer.C:
				// Force kill after grace period
				if ps.cmd.Process != nil {
					_ = ps.cmd.Process.Signal(syscall.SIGKILL)
				}
			}
		}()
	})
}

func (ps *processSource) readLoop(stdout io.ReadCloser, chunkSize int) {
	defer close(ps.ch)
	defer stdout.Close()

	buf := make([]byte, chunkSize)

	for {
		select {
		case <-ps.done:
			return
		default:
		}

		n, err := io.ReadFull(stdout, buf)

		// Deliver any bytes read (even partial on EOF/error)
		if n > 0 {
			// Copy the bytes so the buffer can be reused
			chunk := make([]byte, n)
			copy(chunk, buf[:n])

			evt := streamEvent{
				kind:       "source_bytes",
				sourceName: ps.name,
				data:       chunk,
			}

			select {
			case ps.ch <- evt:
			case <-ps.done:
				return
			}
		}

		if err != nil {
			// EOF or read error — subprocess finished or pipe closed
			return
		}
	}
}
