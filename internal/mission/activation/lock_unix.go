//go:build darwin || linux

package activation

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func (m *Manager) lock() (func(), error) {
	if !filepath.IsAbs(m.Dir) {
		return nil, fmt.Errorf("activation directory must be absolute")
	}
	if err := os.MkdirAll(m.Dir, 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(m.Dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("activation directory must be a private non-symlink directory")
	}
	// O_NOFOLLOW keeps a foreign lock symlink from redirecting file access.
	fd, err := syscall.Open(filepath.Join(m.Dir, "lock"), syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("activation operation busy: %w", err)
	}
	return func() { _ = syscall.Flock(fd, syscall.LOCK_UN); _ = syscall.Close(fd) }, nil
}
