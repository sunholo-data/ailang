package motoko

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

func TestRunRegisteredCommandRegistersBeforeWait(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(config.EnvRigLockDir, dir)
	t.Setenv(config.EnvRigLockHeld, "1")
	cmd := exec.Command("sh", "-c", "sleep 0.4")
	done := make(chan error, 1)
	go func() { done <- runRegisteredCommand(cmd) }()

	children := filepath.Join(dir, "children")
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(children); err == nil && strings.Contains(string(b), " sh") {
			if err := <-done; err != nil {
				t.Fatalf("wait: %v", err)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("child was not registered while it was still running")
}

func TestRunRegisteredCommandStartFailureDoesNotRegister(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(config.EnvRigLockDir, dir)
	t.Setenv(config.EnvRigLockHeld, "1")
	err := runRegisteredCommand(exec.Command(filepath.Join(dir, "missing")))
	if err == nil {
		t.Fatal("expected start failure")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "children")); !os.IsNotExist(statErr) {
		t.Fatalf("start failure registered a child: %v", statErr)
	}
}
