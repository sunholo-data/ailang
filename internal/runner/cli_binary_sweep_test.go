package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// A timed-out package run skips TestMain's cleanup; the next first build must
// remove the dirs of processes that are gone and leave live ones alone.
func TestSweepStaleCLIBuildsRemovesOnlyDeadOwners(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the sweep is a no-op on Windows (only POSIX tests build the CLI)")
	}
	exited := exec.Command("true")
	if err := exited.Run(); err != nil {
		t.Fatal(err)
	}
	deadPID := exited.Process.Pid
	if processAlive(deadPID) {
		t.Skipf("pid %d was reused before the sweep could be tested", deadPID)
	}
	tmp := t.TempDir()
	mk := func(name string) string {
		dir := filepath.Join(tmp, name)
		if err := os.MkdirAll(filepath.Join(dir, "ailang"), 0o755); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	dead := mk(fmt.Sprintf("%s%d-123", cliBuildPrefix, deadPID))
	self := mk(fmt.Sprintf("%s%d-456", cliBuildPrefix, os.Getpid()))
	parent := mk(fmt.Sprintf("%s%d-789", cliBuildPrefix, os.Getppid()))
	freshNoPID := mk(cliBuildPrefix + "abc")
	other := mk("unrelated-dir")

	sweepStaleCLIBuilds(tmp)

	if _, err := os.Stat(dead); !os.IsNotExist(err) {
		t.Errorf("the build of exited pid %d survived the sweep (err %v)", deadPID, err)
	}
	for _, keep := range []string{self, parent, freshNoPID, other} {
		if _, err := os.Stat(keep); err != nil {
			t.Errorf("%s was removed: %v", filepath.Base(keep), err)
		}
	}
}
