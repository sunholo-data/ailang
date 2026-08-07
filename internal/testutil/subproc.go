package testutil

import (
	"bytes"
	"errors"
	"os/exec"
	"testing"
	"time"
)

// RunBounded runs a subprocess within both cap and the enclosing test's
// deadline. It captures the output streams separately and returns the child's
// exit code, including -1 when the process was killed before it could exit.
func RunBounded(t *testing.T, dir string, cap time.Duration, bin string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	ctx, cancel := HangGuardContext(t, cap)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	cmd.WaitDelay = 5 * time.Second

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	err := cmd.Run()
	stdout = stdoutBuffer.String()
	stderr = stderrBuffer.String()
	if err == nil {
		return stdout, stderr, 0
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return stdout, stderr, exitError.ExitCode()
	}
	t.Fatalf("testutil: run %q: %v", bin, err)
	return "", "", -1
}
