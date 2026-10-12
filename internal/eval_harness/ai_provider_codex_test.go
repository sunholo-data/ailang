package eval_harness

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/openai"
)

func TestProviderAdapterCodexRouting(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "metered-key")
	for _, name := range []string{"codex-max", "codex:gpt-6.1-sol"} {
		p, err := newProviderAdapter(name, "")
		if p != nil || err == nil || !strings.Contains(err.Error(), "#903") || !strings.Contains(err.Error(), "chatgpt/") {
			t.Fatalf("%s: %v %v", name, p, err)
		}
	}
	if p, err := newProviderAdapter("gpt-6.1-sol", ai.ProviderCodex); p != nil || err == nil || !strings.Contains(err.Error(), "#903") {
		t.Fatalf("explicit codex: %v %v", p, err)
	}
	p, err := newProviderAdapter("codex-max", ai.ProviderOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.provider.(*openai.Client); !ok {
		t.Fatalf("explicit metered provider: %T", p.provider)
	}
}
