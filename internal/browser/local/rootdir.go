package local

import (
	"fmt"
	"os"
	"runtime"
)

// browserRootMode is the only permission a browser root may carry: owner-only.
const browserRootMode os.FileMode = 0o700

// secureBrowserRoot creates the browser session root and then VERIFIES it.
//
// os.MkdirAll alone is not sufficient here. The default root is a predictable
// path under a world-writable parent (os.TempDir()), and MkdirAll returns nil
// when the path already exists — it does not take ownership and does not
// re-apply the mode. A local attacker who pre-creates that name, or points a
// symlink at it, therefore keeps control of a directory we then write session
// state into.
//
// That mattered less when sessions were anonymous. It matters now: an
// authenticated run's Chromium profile lives under this root, and it holds live
// session cookies for the leased account.
//
// So the post-condition is checked rather than assumed, and a root that fails it
// is a loud provisioning failure rather than a silent downgrade.
func secureBrowserRoot(dir string) error {
	if dir == "" {
		return fmt.Errorf("browser root path is empty")
	}
	if err := os.MkdirAll(dir, browserRootMode); err != nil {
		return err
	}

	// Lstat, not Stat: Stat follows a symlink and would happily describe the
	// attacker's target instead of the link itself.
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("browser root %s is a symlink; refusing to write session state through it", dir)
	}
	if !info.IsDir() {
		return fmt.Errorf("browser root %s exists and is not a directory", dir)
	}

	// Windows does not model POSIX permission bits — Go reports 0777 for
	// directories there regardless of ACL — so tightening and checking a mode
	// there would be theatre. Access control on that platform is the user
	// profile directory's ACL, not a mode.
	if runtime.GOOS == "windows" {
		return nil
	}

	// A pre-existing directory keeps its old mode, because MkdirAll does not
	// re-apply one. The provider owns this tree — it creates and deletes session
	// directories inside it — so the right response to a loose mode is to
	// TIGHTEN it, not to refuse an otherwise usable root. Chmod is also the
	// ownership probe: it fails for a directory some other user created, which
	// is precisely the case that must not be silently accepted.
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		if err := os.Chmod(dir, browserRootMode); err != nil {
			return fmt.Errorf(
				"browser root %s is group- or world-accessible (mode %04o) and could not be tightened: %w; "+
					"an authenticated session's cookies would be readable by other local users",
				dir, perm, err)
		}
	}

	// Re-read rather than assume the chmod took effect.
	final, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if perm := final.Mode().Perm(); perm&0o077 != 0 {
		return fmt.Errorf("browser root %s is still mode %04o after tightening, want %04o", dir, perm, browserRootMode)
	}
	return nil
}
