package eval_harness

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

func TestCodexAdapterRefusesGuessPreservesExplicitOpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-metered")
	_, err := newProviderAdapter("codex-max", "")
	if err == nil || !strings.Contains(err.Error(), "chatgpt/") || !strings.Contains(err.Error(), "codex-max") {
		t.Fatalf("guessed lane: %v", err)
	}
	p, err := newProviderAdapter("codex-max", ai.ProviderOpenAI)
	if err != nil || p.providerType != ai.ProviderOpenAI {
		t.Fatalf("explicit metered lane: %v", err)
	}
}
