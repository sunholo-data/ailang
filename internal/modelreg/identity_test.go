package modelreg

import "testing"

func TestOriginVendor(t *testing.T) {
	c := &ModelsConfig{Models: map[string]ModelConfig{
		"direct":     {Provider: "openai", APIName: "gpt-test"},
		"router":     {Provider: "openrouter", APIName: "openai/gpt-test"},
		"cloud":      {Provider: "ollama", APIName: "minimax:cloud", ModelVendor: "minimax"},
		"unknown":    {Provider: "ollama", APIName: "minimax:cloud"},
		"bad-router": {Provider: "openrouter", APIName: "unqualified"},
	}}
	for name, want := range map[string]string{"direct": "openai", "router": "openai", "cloud": "minimax"} {
		got, err := c.OriginVendor(name)
		if err != nil || got != want {
			t.Fatalf("%s: got %q, %v; want %q", name, got, err, want)
		}
	}
	for _, name := range []string{"unknown", "bad-router", "missing"} {
		if _, err := c.OriginVendor(name); err == nil {
			t.Errorf("%s: unknown origin accepted", name)
		}
	}
}

func TestDispatchOriginRejectsWireContradictions(t *testing.T) {
	pi := "pi"
	for _, wire := range []string{"openrouter/openai/gpt", "openai/gpt", "unqualified", "unknown/model"} {
		c := &ModelsConfig{Models: map[string]ModelConfig{"judge": {Provider: "openrouter", APIName: "anthropic/claude", AgentCLI: &pi, AgentModelName: &wire}}}
		if _, err := c.DispatchOriginVendor("judge"); err == nil {
			t.Errorf("accepted conflicting/ambiguous wire %q", wire)
		}
	}
	wire := "openrouter/anthropic/claude"
	c := &ModelsConfig{Models: map[string]ModelConfig{"judge": {Provider: "openrouter", APIName: "anthropic/claude", AgentCLI: &pi, AgentModelName: &wire}}}
	if got, err := c.DispatchOriginVendor("judge"); err != nil || got != "anthropic" {
		t.Fatalf("consistent identity: %q %v", got, err)
	}
}
