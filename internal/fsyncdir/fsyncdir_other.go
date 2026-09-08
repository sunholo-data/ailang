//go:build !darwin && !linux

package fsyncdir

// Sync is a no-op where directory fsync is not a supported operation.
//
// Windows has no fsync-on-directory: opening a directory and calling Sync returns
// "Access is denied". The durability guarantee the unix build provides — that a rename into
// this directory survives a crash — genuinely cannot be made here, so this returns nil rather
// than pretending. Callers that need that guarantee are already unix-only: see
// internal/mission/activation/lock_other.go, which refuses outright.
//
// Returning an error instead would fail every write on an unsupported host; returning nil
// keeps the code path usable and confines the missing guarantee to this documented seam.
func Sync(string) error { return nil }
