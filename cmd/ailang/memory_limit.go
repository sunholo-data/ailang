package main

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
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

// parseMemorySize converts a human-readable size string to bytes.
// Accepts: "256MB", "1GB", "512mb", "1073741824"
func parseMemorySize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	upper := strings.ToUpper(s)

	// Try suffixed forms
	for _, suffix := range []struct {
		label      string
		multiplier int64
	}{
		{"GB", 1 << 30},
		{"MB", 1 << 20},
		{"KB", 1 << 10},
	} {
		if strings.HasSuffix(upper, suffix.label) {
			numStr := strings.TrimSpace(s[:len(s)-len(suffix.label)])
			n, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number '%s'", numStr)
			}
			return int64(n * float64(suffix.multiplier)), nil
		}
	}

	// Plain bytes
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size '%s' (use 256MB, 1GB, or bytes)", s)
	}
	return n, nil
}
