package motoko

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/executor"
)

// Cost of record for a motoko run = OUR registry's rates × the tokens motoko
// reported in run_summary.
//
// motoko also reports a dollar figure (run_summary.total_cost_usd), from its own
// meter (step_cost_millicents in motoko core), which prices input + output only
// and never sees cache tokens. That is wrong in BOTH directions, measured
// 2026-10-12:
//
//   - direct Anthropic (disjoint usage): cache reads/writes are separate counters
//     the meter ignores. Haiku 5.5 with prompt caching: motoko $0.013, actual $0.055.
//   - OpenAI-shaped routes (OpenRouter, OpenAI, ChatGPT, Gemini, ollama): cached
//     input is INSIDE input_tokens, so the meter bills every cache hit at the full
//     input rate. GLM-5.3-Flash via OpenRouter: motoko $2.00, OpenRouter billed $0.55.
//
// motoko's figure is kept only as diagnostic metadata
// (ProviderData["motoko_self_reported_cost_usd"]), never as the cost.

// motokoInputIncludesCacheReads reports whether motoko's input_tokens for this
// model already contain its cache_read_input_tokens.
//
// motoko copies std/ai's StepResult.input_tokens / cache_read_input_tokens
// straight through, and those are ai.Response.InputTokens /
// CacheReadInputTokens. Their relationship depends on the provider wire:
//
//   - anthropic: input_tokens is FRESH input; cache_read_input_tokens and
//     cache_creation_input_tokens are disjoint counters (internal/ai/anthropic).
//     Banked: input 54, cache_read 2,553,032, cache_creation 124,706.
//   - everything else is OpenAI- or Gemini-shaped: InputTokens is
//     prompt_tokens / promptTokenCount, and the cache read is a SUBSET reported
//     in prompt_tokens_details.cached_tokens / cachedContentTokenCount
//     (internal/ai/openai, openrouter, chatgpt, gemini). No cache-write counter
//     exists on those wires. Banked GLM-5.3-Flash/OpenRouter: input 13,272,269,
//     cache_read 12,998,912, total_tokens = input + output.
//
// The provider is motoko's own routing prefix ("anthropic/" means the DIRECT
// Anthropic API to motoko, unlike ai.GuessProvider, which reads it as an
// OpenRouter vendor id); a bare name falls back to ai.GuessProvider.
func motokoInputIncludesCacheReads(model string) bool {
	provider, _, ok := strings.Cut(model, "/")
	if !ok {
		provider = string(ai.GuessProvider(model))
	}
	return provider != string(ai.ProviderAnthropic)
}

// applyRegistryCost rewrites a parsed motoko Result onto the executor contract:
// InputTokens becomes FRESH input (disjoint from cache reads, as codex already
// does), and CostUSD/CostProvenance come from the registry via
// executor.PriceTokens. Task.Pricing (set by the eval harness and mission
// dispatch) wins; otherwise the model string is resolved through the registry.
//
// CostKilledAt is left as the parser set it: it is the value motoko's own cap
// compared against, which is what actually stopped the run.
func applyRegistryCost(res *executor.Result, task *executor.Task, model string) {
	if res.ProviderData == nil {
		res.ProviderData = map[string]any{}
	}
	if res.CostUSD != 0 {
		res.ProviderData["motoko_self_reported_cost_usd"] = res.CostUSD
	}
	if motokoInputIncludesCacheReads(model) && res.CacheReadInputTokens > 0 {
		if res.CacheReadInputTokens > res.InputTokens {
			// The subset relationship does not hold, so the split is unknown. Do not
			// clamp and price a guess: say so.
			res.ProviderData["motoko_cost_unpriced_reason"] = fmt.Sprintf(
				"cache_read_input_tokens %d exceeds input_tokens %d on a cache-inclusive route (%s)",
				res.CacheReadInputTokens, res.InputTokens, model)
			res.CostUSD, res.CostProvenance = 0, executor.CostProvenanceUnknown
			return
		}
		res.ProviderData["motoko_reported_input_tokens"] = res.InputTokens
		res.InputTokens -= res.CacheReadInputTokens
	}
	var fallback *executor.CostModel
	if task == nil || task.Pricing == nil {
		fallback = executor.HarnessCostModel("motoko", model)
	}
	res.CostUSD, res.CostProvenance = executor.PriceTokens(task, fallback, motokoAuthLane(model), executor.TokenUsage{
		InputTokens:              res.InputTokens,
		OutputTokens:             res.OutputTokens + res.ReasonTokens,
		CacheReadInputTokens:     res.CacheReadInputTokens,
		CacheCreationInputTokens: res.CacheCreationInputTokens,
	})
}
