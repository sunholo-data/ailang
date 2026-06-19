package ollama

import (
	"os"
	"testing"
)

func TestResolveOllamaMaxTokens(t *testing.T) {
	os.Unsetenv("AILANG_OLLAMA_MAX_TOKENS")
	cases := []struct {
		name      string
		req, want int
	}{
		{"small default floored (motoko's 4096)", 4096, defaultOllamaMaxTokens},
		{"zero floored", 0, defaultOllamaMaxTokens},
		{"already above floor kept", 32768, 32768},
		{"exactly floor kept", defaultOllamaMaxTokens, defaultOllamaMaxTokens},
	}
	for _, c := range cases {
		if got := resolveOllamaMaxTokens(c.req); got != c.want {
			t.Errorf("%s: resolveOllamaMaxTokens(%d)=%d, want %d", c.name, c.req, got, c.want)
		}
	}
	// env override wins (per-model value from the registry)
	os.Setenv("AILANG_OLLAMA_MAX_TOKENS", "65536")
	defer os.Unsetenv("AILANG_OLLAMA_MAX_TOKENS")
	if got := resolveOllamaMaxTokens(4096); got != 65536 {
		t.Errorf("env override: got %d, want 65536", got)
	}
}
