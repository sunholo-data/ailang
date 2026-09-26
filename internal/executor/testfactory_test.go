package executor_test

import (
	"sync"

	"github.com/sunholo-data/ailang/internal/executor"
)

// M-MODEL-REGISTRY-SINGLE-SOURCE M6 (D2(a)).
//
// executor.DefaultConfig() no longer seeds a model for any harness — that
// seeding was the UPSTREAM half of the ten hardcoded provider defaults. So the
// global factory can still say which executors are REGISTERED, but can no
// longer BUILD one until a model is supplied. That split is the intended
// contract: a factory that can always build an executor is one that chose a
// model nobody asked for.
//
// In production M7 closes the gap — the coordinator resolves the agent's role
// through the registry and passes the model down. Tests that need a buildable
// factory apply models here, which is the same act done explicitly.
//
// It must be the GLOBAL factory, not a fresh NewFactory: the harnesses register
// themselves onto the global one in their package init(), so a new factory is
// empty.
var testFactoryOnce sync.Once

func testFactory() *executor.ExecutorFactory {
	f := executor.GlobalFactory()
	testFactoryOnce.Do(func() {
		f.UpdateConfig(func(c *executor.Config) {
			c.ClaudeModel = "haiku"
			c.CodexModel = "gpt-5-codex"
			c.OpenCodeModel = "anthropic/claude-haiku-4-5"
			c.PiModel = "anthropic/claude-haiku-4-5"
			c.MotokoModel = "openrouter/anthropic/claude-haiku-4-5"
		})
	})
	return f
}
