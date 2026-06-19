package eval_harness

import "testing"

// M-OLLAMA-PER-MODEL-MAX-TOKENS: the registry's declared max_output_tokens must
// flow through modelMaxOutputTokens onto the Task (and on to motoko's env).
func TestModelMaxOutputTokens(t *testing.T) {
	prev := GlobalModelsConfig
	defer func() { GlobalModelsConfig = prev }()

	GlobalModelsConfig = &ModelsConfig{Models: map[string]ModelConfig{
		"motoko-local-qwen3-6-35b-a3b-mxfp8": {MaxOutputTokens: 32768},
		"model-without-max":                  {},
	}}
	if got := modelMaxOutputTokens("motoko-local-qwen3-6-35b-a3b-mxfp8"); got != 32768 {
		t.Errorf("declared model: got %d, want 32768", got)
	}
	if got := modelMaxOutputTokens("model-without-max"); got != 0 {
		t.Errorf("unset max: got %d, want 0", got)
	}
	if got := modelMaxOutputTokens("unknown-model"); got != 0 {
		t.Errorf("unknown model: got %d, want 0", got)
	}

	GlobalModelsConfig = nil
	if got := modelMaxOutputTokens("anything"); got != 0 {
		t.Errorf("nil registry: got %d, want 0", got)
	}
}
