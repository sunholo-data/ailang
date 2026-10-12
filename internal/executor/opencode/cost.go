package opencode

import (
	"github.com/sunholo-data/ailang/internal/executor"
)

// Cost of record for an opencode run = OUR registry's rates × the tokens
// opencode reported in its step_finish events.
//
// opencode also reports a dollar figure per step (step_finish.part.cost), from
// its own price table (models.dev). That table is not ours: it can lag a price
// change, not know a model, or price a route differently from the one we pin,
// and it is the same failure pi had (2026-10-12: pi billed Haiku 5.5 ~56x over
// list). opencode's figure is kept only as diagnostic metadata
// (ProviderData["opencode_self_reported_cost_usd"]), never as the cost.
//
// No input/cache normalisation is needed here: opencode reports cache tokens
// EXCLUSIVE of input (see the step_finish accumulation in opencode.go and its
// testdata assertion), which is already the fresh-input contract PriceTokens
// requires.

// opencodeRunCost prices an opencode run's accumulated tokens. Task.Pricing (set
// by the eval harness and mission dispatch) wins; otherwise the model string is
// resolved through the registry. outputTokens and reasonTokens are the Result's
// disjoint pair — both bill at the output rate.
func opencodeRunCost(task *executor.Task, model string, inputTokens, outputTokens, reasonTokens, cacheRead, cacheWrite int) (float64, executor.CostProvenance) {
	var fallback *executor.CostModel
	if task == nil || task.Pricing == nil {
		fallback = executor.HarnessCostModel("opencode", model)
	}
	return executor.PriceTokens(task, fallback, executor.AuthLaneForModel(model), executor.TokenUsage{
		InputTokens:              inputTokens,
		OutputTokens:             outputTokens + reasonTokens,
		CacheReadInputTokens:     cacheRead,
		CacheCreationInputTokens: cacheWrite,
	})
}

// withSelfReportedCost records opencode's own dollar figure as diagnostic
// metadata. Zero is omitted (local models, or a wire that carried none).
func withSelfReportedCost(pd map[string]any, selfReported float64) map[string]any {
	if selfReported == 0 {
		return pd
	}
	if pd == nil {
		pd = map[string]any{}
	}
	pd["opencode_self_reported_cost_usd"] = selfReported
	return pd
}
