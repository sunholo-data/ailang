package testutil

import (
	"os"
	"testing"
)

// FastLoopEnv is the opt-OUT switch for the language inner loop. `make
// test-core` sets it; nothing else does — CI and `make test` never see it, so
// every test guarded by SkipInFastLoop runs there unconditionally.
//
// This is the explicit counterpart of `testing.Short`, which gatelint R1
// refuses: `-short` is inert in CI (never passed), so a Short guard cannot be
// told apart from a test that simply never runs. An env var named for the
// one caller that sets it can.
const FastLoopEnv = "AILANG_TEST_FAST_LOOP"

// SkipInFastLoop skips a test that is correct but slow (a corpus walk, a
// multi-second timing fixture) when the developer asked for the fast inner
// loop. reason names what the test walks and how long it takes, so the skip
// line in -v output is self-explaining.
func SkipInFastLoop(t *testing.T, reason string) {
	t.Helper()
	if os.Getenv(FastLoopEnv) == "1" {
		t.Skipf("fast loop (%s=1): %s", FastLoopEnv, reason)
	}
}
