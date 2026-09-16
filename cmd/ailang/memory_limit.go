package main

import (
	"fmt"
	"runtime/debug"

	"github.com/sunholo-data/ailang/internal/config"
)

// applyMemoryLimit parses a human-readable memory size string and sets the
// Go runtime memory limit via debug.SetMemoryLimit. This triggers aggressive
// GC near the limit, providing a cleaner failure mode than OS OOM kill.
//
// Supported formats: "256MB", "1GB", "512mb", "1073741824" (bytes)
func applyMemoryLimit(s string) error {
	bytes, err := parseMemorySize(s)
	if err != nil {
		return fmt.Errorf("invalid --max-memory value '%s': %w", s, err)
	}
	if bytes <= 0 {
		return fmt.Errorf("--max-memory must be positive, got %d", bytes)
	}
	debug.SetMemoryLimit(bytes)
	return nil
}

// parseMemorySize converts a human-readable size string to bytes through the
// one size parser every cap shares (config.ParseByteSize): "256MB", "1GB",
// "8G", "1073741824".
func parseMemorySize(s string) (int64, error) {
	return config.ParseByteSize(s)
}
