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
