package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// loadPreviousLock permits first-time generation, but refuses to overwrite an
// existing lock that cannot be read or validated.
func loadPreviousLock(dir string) (*pkg.LockFile, error) {
	previous, err := pkg.LoadLockFile(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot update existing %s; repair or remove it before running 'ailang lock': %w", pkg.LockFileName, err)
	}
	return previous, nil
}

// preserveLockGeneration retains the original provenance only when resolution
// is unchanged. Resolution must still run to populate caches and detect drift.
func preserveLockGeneration(previous, expected *pkg.LockFile) {
	if previous == nil || len(pkg.LockDrift(previous, expected)) != 0 {
		return
	}
	expected.GeneratedAt = previous.GeneratedAt
	expected.Generator = previous.Generator
	expected.AILANGVersion = previous.AILANGVersion
}
