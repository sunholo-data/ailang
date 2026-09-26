//go:build darwin || linux

// Package fsyncdir fsyncs a directory so a rename into it is durable.
//
// Three call sites did this by hand — activation/files.go, dispatch/receipt.go and
// iteration/review_packet.go — each opening the directory and calling Sync. On Windows every
// one returns "Access is denied": there is no fsync-on-directory there, and a directory
// handle opened this way cannot be synced at all. Measured on the Windows CI runner
// 2026-09-08, it failed three mission tests.
//
// One helper rather than three copies, because the audit's own finding is that a rule
// implemented at four call sites is a rule three of them will forget.
package fsyncdir

import "os"

// Sync flushes the directory entry so a completed rename survives a crash.
func Sync(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
