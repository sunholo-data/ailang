package dashboard_transforms

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo/ailang/internal/embed"
)

// findProjectRoot walks up from the current directory looking for the std/ directory
// which indicates the project root. Returns the path or empty string if not found.
func findProjectRoot() string {
	// Start from current directory
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Walk up looking for std/ directory
	for dir != "/" && dir != "." {
		stdPath := filepath.Join(dir, "std")
		if info, err := os.Stat(stdPath); err == nil && info.IsDir() {
			return dir
		}
		dir = filepath.Dir(dir)
	}

	return ""
}

// newTestEngine creates an embed.Engine with the correct project root.
// Skips the test if the project root (with std/) cannot be found.
func newTestEngine(t *testing.T) *embed.Engine {
	t.Helper()

	root := findProjectRoot()
	if root == "" {
		t.Skip("Cannot find project root with std/ directory - skipping AILANG integration test")
	}

	return embed.New(root)
}

// newBenchEngine creates an embed.Engine with the correct project root for benchmarks.
// Skips the benchmark if the project root (with std/) cannot be found.
func newBenchEngine(b *testing.B) *embed.Engine {
	b.Helper()

	root := findProjectRoot()
	if root == "" {
		b.Skip("Cannot find project root with std/ directory - skipping AILANG benchmark")
	}

	return embed.New(root)
}
