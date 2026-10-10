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
