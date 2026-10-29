//go:build !unix

package fileguard

import "io/fs"

// linkCount is not available from fs.FileInfo here (Windows reports it only
// through a handle query); hard-linked aliases are not detected on these
// platforms.
func linkCount(fs.FileInfo) uint64 { return 1 }
