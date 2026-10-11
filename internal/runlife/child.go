package runlife

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Child owns private pipes; its caller remains the sole exec.Cmd.Wait owner.
// A successful Join requires a terminal receipt, never stdout or process death.
type Child struct {
	request   *os.File
	receipt   *os.File
	childEnds []*os.File
	received  chan struct{}
	waited    chan struct{}
	stopOnce  sync.Once
	waitOnce  sync.Once
	closeOnce sync.Once
	mu        sync.Mutex
	err       error
}

// Prepare registers only an executable with the executing runtime's file
// identity and an explicit run invocation. Symlink aliases retain identity;
// old/different runtime files retain the ordinary process-group boundary.
// A policy supervisor may pass a separate lifeline read end, never its writer.
func Prepare(cmd *exec.Cmd, lifeline *os.File) (*Child, error) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return nil, nil
	}
	if len(cmd.Args) < 2 || cmd.Args[1] != "run" {
		return nil, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	own, err := os.Stat(exe)
	if err != nil {
		return nil, err
	}
	target, err := os.Stat(cmd.Path)
	if err != nil {
		return nil, err
	}
	if !os.SameFile(own, target) {
		return nil, nil
	}
	requestR, requestW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	receiptR, receiptW, err := os.Pipe()
	if err != nil {
		requestR.Close()
		requestW.Close()
		return nil, err
	}
	c := &Child{request: requestW, receipt: receiptR, childEnds: []*os.File{requestR, receiptW}, received: make(chan struct{}), waited: make(chan struct{})}
	fd := 3 + len(cmd.ExtraFiles)
	cmd.ExtraFiles = append(cmd.ExtraFiles, requestR, receiptW)
	meta := strconv.Itoa(fd) + "," + strconv.Itoa(fd+1)
	if lifeline != nil {
		cmd.ExtraFiles = append(cmd.ExtraFiles, lifeline)
		c.childEnds = append(c.childEnds, lifeline)
		meta += "," + strconv.Itoa(fd+2)
	}
	env := cmd.Env
	if env == nil {
		env = os.Environ()
	}
	cmd.Env = append(StripEnvironment(env), envKey+"="+meta)
	return c, nil
}

func StripEnvironment(env []string) []string {
	clean := make([]string, 0, len(env))
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.EqualFold(key, envKey) {
			clean = append(clean, entry)
		}
	}
	return clean
}

func (c *Child) AfterStart() {
	for _, f := range c.childEnds {
		_ = f.Close()
	}
	c.childEnds = nil
	go func() {
		defer close(c.received)
		r := bufio.NewReaderSize(c.receipt, MaxFrame+1)
		ready, err := readFrame(r)
		if err == nil && ready.Kind != "ready" {
			err = fmt.Errorf("missing lifetime registration")
		}
		if err == nil {
			var terminal frame
			terminal, err = readFrame(r)
			if err == nil && terminal.Kind != "receipt" {
				err = fmt.Errorf("missing cleanup receipt")
			}
			if err == nil && terminal.Error != "" {
				err = fmt.Errorf("child cleanup failed: %s", terminal.Error)
			}
		}
		if err != nil {
			err = fmt.Errorf("runtime cleanup receipt: %w", err)
		}
		c.mu.Lock()
		c.err = errors.Join(c.err, err)
		c.mu.Unlock()
	}()
}

// RequestStop enqueues a remaining-duration request and returns without joining.
// The force callback is scoped to the caller's already-owned command group.
func (c *Child) RequestStop(ctx context.Context, force func() error) {
	c.stopOnce.Do(func() {
		remaining := ShutdownTimeout
		if deadline, ok := ctx.Deadline(); ok {
			remaining = time.Until(deadline)
			if remaining < 0 {
				remaining = 0
			}
			if remaining > ShutdownTimeout {
				remaining = ShutdownTimeout
			}
		}
		deadline := time.Now().Add(remaining)
		go func() {
			_ = c.request.SetWriteDeadline(deadline)
			err := writeFrame(c.request, frame{Kind: "stop", RemainingNS: int64(remaining)})
			if err != nil {
				c.mu.Lock()
				c.err = errors.Join(c.err, fmt.Errorf("runtime stop request: %w", err))
				c.mu.Unlock()
			}
			timer := time.NewTimer(time.Until(deadline))
			defer timer.Stop()
			select {
			case <-c.waited:
				return
			case <-timer.C:
			}
			c.mu.Lock()
			defer c.mu.Unlock()
			select {
			case <-c.waited:
				return
			default:
			}
			if force != nil {
				c.err = errors.Join(c.err, force())
			}
		}()
	})
}

func (c *Child) AfterWait() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.waitOnce.Do(func() { close(c.waited) })
}
func (c *Child) Join(ctx context.Context) error {
	select {
	case <-c.received:
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.err
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (c *Child) Abort() {
	c.closeOnce.Do(func() {
		for _, f := range c.childEnds {
			_ = f.Close()
		}
		_ = c.request.Close()
		_ = c.receipt.Close()
	})
}
