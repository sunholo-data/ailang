package main

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

func TestCodexAIHandlerRefusal(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-metered")
	for _, model := range []string{"codex-max", "codex:gpt-6.1-sol"} {
		err := setupAIHandlerDirect(&effects.EffContext{}, model, nil, nil, 0)
		if err == nil || !strings.Contains(err.Error(), "chatgpt/") || !strings.Contains(err.Error(), "#903") {
			t.Fatalf("%s: %v", model, err)
		}
	}
	err := setupAIHandlerFromConfig(&effects.EffContext{}, &eval_harness.ModelConfig{Provider: "codex"}, "codex:gpt-6.1-sol", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "chatgpt/") {
		t.Fatalf("registry path: %v", err)
	}
}
