//go:build unix

package fileguard

import (
	"io/fs"
	"syscall"
)

// linkCount is the file's hard-link count.
func linkCount(fi fs.FileInfo) uint64 {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Nlink)
	}
	return 1
}
