//go:build darwin || linux

package pkg

import (
	"fmt"
	"os"
	"syscall"
)

func openPublishKeyFile(path string) (*os.File, error) {
	if err := Require(AuthNetwork, "reading publisher credentials", OperatorRemedy); err != nil {
		return nil, err
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	info, err := f.Stat()
	if err == nil {
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			err = fmt.Errorf("cannot verify file owner")
		} else {
			err = validatePublishKeyInfo(info, stat.Uid, uint32(os.Geteuid()))
		}
	}
	if err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func validatePublishKeyInfo(info os.FileInfo, owner, current uint32) error {
	if !info.Mode().IsRegular() {
		return fmt.Errorf("must be a regular non-symlink file")
	}
	mode := info.Mode()
	if (mode.Perm() != 0400 && mode.Perm() != 0600) || mode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return fmt.Errorf("must have private mode 0400 or 0600 (no special bits)")
	}
	if owner != current {
		return fmt.Errorf("must be owned by the effective user")
	}
	return nil
}
