package pi

import (
	"strings"

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

// piCostModel resolves the registry rate card for a pi model string.
//
// pi names models "<provider>/<model-id>". The full string is tried first (pi
// rows declare it as agent_model_name). If nothing matches, the bare model id is
// tried, but accepted ONLY when the row it resolves to belongs to that same
// provider — so "anthropic/claude-haiku-5-5" prices as the direct Anthropic row,
// while "ollama/<x>" can never borrow a cloud row's price. Anything else is an
// explicit Unpriced card: $0 with provenance "unknown", not a guess.
func piCostModel(model string) *executor.CostModel {
	if cm, err := executor.CostModelFor(model); err == nil {
		return cm
	}
	if provider, id, ok := strings.Cut(model, "/"); ok && id != "" {
		if cm, err := executor.CostModelFor(id); err == nil && cm.ProviderName == provider {
			return cm
		}
	}
	return executor.UnpricedCostModel("pi", model)
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
