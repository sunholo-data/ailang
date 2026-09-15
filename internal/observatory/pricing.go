// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

// M-V1-SIMPLIFY-S3 M2 (2026-09-15): the observatory prices tokens through
// internal/modelreg and nothing else.
//
// What this replaced (git log -S normalizeModelName -- internal/observatory):
// a second sync.Once loader that searched for models.yml by cwd and ~/go/src
// — ignoring AILANG_MODELS_PATH, the published registry and the embedded
// floor, so an installed binary printed "$0.00" whenever its cwd was wrong; a
// private ~20-entry alias map; and cache pricing hardcoded at 10% (read) /
// 125% (write) of input for EVERY provider, which is Anthropic's schedule and
// nobody else's. The generic OTLP path also never tried api_name, so every
// prod OpenRouter span ("z-ai/glm-5.3-flash", "moonshotai/kimi-k3") was stored
// at cost_usd=null. All of that is now Resolve + CostForName on the registry.

// ErrUnpriced is returned by PriceTokens when the registry has no row for the
// model. It wraps modelreg.ErrUnknownModel. A caller that stores a dollar
// figure MUST branch on it: "unpriced" is a different fact from "$0", and the
// chain rollup already has a display for it (CostStatusUnknown → unknown_stages
// + the incomplete-data warning in `ailang chains stats`).
var ErrUnpriced = modelreg.ErrUnknownModel

// registry returns the loaded model registry, initialising it on first use
// through the ONE precedence chain (explicit path → published → embedded).
func registry() (*modelreg.ModelsConfig, error) {
	if modelreg.GlobalModelsConfig == nil {
		if err := modelreg.InitModelsConfig(); err != nil {
			return nil, err
		}
	}
	return modelreg.GlobalModelsConfig, nil
}

// PriceTokens prices a call by any name the registry can resolve (friendly key,
// api_name, declared alias, dated/dotted variant). Cache reads and writes are
// billed at the model's OWN declared rates (full input rate when undeclared —
// the registry's overstate-visibly stance). Returns ErrUnpriced (wrapped) for
// an unknown model and a registry-load error if no registry can be found.
func PriceTokens(model string, tokensIn, tokensOut, cacheRead, cacheWrite int64) (float64, error) {
	cfg, err := registry()
	if err != nil {
		return 0, err
	}
	return cfg.CostForName(model, int(tokensIn), int(tokensOut), int(cacheRead), int(cacheWrite))
}

// ResolveCostFromTokens distinguishes an UNRESOLVABLE model from one that
// resolves to a $0 rate. Load-bearing for the cost-attribution classifier
// (M-MISSION-COST-CHAINS): a token-bearing stage whose model cannot be
// resolved surfaces as `unknown`, never as fabricated metered $0, while a
// model that legitimately resolves to a free rate rolls up to $0.
//
// Returns (cost, resolved): resolved=false → the caller MUST treat the row as
// unknown. An empty model returns (0, false): there is nothing to resolve.
func ResolveCostFromTokens(model string, tokensIn, tokensOut int64) (float64, bool) {
	return ResolveCostFromTokensWithCache(model, tokensIn, tokensOut, 0, 0)
}

// ResolveCostFromTokensWithCache is ResolveCostFromTokens for callers that know
// the cache split. Same contract.
func ResolveCostFromTokensWithCache(model string, tokensIn, tokensOut, cacheRead, cacheWrite int64) (float64, bool) {
	if model == "" {
		return 0, false
	}
	cost, err := PriceTokens(model, tokensIn, tokensOut, cacheRead, cacheWrite)
	if err != nil {
		return 0, false
	}
	return cost, true
}

// unpricedSeen rate-limits the stderr line below to once per model name per
// process, so a busy receiver does not log every span.
var unpricedSeen sync.Map

// CalculateCostFromTokens is the float-only entry point the OTLP receiver
// still calls for spans that carry tokens but no cost. It cannot return an
// error to its caller, so an unpriced model is made LOUD instead of silent: a
// stderr line naming the model (once per model per process) and $0 stored —
// the receiver is the place to stamp the span with an unpriced marker, which is
// a one-line follow-up outside this file's ownership (see the M2 report).
// Zero tokens price to $0 without a lookup.
func CalculateCostFromTokens(model string, tokensIn, tokensOut int64) float64 {
	return CalculateCostFromTokensWithCache(model, tokensIn, tokensOut, 0, 0)
}

// CalculateCostFromTokensWithCache is CalculateCostFromTokens with the cache
// split. Cache rates come from the model's registry row — NOT a fixed 10%/125%
// of input, which was only ever correct for Anthropic.
func CalculateCostFromTokensWithCache(model string, tokensIn, tokensOut, cacheRead, cacheWrite int64) float64 {
	if tokensIn == 0 && tokensOut == 0 && cacheRead == 0 && cacheWrite == 0 {
		return 0
	}
	cost, err := PriceTokens(model, tokensIn, tokensOut, cacheRead, cacheWrite)
	if err != nil {
		if _, dup := unpricedSeen.LoadOrStore(model, struct{}{}); !dup {
			// Diagnostic goes to stderr so it never corrupts --json stdout.
			if errors.Is(err, ErrUnpriced) {
				fmt.Fprintf(os.Stderr, "observatory: UNPRICED model %q — not in the model registry (%s); recording cost_usd=0. Add the row or an alias to models.yml.\n",
					model, modelreg.LoadedSource.Kind)
			} else {
				fmt.Fprintf(os.Stderr, "observatory: cannot price %q: %v\n", model, err)
			}
		}
		return 0
	}
	return cost
}

// CalculateCacheSavings is what the cache reads WOULD have cost as fresh input
// minus what they cost at the model's declared cache-read rate. A model with no
// declared read rate bills reads at the input rate, so its savings are $0 —
// honest, where the old 90% flat figure was a claim about every provider.
// Unresolvable or empty model → $0 (nothing to attribute).
func CalculateCacheSavings(model string, cacheRead int64) float64 {
	if cacheRead == 0 || model == "" {
		return 0
	}
	cfg, err := registry()
	if err != nil {
		return 0
	}
	_, m, err := cfg.Resolve(model)
	if err != nil {
		return 0
	}
	readRate := m.Pricing.CacheReadPer1K
	if readRate == 0 {
		return 0
	}
	saved := float64(cacheRead) / 1000.0 * (m.Pricing.InputPer1K - readRate)
	if saved < 0 {
		return 0
	}
	return saved
}

// GetPricingConfig returns the registry the observatory prices with — the one
// global in modelreg. Kept for tests that assert pricing is available.
func GetPricingConfig() *modelreg.ModelsConfig {
	cfg, err := registry()
	if err != nil {
		return nil
	}
	return cfg
}

// ResetPricingConfig forces the next lookup to re-run registry precedence.
// Test support only; production never resets.
func ResetPricingConfig() {
	modelreg.GlobalModelsConfig = nil
	unpricedSeen = sync.Map{}
}
