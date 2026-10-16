package proctree

import "errors"

// ErrDumpProtectionUnsupported is returned by DisableDump on a platform with
// no PR_SET_DUMPABLE. It is an error, never a silent no-op: a caller that
// relies on dump protection must be able to see that it does not have it.
var ErrDumpProtectionUnsupported = errors.New("parent dump protection (PR_SET_DUMPABLE) is not supported on this platform")
