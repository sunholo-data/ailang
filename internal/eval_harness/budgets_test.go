package eval_harness

import (
	"testing"
)

// TestResolvedMaxCostUSD verifies the per-model cost-budget resolution
// path: explicit Budgets.MaxCostUSD wins, otherwise default formula
// min($0.50, input × 64 + output × 32) applies.
func TestResolvedMaxCostUSD(t *testing.T) {
	tests := []struct {
		name string
		m    ModelConfig
		want float64
	}{
		{
			name: "explicit budget wins over default formula",
			m: ModelConfig{
				Pricing: Pricing{InputPer1K: 0.015, OutputPer1K: 0.075}, // claude-opus tier
				Budgets: Budgets{MaxCostUSD: 0.80},
			},
			want: 0.80,
		},
		{
			name: "no explicit budget — cheap model uses formula",
			m: ModelConfig{
				// or-minimax-m2-7
				Pricing: Pricing{InputPer1K: 0.0003, OutputPer1K: 0.0012},
			},
			// formula = 0.0003*64 + 0.0012*32 = 0.0192 + 0.0384 = 0.0576
			want: 0.0576,
		},
		{
			name: "no explicit budget — pricey model clipped to ceiling",
			m: ModelConfig{
				Pricing: Pricing{InputPer1K: 0.015, OutputPer1K: 0.075},
			},
			want: 0.50, // formula = 3.36, clipped
		},
		{
			name: "free model (zero pricing) returns 0",
			m: ModelConfig{
				Pricing: Pricing{InputPer1K: 0, OutputPer1K: 0},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.m.ResolvedMaxCostUSD()
			if got != tt.want {
				t.Errorf("ResolvedMaxCostUSD() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestResolvedHardTimeoutSecs verifies the wall-clock safety-net default.
func TestResolvedHardTimeoutSecs(t *testing.T) {
	tests := []struct {
		name string
		m    ModelConfig
		want int
	}{
		{
			name: "explicit timeout wins",
			m:    ModelConfig{Budgets: Budgets{HardTimeoutSecs: 300}},
			want: 300,
		},
		{
			name: "default 600s when omitted",
			m:    ModelConfig{},
			want: 600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.m.ResolvedHardTimeoutSecs()
			if got != tt.want {
				t.Errorf("ResolvedHardTimeoutSecs() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRealModelsYmlBackCompat ensures the existing models.yml (no budgets:
// blocks) still parses cleanly and produces sensible default cost ceilings.
func TestRealModelsYmlBackCompat(t *testing.T) {
	cfg, err := LoadModelsConfig("models.yml")
	if err != nil {
		t.Fatalf("LoadModelsConfig: %v", err)
	}
	if cfg == nil || len(cfg.Models) == 0 {
		t.Fatal("expected models loaded from models.yml")
	}

	// Spot-check a known-cheap model, a known-expensive model, and a
	// known-free local model. They should all resolve to sensible defaults.
	// Use models WITHOUT explicit budgets: blocks so we exercise the
	// default-formula path. (or-glm-5 has explicit $0.30 as of v0.15.1.)
	checks := []struct {
		key       string
		minBudget float64
		maxBudget float64
	}{
		// Free local models: ResolvedMaxCostUSD should be 0
		{key: "ollama-codellama", minBudget: 0, maxBudget: 0},
		// Mid-tier OpenAI cheap (no explicit budget) — formula resolves
		// to ~$0.19 (input × 64K + output × 32K, capped at $0.50).
		{key: "gpt5-4-mini", minBudget: 0.05, maxBudget: 0.50},
		// Pricey flagship — formula × 64K input × 32K output far
		// exceeds $0.50 ceiling, so resolves to clipped value.
		{key: "claude-opus-4-7", minBudget: 0.50, maxBudget: 0.50},
	}

	for _, c := range checks {
		mc, ok := cfg.Models[c.key]
		if !ok {
			t.Logf("model %s not in registry (skipping)", c.key)
			continue
		}
		got := mc.ResolvedMaxCostUSD()
		if got < c.minBudget || got > c.maxBudget+1e-9 {
			t.Errorf("model %s: ResolvedMaxCostUSD()=%v, want in [%v, %v]",
				c.key, got, c.minBudget, c.maxBudget)
		}
	}
}
