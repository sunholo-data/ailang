package pi

import "github.com/sunholo-data/ailang/internal/executor"

// M-MODEL-REGISTRY-SINGLE-SOURCE M6 (D2(a)).
//
// executor.DefaultConfig() no longer seeds a model — that seeding was the
// UPSTREAM half of the ten hardcoded provider defaults, and an unpinned agent
// silently inherited one. Tests that only need a working executor (rather than
// testing model resolution itself) pin one explicitly here.
//
// Pinning in a test is the CORRECT behavior under D2(a), not a workaround: the
// whole point is that a model is always chosen deliberately by somebody. This
// keeps that choice visible in one place per package.
func testConfig() *executor.Config {
	cfg := executor.DefaultConfig()
	cfg.PiModel = "anthropic/claude-haiku-4-5"
	return cfg
}
