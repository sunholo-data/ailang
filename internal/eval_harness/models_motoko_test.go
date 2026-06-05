package eval_harness

import (
	"strings"
	"testing"
)

// TestMotokoModelsResolve verifies the M-MOTOKO-EXECUTOR-ADAPTER models.yml
// entries load cleanly: each model has agent_cli="motoko", a non-empty
// agent_model_name in the openrouter/* form, and the env_var pointing at
// OPENROUTER_API_KEY (NOT ANTHROPIC_API_KEY — Design Freeze cost-control rule).
func TestMotokoModelsResolve(t *testing.T) {
	cfg, err := LoadModelsConfig("models.yml")
	if err != nil {
		t.Fatalf("LoadModelsConfig: %v", err)
	}

	wanted := []string{
		"motoko-claude-haiku-4-5",
		"motoko-claude-sonnet-4-6",
		"motoko-glm-5",
		"motoko-gemma-4",
	}
	for _, name := range wanted {
		t.Run(name, func(t *testing.T) {
			m, err := cfg.GetModel(name)
			if err != nil {
				t.Fatalf("GetModel(%q): %v", name, err)
			}
			if m.AgentCLI == nil || *m.AgentCLI != "motoko" {
				got := "<nil>"
				if m.AgentCLI != nil {
					got = *m.AgentCLI
				}
				t.Errorf("agent_cli = %q, want \"motoko\"", got)
			}
			if m.AgentModelName == nil || !strings.HasPrefix(*m.AgentModelName, "openrouter/") {
				got := "<nil>"
				if m.AgentModelName != nil {
					got = *m.AgentModelName
				}
				t.Errorf("agent_model_name = %q, want openrouter/ prefix (motoko routes via OpenRouter)", got)
			}
			if m.EnvVar != "OPENROUTER_API_KEY" {
				t.Errorf("env_var = %q, want \"OPENROUTER_API_KEY\" (cost-control: NOT ANTHROPIC_API_KEY)", m.EnvVar)
			}
		})
	}
}

// NOTE: TestMotokoModelsInAgentSuite was removed 2026-06-05. The motoko-* entries
// were intentionally pulled from agent_suite in commit 3f52e61c ("the motoko/bun
// harness isn't reliable yet"); the test still asserted their presence and so
// contradicted the deliberate change. Motoko model resolution is covered by
// TestMotokoModelsResolve above; cross-harness membership lives in harness_suite.
