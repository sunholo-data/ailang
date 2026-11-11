package pipeline

import (
	"github.com/sunholo-data/ailang/internal/types"
	"os"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
)

// TestMain isolates the package from a developer's AILANG_NO_CACHE. The
// pipeline reads it directly since #1275, and #1275's own workaround told
// people to export it, which disabled the cache under every cache test here.
func TestMain(m *testing.M) {
	_ = os.Unsetenv(config.EnvNoCache)
	// Every environment free-variable query is recomputed uncached and must
	// match the memoised answer (M-TYPECHECK-ENV-SCAN-COST).
	types.SetEnvVarCrossCheck(true)
	// A nominal type that reaches unification without its defining module is a
	// minting site M-V1-NOMINAL-TYPE-IDENTITY missed: fail loudly (§4.1).
	types.SetUnqualifiedNominalPanic(true)
	os.Exit(m.Run())
}
