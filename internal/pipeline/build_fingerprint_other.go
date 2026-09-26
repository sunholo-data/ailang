//go:build !unix

package pipeline

import "os"

// fileIdentity has no inode to add off unix; size and mtime stand alone.
func fileIdentity(os.FileInfo) string { return "" }
