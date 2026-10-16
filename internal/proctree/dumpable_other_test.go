//go:build !linux

package proctree

import (
	"errors"
	"testing"
)

// Off Linux, DisableDump must say it did nothing, never pretend.
func TestDisableDumpUnsupportedIsAnError(t *testing.T) {
	if err := DisableDump(); !errors.Is(err, ErrDumpProtectionUnsupported) {
		t.Fatalf("DisableDump() = %v, want ErrDumpProtectionUnsupported", err)
	}
}
