//go:build !linux

package proctree

// DisableDump is unsupported off Linux and says so (see ErrDumpProtectionUnsupported).
func DisableDump() error { return ErrDumpProtectionUnsupported }
