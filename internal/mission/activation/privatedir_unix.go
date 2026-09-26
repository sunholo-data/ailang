//go:build darwin || linux

package activation

import (
	"errors"
	"os"
)

// EnsurePrivateDir creates the activation state directory and proves it is private.
//
// The activation record names a frozen work item and a runtime binding; a group- or
// world-readable directory, or a symlink pointing somewhere else, would let a second party
// substitute either one between freeze and dispatch.
func EnsurePrivateDir(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return errors.New("activation state directory must be private and not a symlink")
	}
	return nil
}

// HostSupported reports whether local mission activation can run on this host.
//
// It exists so callers and tests skip on the PACKAGE'S OWN declaration rather than
// re-deriving a GOOS list that then drifts from lock_other.go — the "declared but not
// walked" shape this repo keeps paying for.
func HostSupported() bool { return true }
