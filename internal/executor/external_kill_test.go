package executor

import (
	"errors"
	"os/exec"
	"runtime"
	"testing"
)

// TestIsExternalTermination runs real children: a SIGTERM death (both the
// trapped `exit 143` form Claude Code uses and the untrapped signal form) is
// external; a SIGKILL — the only signal the harness sends — and an ordinary
// non-zero exit are not.
func TestIsExternalTermination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX signals")
	}
	tests := []struct {
		name   string
		script string
		want   bool
	}{
		{"untrapped SIGTERM", "kill -TERM $$", true},
		{"trapped SIGTERM, exit 143 (claude)", "exit 143", true},
		{"harness SIGKILL", "kill -KILL $$", false},
		{"ordinary failure", "exit 1", false},
		{"clean exit", "exit 0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := exec.Command("/bin/sh", "-c", tt.script).Run()
			if got := IsExternalTermination(err); got != tt.want {
				t.Errorf("IsExternalTermination(%v) = %v, want %v", err, got, tt.want)
			}
		})
	}
	if IsExternalTermination(errors.New("signal: terminated")) {
		t.Error("a bare error string is not a process exit; only *exec.ExitError counts")
	}
}
