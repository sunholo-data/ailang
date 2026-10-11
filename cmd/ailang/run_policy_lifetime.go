package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/proctree"
	"github.com/sunholo-data/ailang/internal/runlife"
)

// policyLifetime serializes worker admission against inherited cancellation.
// The supervisor owns the lifeline writer; no worker or descendant receives it.
type policyLifetime struct {
	cmd       *exec.Cmd
	child     *runlife.Child
	lifeline  *os.File
	done      chan struct{}
	mu        sync.Mutex
	stopped   bool
	deadline  time.Time
	err       error
	groupOnce sync.Once
	groupErr  error
}

func preparePolicyLifetime(cmd *exec.Cmd) (*policyLifetime, error) {
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	child, err := runlife.Prepare(cmd, reader)
	if err != nil || child == nil {
		reader.Close()
		writer.Close()
		return nil, errors.Join(err, fmt.Errorf("policy runtime could not register lifetime"))
	}
	l := &policyLifetime{cmd: cmd, child: child, lifeline: writer, done: make(chan struct{})}
	cmd.Cancel = func() error { _ = l.killGroup(); return nil } // policy limits stop execution immediately
	if err := runlife.Bind(l.stop); err != nil {
		l.close()
		return nil, err
	}
	return l, nil
}

func (l *policyLifetime) start() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.stopped {
		return fmt.Errorf("policy owner stopped before worker admission")
	}
	if err := l.cmd.Start(); err != nil {
		return err
	}
	l.child.AfterStart()
	return nil
}

func (l *policyLifetime) stop(ctx context.Context) error {
	l.mu.Lock()
	l.stopped = true
	if deadline, ok := ctx.Deadline(); ok {
		l.deadline = deadline
	}
	started := l.cmd.Process != nil
	l.mu.Unlock()
	if !started {
		return nil
	}
	l.child.RequestStop(ctx, l.killGroup)
	select {
	case <-l.done:
		l.mu.Lock()
		defer l.mu.Unlock()
		return l.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *policyLifetime) remaining() time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.deadline.IsZero() {
		return proctree.DefaultWaitDelay
	}
	remaining := time.Until(l.deadline)
	if remaining < 0 {
		return 0
	}
	return remaining
}
func (l *policyLifetime) finish() {
	l.child.AfterWait()
	ctx, cancel := context.WithTimeout(context.Background(), l.remaining())
	defer cancel()
	err := errors.Join(l.child.Join(ctx), proctree.ResolveGroupError(l.cmd.Process.Pid, l.killGroup()))
	l.mu.Lock()
	l.err = err
	l.mu.Unlock()
	close(l.done)
	runlife.Complete(err)
}
func (l *policyLifetime) close() { l.child.Abort(); _ = l.lifeline.Close() }

// Hard policy limits and inherited cancellation share one checked force stop.
// Natural completion also clears members which inherited the worker's group.
func (l *policyLifetime) killGroup() error {
	l.groupOnce.Do(func() { l.groupErr = proctree.KillGroup(l.cmd.Process.Pid) })
	return l.groupErr
}
func (l *policyLifetime) afterWait() { l.child.AfterWait(); _ = l.killGroup() }
