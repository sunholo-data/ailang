package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

// Only the single GPU serializes agent rows now. motoko's fixed port 8080 was
// the second cause; since 2026-09-28 every motoko run binds its own ENV_PORT,
// and on 2026-10-02 six concurrent motoko trials scored 6/6 with no collision.
// An ollama-CLOUD motoko row (agent-only, no GPU) must therefore keep its
// --parallel, while a local ollama row of any harness still clamps to 1.
func TestClampConcurrency_OnlyLocalGPUSerializes(t *testing.T) {
	motoko, pi := "motoko", "pi"
	cloudName := "ollama/glm-5:cloud"
	origModels := modelreg.GlobalModelsConfig
	modelreg.GlobalModelsConfig = &modelreg.ModelsConfig{
		Models: map[string]modelreg.ModelConfig{
			"motoko-cloud-glm": {Provider: "ollama", APIName: "glm-5:cloud", AgentCLI: &motoko, AgentModelName: &cloudName},
			"pi-local-qwen":    {Provider: "ollama-rig", APIName: "qwen3.8:27b", AgentCLI: &pi},
		},
	}
	defer func() { modelreg.GlobalModelsConfig = origModels }()

	tests := []struct {
		name   string
		models []string
		want   int
	}{
		{name: "ollama-cloud motoko keeps its parallelism", models: []string{"motoko-cloud-glm"}, want: 6},
		{name: "local GPU row serializes", models: []string{"pi-local-qwen"}, want: 1},
		{name: "a GPU row in the set serializes the set", models: []string{"motoko-cloud-glm", "pi-local-qwen"}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := 6
			clampConcurrencyForSerializedLanes(true, &n, tt.models)
			if n != tt.want {
				t.Fatalf("parallel = %d, want %d", n, tt.want)
			}
		})
	}
}
