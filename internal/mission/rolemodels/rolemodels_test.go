package rolemodels

import "testing"

func TestCanonicalKey(t *testing.T) {
	for _, id := range []string{"gpt-6.1-sol", "gpt6-1-sol", "codex:gpt-6.1-sol", "openai/gpt-6.1-sol", "GPT-6.1-SOL", "CODEX:OPENAI/GPT_6.1_SOL"} {
		if got := CanonicalKey(id); got != "gpt61sol" {
			t.Errorf("CanonicalKey(%q) = %q, want gpt61sol", id, got)
		}
	}
	if CanonicalKey("gpt-6-astra") != CanonicalKey("gpt6-astra") {
		t.Error("astra aliases must canonicalize equally")
	}
	if CanonicalKey("gpt-6-sol") == CanonicalKey("gpt-6.1-sol") {
		t.Error("retired sol must differ from 6.1-sol")
	}
	if CanonicalKey("a:b:gpt-6.1-sol") != "b:gpt61sol" {
		t.Error("strip exactly one provider prefix")
	}
}

func TestAllowed(t *testing.T) {
	for _, id := range []string{"gpt-6.1-sol", "gpt6-1-sol", "codex:gpt-6.1-sol", "openai/gpt-6.1-sol", "gpt-6-luna", "codex:gpt-6-luna", "gpt6-luna", "claude-opus-5-5", "kimi-k3", "glm-5.3", "gemini-4-argon"} {
		if !Allowed(id) {
			t.Errorf("allowed model %q refused", id)
		}
	}
	for _, id := range []string{"gpt-6-astra", "gpt6-astra", "codex:gpt-6-sol", "gpt-5.6-luna", "gpt5-6-luna", "gpt-5.6-sol"} {
		if Allowed(id) {
			t.Errorf("retired OpenAI model %q allowed", id)
		}
	}
}

func TestIsOpenAI(t *testing.T) {
	for _, id := range []string{"GPT6-1-SOL", "openai/gpt-6-sol", "provider:my-astra", "CODEX:unknown"} {
		if !IsOpenAI(id) {
			t.Errorf("OpenAI recognition missed %q", id)
		}
	}
	for _, id := range []string{"claude-opus-5-5", "kimi-k3", "glm-5.3", "gemini-4-argon"} {
		if IsOpenAI(id) {
			t.Errorf("non-OpenAI model %q misclassified", id)
		}
	}
}
