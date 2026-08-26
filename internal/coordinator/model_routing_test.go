package coordinator

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestResolveModel(t *testing.T) {
	routing := ModelRouting{
		"designer": {"opus", "kimi-k3"},
		"executor": {"opus"},
	}

	// Explicit pin wins over role.
	m, err := ResolveModel(&AgentConfig{ID: "a", Model: "sonnet", Role: "designer"}, routing)
	if err != nil || m != "sonnet" {
		t.Errorf("explicit pin must win: %q, %v", m, err)
	}

	// Role resolves to the chain's primary.
	m, err = ResolveModel(&AgentConfig{ID: "b", Role: "designer"}, routing)
	if err != nil || m != "opus" {
		t.Errorf("role must resolve to primary: %q, %v", m, err)
	}

	// No role, no model = provider default (pre-M5 behavior, not an error).
	m, err = ResolveModel(&AgentConfig{ID: "c"}, routing)
	if err != nil || m != "" {
		t.Errorf("unrouted agent must fall through silently: %q, %v", m, err)
	}

	// A role the table does not know is LOUD — the error names the agent, the
	// role, and what the table does have.
	_, err = ResolveModel(&AgentConfig{ID: "d", Role: "judge"}, routing)
	if err == nil {
		t.Fatal("missing role must be an error, not a silent fallback")
	}
	for _, want := range []string{"d", "judge", "designer"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q: %v", want, err)
		}
	}
}

func TestModelRouting_ParsesFromConfig(t *testing.T) {
	var cfg CoordinatorConfig
	src := `
model_routing:
  designer: [opus, kimi-k3]
  evaluator: [sonnet]
agents:
  - id: x
    inbox: x
    workspace: org/repo
    role: designer
`
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(cfg.ModelRouting["designer"]) != 2 {
		t.Errorf("designer chain: %v", cfg.ModelRouting["designer"])
	}
	if cfg.Agents[0].Role != "designer" {
		t.Errorf("role: %q", cfg.Agents[0].Role)
	}
}
