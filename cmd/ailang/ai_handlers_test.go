package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

func TestAIHandlerCodexRefusal(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "metered-key")
	for _, name := range []string{"codex-max", "codex:gpt-6.1-sol"} {
		ctx := effects.NewEffContext(nil)
		err := setupAIHandlerDirect(ctx, name, nil, nil, 0)
		if err == nil || !strings.Contains(err.Error(), "#903") || !strings.Contains(err.Error(), "chatgpt/") || ctx.AI != nil {
			t.Fatalf("direct %s: %v", name, err)
		}
	}
	ctx := effects.NewEffContext(nil)
	err := setupAIHandlerFromConfig(ctx, &eval_harness.ModelConfig{Provider: "codex", APIName: "gpt-6.1-sol"}, "codex-row", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "#903") || ctx.AI != nil {
		t.Fatalf("registry: %v", err)
	}
	// An explicit metered registry row retains its chosen provider even with a codex API name.
	ctx = effects.NewEffContext(nil)
	err = setupAIHandlerFromConfig(ctx, &eval_harness.ModelConfig{Provider: "openai", APIName: "codex-max"}, "metered-row", nil, nil)
	if err != nil || ctx.AI == nil {
		t.Fatalf("explicit openai: %v", err)
	}
}

func TestAIHandlerCodexMeteredRegistryRow(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "metered-key")
	cfg, err := eval_harness.LoadModelsConfig(filepath.Join("..", "..", "internal", "modelreg", "models.yml"))
	if err != nil {
		t.Fatal(err)
	}
	model, err := cfg.GetModel("gpt5-2-codex")
	if err != nil {
		t.Fatal(err)
	}
	ctx := effects.NewEffContext(nil)
	if err := setupAIHandlerFromConfig(ctx, model, "gpt5-2-codex", nil, nil); err != nil || ctx.AI == nil {
		t.Fatalf("metered registry row: %v", err)
	}
}
