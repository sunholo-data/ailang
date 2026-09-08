//go:build !darwin && !linux

package activation

import (
	"errors"
	"os"
)

// EnsurePrivateDir creates the activation state directory.
//
// The unix build proves the directory is private via its permission bits. Windows does not
// carry them — os.MkdirAll(dir, 0700) yields 0777 — so the check there rejected every
// directory it created and failed two tests before activation ever reached the one place
// that is SUPPOSED to decline: lock_other.go, which states that local mission activation
// requires macOS or Linux host locking.
//
// Two different errors for one unsupported platform is worse than one. This keeps the
// symlink and is-a-directory guarantees, which ARE portable, and leaves the refusal to the
// lock — so an operator on Windows gets the accurate reason rather than a permissions
// complaint about a guarantee the filesystem cannot express.
func EnsurePrivateDir(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("activation state directory must be a directory, not a symlink or file")
	}
	return nil
}

// HostSupported reports whether local mission activation can run on this host.
//
// False here: lock_other.go refuses, because activation needs flock-style host locking to
// guarantee one owner of a work item, and this build has no equivalent.
func HostSupported() bool { return false }
