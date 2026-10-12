package modelreg

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// ClaudeCreditModel identifies the explicitly selected pilot allowlist model.
// It is not a fallback for unresolved executor configuration.
const ClaudeCreditModel = "claude-haiku-5-5"

// ClaudeCreditPricing is a reviewed single-inference Messages contract. Bounds
// include cached input and hidden thinking output. Server tools, compaction,
// fallback/advisor iterations, batch, priority, residency premiums and unknown
// beta features must be rejected by the gateway before using this contract.
// Rates are micro-USD per million tokens, never floating-point dollars.
type ClaudeCreditPricing struct {
	Revision          string            `yaml:"revision"`
	SourceURL         string            `yaml:"source_url"`
	VerifiedOn        string            `yaml:"verified_on"`
	BoundsVerified    bool              `yaml:"bounds_verified"`
	MaxInputTokens    int64             `yaml:"max_input_tokens"`
	MaxOutputTokens   int64             `yaml:"max_output_tokens"`
	ShortContextLimit int64             `yaml:"short_context_limit"`
	ShortContext      ClaudeCreditRates `yaml:"short_context"`
	LongContext       ClaudeCreditRates `yaml:"long_context"`
}

// ClaudeCreditRates prices mutually exclusive provider usage categories.
type ClaudeCreditRates struct {
	Input        int64 `yaml:"input"`
	Output       int64 `yaml:"output"`
	CacheRead    int64 `yaml:"cache_read"`
	CacheWrite5m int64 `yaml:"cache_write_5m"`
	CacheWrite1h int64 `yaml:"cache_write_1h"`
}

// ClaudeUsage is one complete provider usage record. InputTokens excludes all
// cache categories; OutputTokens includes thinking and text. The gateway must
// validate the API's cache_creation_input_tokens equals the two TTL categories
// before constructing this value, and reject unsupported billable categories.
type ClaudeUsage struct {
	InputTokens        int64
	OutputTokens       int64
	CacheReadTokens    int64
	CacheWrite5mTokens int64
	CacheWrite1hTokens int64
}

// ReserveClaudeRequest returns a sufficient upper bound in micro-USD, plus the
// immutable rate-card digest. It reserves the entire model input window at the
// highest declared input/cache rate and all requested output, including hidden
// thinking. Prompt estimates and count_tokens responses cannot reduce it.
func ReserveClaudeRequest(model string, maxOutputTokens int) (int64, string, error) {
	return GlobalModelsConfig.ReserveClaudeRequest(model, maxOutputTokens)
}

// ReserveClaudeRequest is the registry-scoped equivalent for explicit registries.
func (c *ModelsConfig) ReserveClaudeRequest(model string, maxOutputTokens int) (int64, string, error) {
	p, revision, err := c.claudeCreditCard(model)
	if err != nil {
		return 0, "", err
	}
	if maxOutputTokens <= 0 || int64(maxOutputTokens) > p.MaxOutputTokens {
		return 0, "", fmt.Errorf("Claude credit output limit must be between 1 and %d", p.MaxOutputTokens)
	}
	inputRate, outputRate := int64(0), int64(0)
	for _, r := range []ClaudeCreditRates{p.ShortContext, p.LongContext} {
		inputRate = max(inputRate, r.Input, r.CacheRead, r.CacheWrite5m, r.CacheWrite1h)
		outputRate = max(outputRate, r.Output)
	}
	amount, err := claudeRoundedCost([]int64{p.MaxInputTokens, int64(maxOutputTokens)}, []int64{inputRate, outputRate})
	return amount, revision, err
}

// ClaudeUsageCost settles a supported complete usage record in micro-USD.
func ClaudeUsageCost(model string, usage ClaudeUsage) (int64, error) {
	return GlobalModelsConfig.ClaudeUsageCost(model, usage)
}

// ClaudeUsageCostAtRevision refuses settlement if rates changed after reservation.
// The caller retains the original reservation until that exposure is reconciled.
func ClaudeUsageCostAtRevision(model string, usage ClaudeUsage, revision string) (int64, error) {
	return GlobalModelsConfig.ClaudeUsageCostAtRevision(model, usage, revision)
}

// ClaudeUsageCost is the registry-scoped settlement helper.
func (c *ModelsConfig) ClaudeUsageCost(model string, usage ClaudeUsage) (int64, error) {
	return c.ClaudeUsageCostAtRevision(model, usage, "")
}

// ClaudeUsageCostAtRevision uses explicit pricing provenance when provided.
func (c *ModelsConfig) ClaudeUsageCostAtRevision(model string, usage ClaudeUsage, revision string) (int64, error) {
	p, currentRevision, err := c.claudeCreditCard(model)
	if err != nil {
		return 0, err
	}
	if revision != "" && revision != currentRevision {
		return 0, fmt.Errorf("Claude credit pricing revision changed; retain reservation")
	}
	counts := []int64{usage.InputTokens, usage.CacheReadTokens, usage.CacheWrite5mTokens, usage.CacheWrite1hTokens}
	totalInput := int64(0)
	for _, n := range counts {
		if n < 0 || n > p.MaxInputTokens-totalInput {
			return 0, fmt.Errorf("invalid Claude credit input usage")
		}
		totalInput += n
	}
	if usage.OutputTokens < 0 || usage.OutputTokens > p.MaxOutputTokens {
		return 0, fmt.Errorf("invalid Claude credit output usage")
	}
	r := p.ShortContext
	if totalInput > p.ShortContextLimit {
		r = p.LongContext
	}
	return claudeRoundedCost(append(counts, usage.OutputTokens), []int64{r.Input, r.CacheRead, r.CacheWrite5m, r.CacheWrite1h, r.Output})
}

func (c *ModelsConfig) claudeCreditCard(model string) (*ClaudeCreditPricing, string, error) {
	if c == nil {
		return nil, "", fmt.Errorf("Claude credit model registry is not loaded")
	}
	// This lane deliberately does not resolve movable CLI aliases or dated IDs.
	if model != "claude-haiku-5-5" {
		return nil, "", fmt.Errorf("model %q is not allowed for Claude API credits", model)
	}
	m, ok := c.Models[model]
	p := m.Pricing.ClaudeCredit
	if !ok || m.APIName != model || m.Provider != "anthropic" || p == nil {
		return nil, "", fmt.Errorf("Claude credit model has no canonical rate card")
	}
	if !p.BoundsVerified || p.Revision == "" || p.SourceURL == "" || p.MaxInputTokens != 1000000 || p.MaxOutputTokens != 128000 || p.ShortContextLimit != 100000 {
		return nil, "", fmt.Errorf("Claude credit request contract is incomplete or unverified")
	}
	if _, err := time.Parse("2006-01-02", p.VerifiedOn); err != nil {
		return nil, "", fmt.Errorf("Claude credit pricing verification date is invalid")
	}
	for _, r := range []ClaudeCreditRates{p.ShortContext, p.LongContext} {
		for _, rate := range []int64{r.Input, r.Output, r.CacheRead, r.CacheWrite5m, r.CacheWrite1h} {
			if rate <= 0 || rate > math.MaxInt64/(p.MaxInputTokens+p.MaxOutputTokens) {
				return nil, "", fmt.Errorf("Claude credit rate is missing, invalid or too large")
			}
		}
	}
	encoded, err := json.Marshal(p)
	if err != nil {
		return nil, "", fmt.Errorf("encode Claude credit pricing: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return p, p.Revision + ":" + hex.EncodeToString(digest[:]), nil
}

func claudeRoundedCost(counts, rates []int64) (int64, error) {
	var numerator int64
	for i, n := range counts {
		if n != 0 && rates[i] > (math.MaxInt64-numerator)/n {
			return 0, fmt.Errorf("Claude credit cost overflow")
		}
		numerator += n * rates[i]
	}
	const million = int64(1000000)
	amount := numerator / million
	if numerator%million != 0 {
		amount++ // Round the complete request up, never discard fractional micro-USD.
	}
	return amount, nil
}
