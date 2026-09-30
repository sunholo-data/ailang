package eval_harness

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/modelreg"
	"testing"
)

func TestModelFamilyParsed(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("InitModelsConfig: %v", err)
	}
	cases := []struct {
		key        string
		wantFamily string
	}{
		{"claude-sonnet-4-6", "claude-sonnet-4-6"},
		{"opencode-sonnet-4-6", "claude-sonnet-4-6"},
		{"gemini-3-flash", "gemini-3-flash"},
		{"opencode-gemini-3-flash", "gemini-3-flash"},
		{"claude-haiku-4-5", "claude-haiku-4-5"},
		{"opencode-haiku", "claude-haiku-4-5"},
	}
	for _, tc := range cases {
		cfg, ok := modelreg.GlobalModelsConfig.Models[tc.key]
		if !ok {
			t.Errorf("model %q not found in models.yml", tc.key)
			continue
		}
		if cfg.ModelFamily != tc.wantFamily {
			t.Errorf("model %q: model_family=%q, want %q", tc.key, cfg.ModelFamily, tc.wantFamily)
		}
	}
	// harness_suite went 7 -> 8 in v0.18.0 (motoko-claude-sonnet-4-6) and back to
	// 7 on 2026-09-29, when that row was retired: it reached Claude via OpenRouter,
	// which TestModels_FirstPartyVendorsNeverViaOpenRouter now forbids.
	if len(modelreg.GlobalModelsConfig.HarnessSuite) != 7 {
		t.Errorf("harness_suite: expected 7 models, got %d: %v", len(modelreg.GlobalModelsConfig.HarnessSuite), modelreg.GlobalModelsConfig.HarnessSuite)
	}
}

func TestTTFTConfigParsed(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("InitModelsConfig: %v", err)
	}
	cfg, ok := modelreg.GlobalModelsConfig.Models["opencode-gemma4-e4b"]
	if !ok {
		t.Fatal("opencode-gemma4-e4b not found")
	}
	fmt.Printf("TTFTTimeoutSeconds: %d\n", cfg.TTFTTimeoutSeconds)
	fmt.Printf("GenerationTimeoutSeconds: %d\n", cfg.GenerationTimeoutSeconds)
	if cfg.TTFTTimeoutSeconds != 300 {
		t.Errorf("expected TTFTTimeoutSeconds=300, got %d", cfg.TTFTTimeoutSeconds)
	}
	if cfg.GenerationTimeoutSeconds != 120 {
		t.Errorf("expected GenerationTimeoutSeconds=120, got %d", cfg.GenerationTimeoutSeconds)
	}
	fmt.Printf("OllamaSuite: %v\n", modelreg.GlobalModelsConfig.OllamaSuite)
}
