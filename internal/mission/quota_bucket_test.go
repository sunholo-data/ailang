package mission

import "testing"

func TestQuotaBucketFor(t *testing.T) {
	for _, tc := range []struct {
		raw, want string
		wantError bool
	}{
		{"opencode", "", true},
		{"OpenCode", "", true},
		{"opencode:openrouter/z-ai/glm-5.3", "openrouter", false},
		{"opencode/ollama/glm-5.3:cloud", "ollama", false},
		{"opencode:anthropic/claude-sonnet-5-5", "anthropic", false},
		{"OpenCode/CoDeX/gpt-5.5", "codex", false},
		{"OPENCODE:OPENROUTER/x", "openrouter", false},
		{"opencode:openai/gpt-5.5", "", true},
		{"opencode:ollama-other/x", "", true},
		{"anthropic", "anthropic", false},
		{"codex", "codex", false},
		{"ollama", "ollama", false},
		{"openrouter", "openrouter", false},
		{"unlabeled", "unlabeled", false},
		{"Anthropic", "Anthropic", false},
		{"opencode-other", "opencode-other", false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := QuotaBucketFor(tc.raw)
			if tc.wantError {
				if err == nil || err.Error() != "opencode is a harness, not a quota bucket; record the backend bucket instead" {
					t.Fatalf("harness diagnostic = %v", err)
				}
			} else if err != nil || got != tc.want {
				t.Fatalf("bucket = %q, err = %v; want %q", got, err, tc.want)
			}
		})
	}
}
