package observatory

import (
	"testing"
)

func TestNormalizeModelName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Empty/nil
		{"", ""},

		// OpenAI models
		{"gpt-5", "gpt5"},
		{"gpt-5-mini", "gpt5-mini"},
		{"gpt-5.1", "gpt5-1"},
		{"gpt-5.1-chat-latest", "gpt5-1-instant"},
		{"gpt-5.2", "gpt5-2"},

		// Claude models with date suffixes
		{"claude-sonnet-4-5-20250929", "claude-sonnet-4-5"},
		{"claude-haiku-4-5-20251001", "claude-haiku-4-5"},
		{"claude-opus-4-5-20251101", "claude-opus-4-5"},

		// Claude models without suffixes
		{"claude-sonnet-4-5", "claude-sonnet-4-5"},
		{"claude-haiku-4-5", "claude-haiku-4-5"},
		{"claude-opus-4-5", "claude-opus-4-5"},

		// Gemini models
		{"gemini-2.5-pro", "gemini-2-5-pro"},
		{"gemini-2.5-flash", "gemini-2-5-flash"},
		{"gemini-3-flash-preview", "gemini-3-flash"},
		{"gemini-3-pro-preview", "gemini-3-pro"},

		// Unknown model (pass through)
		{"unknown-model", "unknown-model"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeModelName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeModelName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsAllDigits(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"123", true},
		{"12345678", true},
		{"12a34", false},
		{"abc", false},
		{" 123", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isAllDigits(tt.input)
			if result != tt.expected {
				t.Errorf("isAllDigits(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCalculateCostFromTokens_ZeroTokens(t *testing.T) {
	// Reset for clean test
	ResetPricingConfig()

	// Zero tokens should always return 0 (no calculation needed)
	cost := CalculateCostFromTokens("claude-sonnet-4-5", 0, 0)
	if cost != 0.0 {
		t.Errorf("CalculateCostFromTokens with zero tokens = %f, want 0.0", cost)
	}
}

func TestCalculateCostFromTokens_WithConfig(t *testing.T) {
	// This test requires models.yml to be available
	// Skip if not in development environment
	ResetPricingConfig()

	// Try to load pricing config
	initPricing()
	if pricingConfig == nil {
		t.Skip("models.yml not available in test environment")
	}

	// Test Claude Sonnet pricing
	// From models.yml: input_per_1k: 0.003, output_per_1k: 0.015
	cost := CalculateCostFromTokens("claude-sonnet-4-5", 1000, 1000)
	expectedCost := 0.003 + 0.015 // $0.003 input + $0.015 output = $0.018

	if cost < expectedCost*0.99 || cost > expectedCost*1.01 {
		t.Errorf("CalculateCostFromTokens(claude-sonnet-4-5, 1000, 1000) = %f, want ~%f", cost, expectedCost)
	}

	// Test with API name format (should be normalized)
	cost2 := CalculateCostFromTokens("claude-sonnet-4-5-20250929", 1000, 1000)
	if cost2 < expectedCost*0.99 || cost2 > expectedCost*1.01 {
		t.Errorf("CalculateCostFromTokens(claude-sonnet-4-5-20250929, 1000, 1000) = %f, want ~%f", cost2, expectedCost)
	}
}

func TestCalculateCostFromTokens_UnknownModel(t *testing.T) {
	// Reset for clean test
	ResetPricingConfig()

	// Unknown model should return 0 (no silent fallbacks)
	cost := CalculateCostFromTokens("unknown-nonexistent-model", 1000, 1000)
	if cost != 0.0 {
		t.Errorf("CalculateCostFromTokens with unknown model = %f, want 0.0", cost)
	}
}

func TestCalculateCostFromTokens_EmptyModel(t *testing.T) {
	// Empty model should return 0
	cost := CalculateCostFromTokens("", 1000, 1000)
	if cost != 0.0 {
		t.Errorf("CalculateCostFromTokens with empty model = %f, want 0.0", cost)
	}
}
