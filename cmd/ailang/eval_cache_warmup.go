package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// Prompt-cache warm-up (M-ANTHROPIC-CACHE-HIT-RATE, design decision D4).
//
// Why this exists: a cache entry only becomes readable once the FIRST response
// has begun streaming. The suite fans out `--parallel 10` by default, so without
// a warm-up all ten concurrent calls miss and each pays a full ~1.25x cache
// WRITE for the same prefix. On a 30-benchmark suite that is the difference
// between a ~52% and a ~86% saving on the shared teaching prompt:
//
//	no warm-up : 10 writes x 1.25 + 20 reads x 0.10 = 14.50  (vs 30.00 uncached)
//	warm-up    :  1 write  x 1.25 + 29 reads x 0.10 =  4.15
//
// One extra serial round-trip buys 34 percentage points.
//
// Design constraints this respects:
//   - NEVER fatal. A warm-up is an optimisation; if it fails for any reason the
//     suite must run exactly as it would have anyway. Every error path here
//     warns and returns.
//   - Only where it pays. Anthropic-only (the other providers cache
//     automatically with no notion of a warm-up), and only when a (model,
//     language) group has at least 2 jobs — with one job there is no second
//     call to read the entry back.
//   - Cheap. MaxTokens is 1: the point is to make the provider PREFILL and cache
//     the prompt prefix, not to get an answer. The response is discarded.

// warmupMaxTokens keeps the warm-up call's output to a single token. The prefill
// (and therefore the cache write) happens regardless of the output budget, so
// there is no reason to pay for more.
//
// Not 0: our Generate path treats MaxTokens <= 0 as "unset" and substitutes
// 4096, which would make the warm-up more expensive than the calls it is meant
// to accelerate.
const warmupMaxTokens = 1

// warmupTask is the volatile suffix for the warm-up request. It is deliberately
// NOT a real benchmark task — the cached prefix is the teaching prompt, and what
// follows it is never part of the cached span.
const warmupTask = "\n\n## Task\n\nReply with the single word: ok."

// cacheWarmupKey identifies a group of jobs that share a cacheable prefix.
// Language matters because the teaching prompt is per-language.
type cacheWarmupKey struct {
	model    string
	language string
}

// warmPromptCaches makes one cheap call per (model, language) group so the
// shared teaching prompt is cached before the parallel fan-out begins.
//
// Returns the number of groups successfully warmed (for logging/tests).
func warmPromptCaches(ctx context.Context, jobs []Job, timeout time.Duration) int {
	groups := groupJobsForWarmup(jobs)
	if len(groups) == 0 {
		return 0
	}

	warmed := 0
	for key, sample := range groups {
		if err := warmOneCache(ctx, key, sample, timeout); err != nil {
			// Non-fatal by design — see the file comment.
			fmt.Fprintf(os.Stderr, "⚠ prompt-cache warm-up skipped for %s/%s: %v\n",
				key.model, key.language, err)
			continue
		}
		warmed++
	}
	if warmed > 0 {
		fmt.Printf("🔥 Warmed prompt cache for %d model/language group(s)\n", warmed)
	}
	return warmed
}

// groupJobsForWarmup returns one representative job per (model, language) group
// that is worth warming: Anthropic-backed and running at least 2 jobs.
func groupJobsForWarmup(jobs []Job) map[cacheWarmupKey]Job {
	counts := make(map[cacheWarmupKey]int)
	sample := make(map[cacheWarmupKey]Job)
	for _, j := range jobs {
		k := cacheWarmupKey{model: j.Model, language: j.Language}
		counts[k]++
		if _, seen := sample[k]; !seen {
			sample[k] = j
		}
	}

	out := make(map[cacheWarmupKey]Job)
	for k, n := range counts {
		if n < 2 {
			// A single call can never read back what it wrote; warming would be
			// a pure ~1.25x cost.
			continue
		}
		if !isAnthropicModel(k.model) {
			// Other providers cache automatically on the server side with no
			// client-visible write step, so there is nothing to pre-warm.
			continue
		}
		out[k] = sample[k]
	}
	return out
}

// isAnthropicModel reports whether a models.yml model id resolves to the
// Anthropic provider. Resolution failures are treated as "not Anthropic" so an
// unknown model silently skips the warm-up rather than aborting the suite.
func isAnthropicModel(model string) bool {
	_, provider, err := eval_harness.ResolveModelName(model)
	if err != nil {
		return false
	}
	return ai.ProviderFromString(provider) == ai.ProviderAnthropic
}

// warmOneCache issues the single prefill call for one group.
func warmOneCache(ctx context.Context, key cacheWarmupKey, sample Job, timeout time.Duration) error {
	specPath := filepath.Join(evalBenchmarkDir, sample.Benchmark+".yml")
	spec, err := eval_harness.LoadSpec(specPath)
	if err != nil {
		return fmt.Errorf("load spec: %w", err)
	}

	// The base is identical across every benchmark in a language (asserted by
	// TestPromptParts_BaseIsStableAcrossBenchmarks), so any job in the group
	// gives the prefix the whole group will share.
	base, _ := spec.PromptPartsForLanguage(key.language)
	if base == "" {
		return fmt.Errorf("no cacheable base prompt for language %q", key.language)
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return warmupCallFn(callCtx, key.model, base, warmupTask, warmupMaxTokens)
}

// warmupCallFn performs the actual prefill request. It is a package-level var so
// tests can exercise the ORCHESTRATION around it — grouping, spec loading,
// prefix derivation, error propagation, the warmed counter — without a provider,
// an API key, or a network. The network call itself is covered by the live,
// key-gated probe in internal/ai/anthropic (TestLiveGeneratePromptCache); there
// is nothing this indirection hides that a unit test could have checked anyway.
//
// NewAIAgent is the STANDARD-mode (direct HTTP) path — it always requires
// ANTHROPIC_API_KEY, via getAPIKeyForProvider in internal/eval_harness/ai_provider.go.
// isAnthropicModel (above) does not distinguish "runs in standard mode" from
// "runs in agent mode via the claude CLI executor" — so on every --agent run with
// an Anthropic model, this warm-up call ALWAYS attempts the key-based path and
// therefore ALWAYS fails/warns when ANTHROPIC_API_KEY is unset. That is the
// CORRECT state for agent mode: agent-mode Claude calls go through a completely
// separate executor (internal/executor) that shells out to the `claude` CLI and
// authenticates via Keychain OAuth/subscription — an inherited ANTHROPIC_API_KEY
// silently wins over OAuth there and bills the metered API instead (see
// reference_headless_claude_billing_rig in project memory; this cost a real
// billing incident once). So: "prompt-cache warm-up skipped ... ANTHROPIC_API_KEY
// not set" on an agent run is expected noise, not a sign anything is broken — the
// actual per-benchmark claude executor calls right after it use OAuth and don't
// need the key at all. Do NOT "fix" this by exporting the key into the
// environment; that fixes nothing here (warm-up is best-effort and non-fatal
// either way, see the design-constraints comment atop this file) and actively
// risks metered billing on every subsequent claude CLI call in that shell.
var warmupCallFn = func(ctx context.Context, model, cachedPrefix, task string, maxTokens int) error {
	agent, err := eval_harness.NewAIAgent(model, 0)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}
	if _, err := agent.GenerateCodeWarmup(ctx, cachedPrefix, task, maxTokens); err != nil {
		return fmt.Errorf("prefill call: %w", err)
	}
	return nil
}
