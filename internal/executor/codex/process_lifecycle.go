package codex

import (
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// nativeProcessLifecycle shares one Wait between the event reader and cleanup.
// Cleanup always kills the owned group, including children of an exited leader,
// then joins the refresh owner before a cloud credential lease can be released.
func nativeProcessLifecycle(cmd *exec.Cmd) (wait func() error, stop func(), finish func() error) {
	var waitOnce sync.Once
	waitDone := make(chan struct{})
	var waitErr error
	wait = func() error {
		waitOnce.Do(func() { go func() { waitErr = cmd.Wait(); close(waitDone) }() })
		select {
		case <-waitDone:
			return waitErr
		case <-time.After(5 * time.Second):
			return ErrProcessTerminationUnconfirmed
		}
	}
	var terminationMu sync.Mutex
	var terminationErr error
	stop = func() {
		if err := terminateProcessTree(cmd); err != nil {
			terminationMu.Lock()
			terminationErr = err
			terminationMu.Unlock()
		}
	}
	finish = func() error {
		stop()
		terminationMu.Lock()
		killErr := terminationErr
		terminationMu.Unlock()
		if err := wait(); errors.Is(err, ErrProcessTerminationUnconfirmed) || killErr != nil {
			return fmt.Errorf("%w: wait=%v terminate=%v", ErrProcessTerminationUnconfirmed, err, killErr)
		}
		return nil
	}
	return wait, stop, finish
}
