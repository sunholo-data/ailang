package quorum

import "testing"

func TestCanonicalModelKey(t *testing.T) {
	for _, id := range []string{"gpt-6.1-sol", "gpt6-1-sol", "codex:gpt-6.1-sol", "openai/gpt-6.1-sol", "GPT-6.1-SOL", "CODEX:OPENAI/GPT_6.1_SOL"} {
		if got := CanonicalModelKey(id); got != "gpt61sol" {
			t.Errorf("CanonicalModelKey(%q) = %q, want gpt61sol", id, got)
		}
	}
	if CanonicalModelKey("gpt-6-astra") != CanonicalModelKey("gpt6-astra") {
		t.Error("astra aliases must canonicalize equally")
	}
	if CanonicalModelKey("gpt-6-sol") == CanonicalModelKey("gpt-6.1-sol") {
		t.Error("retired sol must differ from 6.1-sol")
	}
	if CanonicalModelKey("a:b:gpt-6.1-sol") != "b:gpt61sol" {
		t.Error("strip exactly one provider prefix")
	}
}

func TestModelAllowed(t *testing.T) {
	for _, id := range []string{"gpt-6.1-sol", "gpt6-1-sol", "codex:gpt-6.1-sol", "openai/gpt-6.1-sol", "gpt-6-luna", "codex:gpt-6-luna", "gpt6-luna", "claude-opus-5-5", "kimi-k3", "glm-5.3", "gemini-4-argon"} {
		if !ModelAllowed(id) {
			t.Errorf("allowed model %q refused", id)
		}
	}
	for _, id := range []string{"gpt-6-astra", "gpt6-astra", "codex:gpt-6-sol", "gpt-5.6-luna", "gpt5-6-luna", "gpt-5.6-sol"} {
		if ModelAllowed(id) {
			t.Errorf("retired OpenAI model %q allowed", id)
		}
	}
}

func TestIsOpenAIModel(t *testing.T) {
	for _, id := range []string{"GPT6-1-SOL", "openai/gpt-6-sol", "provider:my-astra", "CODEX:unknown"} {
		if !IsOpenAIModel(id) {
			t.Errorf("OpenAI recognition missed %q", id)
		}
	}
	for _, id := range []string{"claude-opus-5-5", "kimi-k3", "glm-5.3", "gemini-4-argon"} {
		if IsOpenAIModel(id) {
			t.Errorf("non-OpenAI model %q misclassified", id)
		}
	}
}
