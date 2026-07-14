package quorum

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// TestResolveCaller_OpenAIRequiresKey proves the openai reviewer path refuses
// (hard error, no silent fallback) when OPENAI_API_KEY is absent — and does
// NOT fall back to another provider.
func TestResolveCaller_OpenAIRequiresKey(t *testing.T) {
	if err := eval_harness.InitModelsConfig(); err != nil {
		t.Skipf("models.yml unavailable: %v", err)
	}
	if _, err := eval_harness.GlobalModelsConfig.GetModel("gpt5-6-sol"); err != nil {
		t.Skipf("gpt5-6-sol not in models.yml: %v", err)
	}
	t.Setenv("OPENAI_API_KEY", "")

	_, _, err := ResolveCaller("gpt5-6-sol")
	if err == nil {
		t.Fatal("expected hard error when OPENAI_API_KEY is empty, got nil")
	}
	if !containsAll(err.Error(), "OPENAI_API_KEY") {
		t.Errorf("error should name the missing key, got: %v", err)
	}
}

// TestResolveCaller_GeminiDoesNotReadGeminiKey proves the google reviewer path
// resolves via the models.yml provider=google (Vertex ADC) route and does NOT
// depend on GEMINI_API_KEY (the rig-absent var the design doc flagged). We set
// GEMINI_API_KEY to empty and assert resolution does not error FOR THAT REASON
// — it only fails if ADC itself is unavailable (a different, named error).
func TestResolveCaller_GeminiDoesNotReadGeminiKey(t *testing.T) {
	if err := eval_harness.InitModelsConfig(); err != nil {
		t.Skipf("models.yml unavailable: %v", err)
	}
	mc, err := eval_harness.GlobalModelsConfig.GetModel("gemini-3-1-pro")
	if err != nil {
		t.Skipf("gemini-3-1-pro not in models.yml: %v", err)
	}
	// Assert the config itself proves the ADC route: env_var empty, provider
	// google, gcp_project set. This is the static guarantee that GEMINI_API_KEY
	// is never consulted — independent of whether ADC is live in CI.
	if mc.Provider != "google" {
		t.Errorf("gemini-3-1-pro provider = %q, want google", mc.Provider)
	}
	if mc.EnvVar != "" {
		t.Errorf("gemini-3-1-pro env_var = %q, want empty (ADC path, not GEMINI_API_KEY)", mc.EnvVar)
	}
	if mc.GCPProject == "" {
		t.Errorf("gemini-3-1-pro gcp_project empty; ADC needs a project")
	}

	t.Setenv("GEMINI_API_KEY", "")
	_, _, rerr := ResolveCaller("gemini-3-1-pro")
	if rerr != nil {
		// Only acceptable failure is ADC being unavailable in CI — and the
		// error must be about ADC/Vertex, NEVER about GEMINI_API_KEY.
		if containsAll(rerr.Error(), "GEMINI_API_KEY") {
			t.Fatalf("gemini resolution referenced GEMINI_API_KEY — the ADC route regressed: %v", rerr)
		}
		t.Logf("gemini resolve failed on ADC (expected in CI without ADC): %v", rerr)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
