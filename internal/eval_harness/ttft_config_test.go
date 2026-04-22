package eval_harness

import (
	"fmt"
	"testing"
)

func TestTTFTConfigParsed(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("InitModelsConfig: %v", err)
	}
	cfg, ok := GlobalModelsConfig.Models["opencode-gemma4-e4b"]
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
	fmt.Printf("OllamaSuite: %v\n", GlobalModelsConfig.OllamaSuite)
}
