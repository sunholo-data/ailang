package pi

import (
	"github.com/sunholo-data/ailang/internal/executor"
)

// Cost of record for a pi run = OUR registry's rates × the tokens pi reported.
//
// pi also reports a dollar figure per turn (message_end.message.usage.cost.total),
// computed from pi's OWN built-in price table. That table is wrong for models it
// does not know or prices differently. Measured 2026-10-12: six real tasks on
// anthropic/claude-haiku-5-5 — pi reported $15.49, the tokens at Haiku 5.5 list
// price came to $0.275 (~56x over). Banked as Result.CostUSD, that figure tripped
// the mission stage cost cap on healthy runs. It is kept only as diagnostic
// metadata (ProviderData["pi_self_reported_cost_usd"]), never as the cost.

// piCostModel resolves the registry rate card for a pi "<provider>/<model-id>"
// string. See executor.HarnessCostModel for the resolution rules.
func piCostModel(model string) *executor.CostModel {
	return executor.HarnessCostModel("pi", model)
}

// piRunCost prices a pi run's accumulated tokens. Task.Pricing (set by the eval
// harness and mission dispatch) wins; otherwise the model is resolved through
// the registry. outputTokens and reasonTokens are the Result's disjoint pair —
// both bill at the output rate.
func piRunCost(task *executor.Task, model string, inputTokens, outputTokens, reasonTokens, cacheRead, cacheWrite int) (float64, executor.CostProvenance) {
	var fallback *executor.CostModel
	if task == nil || task.Pricing == nil {
		fallback = piCostModel(model)
	}
	return executor.PriceTokens(task, fallback, executor.AuthLaneForModel(model), executor.TokenUsage{
		InputTokens:              inputTokens,
		OutputTokens:             outputTokens + reasonTokens,
		CacheReadInputTokens:     cacheRead,
		CacheCreationInputTokens: cacheWrite,
	})
}

// withSelfReportedCost records pi's own dollar figure as diagnostic metadata.
// Zero is omitted (local models, or a wire that carried none).
func withSelfReportedCost(pd map[string]any, selfReported float64) map[string]any {
	if selfReported == 0 {
		return pd
	}
	if pd == nil {
		pd = map[string]any{}
	}
	pd["pi_self_reported_cost_usd"] = selfReported
	return pd
}
