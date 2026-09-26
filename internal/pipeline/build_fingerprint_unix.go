//go:build unix

package pipeline

import (
	"fmt"
	"os"
	"syscall"
)

// fileIdentity adds the inode to the dirty-build fingerprint: a rebuild or a
// `cp -p` writes a new file, so two same-size binaries stamped within one
// coarse mtime tick (HFS+, FAT, some network mounts) still differ.
func fileIdentity(fi os.FileInfo) string {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return fmt.Sprintf("-i%d", st.Ino)
	}
	return ""
}
