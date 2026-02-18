// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sunholo/ailang/internal/eval_harness"
)

// pricingConfig holds the loaded pricing configuration from models.yml.
// It's loaded once at startup and cached for performance.
var (
	pricingConfig     *eval_harness.ModelsConfig
	pricingConfigOnce sync.Once
)

// initPricing loads the models.yml pricing configuration.
// Called automatically on first use of CalculateCostFromTokens.
func initPricing() {
	pricingConfigOnce.Do(func() {
		// Search paths for models.yml
		searchPaths := []string{
			"internal/eval_harness/models.yml",
			"../internal/eval_harness/models.yml",
			"../../internal/eval_harness/models.yml",
		}

		// Also try from home directory for installed binaries
		if home, err := os.UserHomeDir(); err == nil {
			// Check go/src path for development
			searchPaths = append(searchPaths, filepath.Join(home, "go/src/github.com/sunholo/ailang/internal/eval_harness/models.yml"))
		}

		// Try cwd-based paths first
		if cwd, err := os.Getwd(); err == nil {
			for _, rel := range []string{
				"internal/eval_harness/models.yml",
				"../internal/eval_harness/models.yml",
				"../../internal/eval_harness/models.yml",
			} {
				searchPaths = append(searchPaths, filepath.Join(cwd, rel))
			}
		}

		for _, path := range searchPaths {
			config, err := eval_harness.LoadModelsConfig(path)
			if err == nil {
				pricingConfig = config
				fmt.Printf("observatory: loaded pricing config from %s (%d models)\n", path, len(config.Models))
				return
			}
		}

		fmt.Printf("observatory: WARNING: pricing config not loaded (models.yml not found), costs will show $0.00\n")
	})
}

// CalculateCostFromTokens calculates cost from model name and token counts.
// Returns 0.0 if:
//   - Pricing config not loaded
//   - Model not found in config
//   - Tokens are zero
//
// This follows the "no silent fallbacks" principle - return 0 rather than guess.
func CalculateCostFromTokens(model string, tokensIn, tokensOut int64) float64 {
	if tokensIn == 0 && tokensOut == 0 {
		return 0.0
	}

	initPricing()

	if pricingConfig == nil {
		return 0.0
	}

	// Try exact match first
	cost, err := pricingConfig.CalculateCostForModel(model, int(tokensIn), int(tokensOut))
	if err == nil {
		return cost
	}

	// Try normalized model name (strip date suffixes, etc.)
	normalizedModel := normalizeModelName(model)
	if normalizedModel != model {
		cost, err = pricingConfig.CalculateCostForModel(normalizedModel, int(tokensIn), int(tokensOut))
		if err == nil {
			return cost
		}
	}

	// Model not found - return 0 (no silent fallbacks per CLAUDE.md)
	return 0.0
}

// normalizeModelName normalizes model names to match models.yml keys.
// Examples:
//   - "claude-sonnet-4-5-20250929" -> "claude-sonnet-4-5"
//   - "claude-sonnet-4-6" -> "claude-sonnet-4-6"
//   - "gpt-5" -> "gpt5"
//   - "gemini-2.5-pro" -> "gemini-2-5-pro"
func normalizeModelName(model string) string {
	if model == "" {
		return ""
	}

	// Strip date suffixes (YYYYMMDD format)
	// Pattern: -YYYYMMDD at end
	if len(model) > 9 {
		suffix := model[len(model)-9:]
		if suffix[0] == '-' && isAllDigits(suffix[1:]) {
			model = model[:len(model)-9]
		}
	}

	// Map common API names to friendly names
	modelMappings := map[string]string{
		// OpenAI
		"gpt-5":               "gpt5",
		"gpt-5-mini":          "gpt5-mini",
		"gpt-5.1":             "gpt5-1",
		"gpt-5.1-chat-latest": "gpt5-1-instant",
		"gpt-5.2":             "gpt5-2",
		"gpt-5.2-chat-latest": "gpt5-2-instant",

		// Claude - also handle versioned names
		"claude-sonnet-4-6":          "claude-sonnet-4-6",
		"claude-sonnet-4-5":          "claude-sonnet-4-5",
		"claude-haiku-4-5":           "claude-haiku-4-5",
		"claude-opus-4-5":            "claude-opus-4-5",
		"claude-opus-4-6":            "claude-opus-4-6",
		"claude-sonnet-4-5-20250929": "claude-sonnet-4-5",
		"claude-haiku-4-5-20251001":  "claude-haiku-4-5",
		"claude-opus-4-5-20251101":   "claude-opus-4-5",

		// Gemini
		"gemini-2.5-pro":         "gemini-2-5-pro",
		"gemini-2.5-flash":       "gemini-2-5-flash",
		"gemini-3-flash-preview": "gemini-3-flash",
		"gemini-3-pro-preview":   "gemini-3-pro",
	}

	if mapped, ok := modelMappings[model]; ok {
		return mapped
	}

	// Try replacing dots with dashes for Gemini-style versions
	normalized := strings.ReplaceAll(model, ".", "-")
	if normalized != model {
		if _, ok := modelMappings[normalized]; ok {
			return modelMappings[normalized]
		}
	}

	return model
}

// isAllDigits returns true if the string contains only ASCII digits.
func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// GetPricingConfig returns the loaded pricing configuration.
// Returns nil if not loaded. Used for testing.
func GetPricingConfig() *eval_harness.ModelsConfig {
	initPricing()
	return pricingConfig
}

// ResetPricingConfig resets the pricing config for testing.
// NOT for production use.
func ResetPricingConfig() {
	pricingConfigOnce = sync.Once{}
	pricingConfig = nil
}

// CalculateCostFromTokensWithCache calculates cost including cache token pricing.
// Cache read tokens are charged at 10% of input price (90% discount).
// Cache creation tokens are charged at 125% of input price (25% premium).
// Returns 0.0 if model not found or tokens are all zero.
func CalculateCostFromTokensWithCache(model string, tokensIn, tokensOut, cacheRead, cacheCreation int64) float64 {
	if tokensIn == 0 && tokensOut == 0 && cacheRead == 0 && cacheCreation == 0 {
		return 0.0
	}

	initPricing()

	if pricingConfig == nil {
		return 0.0
	}

	// Find model config
	normalizedModel := normalizeModelName(model)
	var inputPer1K, outputPer1K float64

	// Try to find model in config
	for modelID, m := range pricingConfig.Models {
		if modelID == model || modelID == normalizedModel || m.APIName == model {
			inputPer1K = m.Pricing.InputPer1K
			outputPer1K = m.Pricing.OutputPer1K
			break
		}
	}

	// If model not found, return 0 (no silent fallbacks)
	if inputPer1K == 0 && outputPer1K == 0 {
		return 0.0
	}

	// Calculate costs
	inputCost := float64(tokensIn) / 1000.0 * inputPer1K
	outputCost := float64(tokensOut) / 1000.0 * outputPer1K

	// Cache read at 10% of input price (90% discount per Anthropic pricing)
	cacheReadCost := float64(cacheRead) / 1000.0 * inputPer1K * 0.1

	// Cache creation at 125% of input price (25% premium per Anthropic pricing)
	cacheCreationCost := float64(cacheCreation) / 1000.0 * inputPer1K * 1.25

	return inputCost + outputCost + cacheReadCost + cacheCreationCost
}

// CalculateCacheSavings calculates how much was saved by using cache reads.
// Returns the difference between what full-price input would have cost and cache read cost.
// Cache reads are 90% cheaper than regular input tokens.
func CalculateCacheSavings(model string, cacheRead int64) float64 {
	if cacheRead == 0 {
		return 0.0
	}

	initPricing()

	if pricingConfig == nil {
		return 0.0
	}

	// Find model config
	normalizedModel := normalizeModelName(model)
	var inputPer1K float64

	for modelID, m := range pricingConfig.Models {
		if modelID == model || modelID == normalizedModel || m.APIName == model {
			inputPer1K = m.Pricing.InputPer1K
			break
		}
	}

	if inputPer1K == 0 {
		return 0.0
	}

	// Full price would be: cacheRead * inputPer1K / 1000
	// Cache price is: cacheRead * inputPer1K * 0.1 / 1000
	// Savings is: cacheRead * inputPer1K * 0.9 / 1000
	return float64(cacheRead) / 1000.0 * inputPer1K * 0.9
}
