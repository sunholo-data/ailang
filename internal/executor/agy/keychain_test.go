package agy

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func realKeychainState(t *testing.T) string {
	t.Helper()
	a, _ := exec.Command("/usr/bin/security", "list-keychains", "-d", "user").CombinedOutput()
	b, _ := exec.Command("/usr/bin/security", "default-keychain", "-d", "user").CombinedOutput()
	return string(a) + string(b)
}

// showKeychainInfoBounded runs `security show-keychain-info` the way securityRun does,
// but killed after 20s; a timeout returns context.DeadlineExceeded.
func showKeychainInfoBounded(home, kc string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/security", "show-keychain-info", kc)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + home}
	cmd.WaitDelay = 2 * time.Second
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return out, context.DeadlineExceeded
	}
	return out, err
}

// Against the real `security` tool: the route HOME gets its own locked default keychain,
// a second call reuses it, and the operator's own keychain setup is untouched.
func TestEnsureLockedKeychain_Darwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS only")
	}
	home := t.TempDir()
	before := realKeychainState(t)

	if err := EnsureLockedKeychain(home); err != nil {
		t.Fatal(err)
	}
	kc := keychainPath(home)
	st, err := os.Stat(kc)
	if err != nil {
		t.Fatalf("keychain not created: %v", err)
	}
	if out, _ := securityRun(home, "default-keychain", "-d", "user"); !strings.Contains(string(out), kc) {
		t.Fatalf("default keychain under the route HOME = %q, want %s", out, kc)
	}
	// Locked: reading its settings is refused without a dialog (exit 36). Outside a
	// login session (a CI runner) `security` instead waits for the keychain password
	// on the terminal, which hung macOS CI for the full 15m test timeout, so this probe
	// is bounded and that case skips: there is no dialog to prevent there anyway.
	if out, err := showKeychainInfoBounded(home, kc); errors.Is(err, context.DeadlineExceeded) {
		t.Skipf("show-keychain-info waited for a password (no login session); locked check needs one: %s", out)
	} else if err == nil {
		t.Fatalf("keychain is unlocked (show-keychain-info succeeded: %s); agy would write to it", out)
	} else if ee := (*exec.ExitError)(nil); !errors.As(err, &ee) || ee.ExitCode() != 36 {
		t.Fatalf("show-keychain-info: want exit 36 (locked), got %v: %s", err, out)
	}

	if err := EnsureLockedKeychain(home); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if st2, _ := os.Stat(kc); !st2.ModTime().Equal(st.ModTime()) || st2.Size() != st.Size() {
		t.Fatal("second call recreated the keychain")
	}
	if after := realKeychainState(t); after != before {
		t.Fatalf("operator keychain setup changed:\nbefore %s\nafter  %s", before, after)
	}
}

func TestEnsureLockedKeychain_NoHomeIsNoop(t *testing.T) {
	calls := 0
	old := securityRun
	t.Cleanup(func() { securityRun = old })
	securityRun = func(string, ...string) ([]byte, error) { calls++; return nil, nil }
	if err := EnsureLockedKeychain(""); err != nil || calls != 0 {
		t.Fatalf("empty home: err=%v calls=%d", err, calls)
	}
}
