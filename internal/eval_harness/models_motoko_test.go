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

// TestMotokoModelsInAgentSuite verifies the 4 motoko entries are members of
// agent_suite — without this, `ailang eval-suite --models agent_suite` would
// silently exclude motoko from the threshold-measurement experiment that is
// the strategic point of the M-MOTOKO-EXECUTOR-ADAPTER sprint.
func TestMotokoModelsInAgentSuite(t *testing.T) {
	cfg, err := LoadModelsConfig("models.yml")
	if err != nil {
		t.Fatalf("LoadModelsConfig: %v", err)
	}

	suite := cfg.GetAgentSuite()
	suiteSet := make(map[string]bool, len(suite))
	for _, name := range suite {
		suiteSet[name] = true
	}

	wanted := []string{
		"motoko-claude-haiku-4-5",
		"motoko-claude-sonnet-4-6",
		"motoko-glm-5",
		"motoko-gemma-4",
	}
	for _, name := range wanted {
		if !suiteSet[name] {
			t.Errorf("agent_suite missing %q", name)
		}
	}
}
