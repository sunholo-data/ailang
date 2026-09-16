package main

import (
	"github.com/sunholo-data/ailang/internal/config"
)

// parseMemorySize converts a human-readable size string to bytes through the
// one size parser every cap shares (config.ParseByteSize): "256MB", "1GB",
// "8G", "1073741824".
func parseMemorySize(s string) (int64, error) {
	return config.ParseByteSize(s)
}
