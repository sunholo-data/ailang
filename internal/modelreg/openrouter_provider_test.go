package modelreg

import (
	"slices"
	"strings"
	"testing"
)

// One openrouter_provider block per api_name applies to every row sharing it;
// a conflicting second block, or a block on a non-openrouter row, is refused.
func TestOpenRouterProvider(t *testing.T) {
	c := &ModelsConfig{Models: map[string]ModelConfig{
		"a": {APIName: "v/m", Provider: "openrouter", OpenRouterProvider: map[string]any{"sort": "throughput"}},
		"b": {APIName: "v/m", Provider: "openrouter"},
	}}
	if got := c.OpenRouterProviderByAPIName()["v/m"]["sort"]; got != "throughput" {
		t.Fatalf("block for v/m = %v, want sort=throughput", got)
	}
	c.Models["b"] = ModelConfig{APIName: "v/m", Provider: "openrouter", OpenRouterProvider: map[string]any{"sort": "price"}}
	if p := c.validateOpenRouterProvider([]string{"a", "b"}); len(p) != 1 || !strings.Contains(p[0], "different values") {
		t.Fatalf("conflicting blocks must be refused, got %v", p)
	}
	c.Models["b"] = ModelConfig{APIName: "v/x", Provider: "ollama", OpenRouterProvider: map[string]any{"sort": "price"}}
	if p := c.validateOpenRouterProvider([]string{"a", "b"}); len(p) != 1 || !strings.Contains(p[0], "provider is") {
		t.Fatalf("a non-openrouter row with a block must be refused, got %v", p)
	}
}

// The shipped registry routes GLM-5.3-Flash over the hosts pi actually reaches:
// those accepting max output >= 943717 (so not inference-net), minus the two
// slow cheapest ones (open-inference, morph) that price weighting inside `only`
// sent motoko to and pi never reached (glm-5.3-flash A/B, 2026-10-16).
func TestShippedGLMFlashRouting(t *testing.T) {
	shipped, err := LoadModelsConfigBytes(embeddedModelsYAML)
	if err != nil {
		t.Fatal(err)
	}
	if p := shipped.validateOpenRouterProvider(func() []string {
		var n []string
		for k := range shipped.Models {
			n = append(n, k)
		}
		slices.Sort(n)
		return n
	}()); len(p) != 0 {
		t.Fatalf("shipped registry: %v", p)
	}
	only, _ := shipped.OpenRouterProviderByAPIName()["z-ai/glm-5.3-flash"]["only"].([]any)
	var hosts []string
	for _, h := range only {
		hosts = append(hosts, h.(string))
	}
	if !slices.Contains(hosts, "crusoe") || len(hosts) != 11 ||
		slices.Contains(hosts, "inference-net") || slices.Contains(hosts, "open-inference") || slices.Contains(hosts, "morph") {
		t.Fatalf("glm-5.3-flash only = %v, want the 11 hosts pi reaches (no inference-net, open-inference, morph)", hosts)
	}
}
