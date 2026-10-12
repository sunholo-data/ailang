//go:build unix

package proctree

import (
	"errors"
	"syscall"
	"testing"
)

func TestKillGroupChecksDisappearanceAfterPermissionError(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		killErr, probeErr, wantErr error
		wantCalls                  int
	}{
		{"killed", nil, nil, nil, 1},
		{"already absent", syscall.ESRCH, nil, nil, 1},
		{"disappeared during kill", syscall.EPERM, syscall.ESRCH, nil, 2},
		{"permission denied", syscall.EPERM, syscall.EPERM, syscall.EPERM, 2},
		{"still present", syscall.EPERM, nil, syscall.EPERM, 2},
		{"probe failed", syscall.EPERM, syscall.EIO, syscall.EPERM, 2},
		{"other kill error", syscall.EINVAL, nil, syscall.EINVAL, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			err := killProcessGroupWith(1234, func(pid int, sig syscall.Signal) error {
				calls++
				if pid != -1234 {
					t.Fatalf("signalled wrong group: %d", pid)
				}
				if calls == 1 {
					if sig != syscall.SIGKILL {
						t.Fatalf("first signal: %v", sig)
					}
					return tc.killErr
				}
				if calls != 2 || sig != 0 {
					t.Fatalf("unexpected signal retry: %d %v", calls, sig)
				}
				return tc.probeErr
			})
			if !errors.Is(err, tc.wantErr) || calls != tc.wantCalls {
				t.Fatalf("error=%v calls=%d; want error=%v calls=%d", err, calls, tc.wantErr, tc.wantCalls)
			}
		})
	}
}

func TestResolveGroupErrorPreservesUnconfirmedAndJoinedErrors(t *testing.T) {
	ioErr := errors.New("owned I/O failed")
	joined := errors.Join(syscall.EPERM, ioErr)
	for _, tc := range []struct {
		name             string
		err, probe, want error
		calls            int
	}{
		{"gone after reap", syscall.EPERM, syscall.ESRCH, nil, 1},
		{"still live", syscall.EPERM, nil, syscall.EPERM, 1},
		{"denied probe", syscall.EPERM, syscall.EPERM, syscall.EPERM, 1},
		{"failed probe", syscall.EPERM, syscall.EIO, syscall.EPERM, 1},
		{"unrelated error", syscall.EIO, syscall.ESRCH, syscall.EIO, 0},
		{"joined error", joined, syscall.ESRCH, joined, 0},
		{"success", nil, syscall.ESRCH, nil, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			got := resolveGroupErrorWith(9876, tc.err, func(pid int, sig syscall.Signal) error {
				calls++
				if pid != -9876 || sig != 0 {
					t.Fatalf("unexpected group/signal: %d %v", pid, sig)
				}
				return tc.probe
			})
			if got != tc.want || calls != tc.calls {
				t.Fatalf("error=%v calls=%d; want %v %d", got, calls, tc.want, tc.calls)
			}
		})
	}
}
