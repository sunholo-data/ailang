package modelreg

import "testing"

// M-LYCEUM-PROVIDER M1: `provider: "lyceum"` rows must classify as dual-mode
// (standard-eval capable), not agent-only — otherwise standard-mode runs are
// blocked with the 2026-05-23 "agent-only models" error and the route is
// unreachable from the eval harness.
func TestSupportsStandardEval_Lyceum(t *testing.T) {
	yaml := `
models:
  lyceum-glm-5-3-flash:
    api_name: "z-ai/glm-5.3-flash"
    provider: "lyceum"
    env_var: "LYCEUM_API_KEY"
    default_thinking: "always_on"
    pricing:
      input_per_1k: 0.0002
      output_per_1k: 0.0005
  control-ollama:
    api_name: "llama3"
    provider: "ollama"
    default_thinking: "none"
`
	cfg, err := LoadModelsConfigBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("LoadModelsConfigBytes failed: %v", err)
	}

	if !cfg.SupportsStandardEval("lyceum-glm-5-3-flash") {
		t.Error("SupportsStandardEval(lyceum row) = false, want true (dual-mode cloud provider)")
	}
	if cfg.SupportsStandardEval("control-ollama") {
		t.Error("SupportsStandardEval(ollama row) = true, want false (control)")
	}
}
