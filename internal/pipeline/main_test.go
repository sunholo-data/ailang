package pipeline

import (
	"os"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
)

// TestMain isolates the package from a developer's AILANG_NO_CACHE. The
// pipeline reads it directly since #1275, and #1275's own workaround told
// people to export it, which disabled the cache under every cache test here.
func TestMain(m *testing.M) {
	_ = os.Unsetenv(config.EnvNoCache)
	os.Exit(m.Run())
}
