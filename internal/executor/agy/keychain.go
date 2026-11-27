package agy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// A locked keychain for every agy HOME (macOS only).
//
// agy saves a refreshed sign-in token to the macOS login keychain. Our route HOMEs
// (~/.ailang/state/agy/<route>) have none, so on every refresh the Security framework
// raised a "Keychain Not Found" SecurityAgent dialog. agy 1.5.15 waits 5s, gives up
// and writes its token file instead, but the dialog STAYS OPEN after agy exits, and
// the rig's quota reads (several per hour across the mission loops) piled them up on
// the studio's screen (measured 2026-11-27, reproduced with an expired token).
//
// A keychain of the HOME's own, set as that HOME's default and left locked, makes the
// save fail at once (`security` exit 36, "User interaction is not allowed") with no
// dialog, and agy keeps using its token file, which is the route's source of truth.
// Nobody knows the keychain's password: it is random, used once to create the file,
// and discarded, so it never unlocks and never holds a credential. The operator's own
// keychain search list is not touched, because every `security` call runs with
// HOME set to the route HOME.

// keychainPath is the route HOME's own login keychain.
func keychainPath(home string) string {
	return filepath.Join(home, "Library", "Keychains", "login.keychain-db")
}

// securityRun runs /usr/bin/security under home. A variable so tests can count calls.
var securityRun = func(home string, args ...string) ([]byte, error) {
	cmd := exec.Command("/usr/bin/security", args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + home}
	cmd.Stdin = nil
	return cmd.CombinedOutput()
}

// EnsureLockedKeychain gives home a locked default keychain of its own, so agy's
// token saves under it fail without raising a dialog. Idempotent and cheap; a no-op
// off macOS. An error means only that dialogs may appear: callers report it and run.
func EnsureLockedKeychain(home string) error {
	if runtime.GOOS != "darwin" || home == "" {
		return nil
	}
	kc := keychainPath(home)
	if _, err := os.Stat(kc); err != nil {
		if err := os.MkdirAll(filepath.Dir(kc), 0o700); err != nil {
			return fmt.Errorf("agy keychain: %w", err)
		}
		buf := make([]byte, 24)
		if _, err := rand.Read(buf); err != nil {
			return fmt.Errorf("agy keychain: %w", err)
		}
		out, err := securityRun(home, "create-keychain", "-p", hex.EncodeToString(buf), kc)
		// A concurrent launch may have created it first; only a missing file is fatal.
		if _, statErr := os.Stat(kc); err != nil && statErr != nil {
			return fmt.Errorf("agy keychain: create %s: %v: %s", kc, err, strings.TrimSpace(string(out)))
		}
	}
	if out, _ := securityRun(home, "default-keychain", "-d", "user"); !strings.Contains(string(out), kc) {
		if out, err := securityRun(home, "list-keychains", "-d", "user", "-s", kc); err != nil {
			return fmt.Errorf("agy keychain: search list: %v: %s", err, strings.TrimSpace(string(out)))
		}
		if out, err := securityRun(home, "default-keychain", "-d", "user", "-s", kc); err != nil {
			return fmt.Errorf("agy keychain: default: %v: %s", err, strings.TrimSpace(string(out)))
		}
	}
	if out, err := securityRun(home, "lock-keychain", kc); err != nil {
		return fmt.Errorf("agy keychain: lock: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
