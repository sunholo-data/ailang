package types

import (
	"os"
	"testing"
)

// Every environment free-variable query in this package's tests is recomputed
// uncached and must match the memoised answer (M-TYPECHECK-ENV-SCAN-COST).
func TestMain(m *testing.M) {
	SetEnvVarCrossCheck(true)
	os.Exit(m.Run())
}
