package modelreg

import (
	"slices"
	"testing"
)

// openrouter_ignore is unioned across rows that share an api_name, and the
// shipped registry keeps GLM-5.3-Flash off inference-net.
func TestOpenRouterIgnoreByAPIName(t *testing.T) {
	c := &ModelsConfig{Models: map[string]ModelConfig{
		"a": {APIName: "v/m", Provider: "openrouter", OpenRouterIgnore: []string{"x"}},
		"b": {APIName: "v/m", Provider: "openrouter", OpenRouterIgnore: []string{"x", "y"}},
		"c": {APIName: "v/m", Provider: "ollama", OpenRouterIgnore: []string{"z"}},
	}}
	got := c.OpenRouterIgnoreByAPIName()["v/m"]
	slices.Sort(got)
	if !slices.Equal(got, []string{"x", "y"}) {
		t.Fatalf("union = %v, want [x y] (non-openrouter rows excluded)", got)
	}
	shipped, err := LoadModelsConfigBytes(embeddedModelsYAML)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(shipped.OpenRouterIgnoreByAPIName()["z-ai/glm-5.3-flash"], "inference-net") {
		t.Fatal("the shipped registry must keep GLM-5.3-Flash off inference-net")
	}
}
