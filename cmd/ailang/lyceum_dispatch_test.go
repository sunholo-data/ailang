package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// E2E: a models.yml row with provider "lyceum" dispatches through
// setupAIHandlerFromConfig to an OpenAI-compatible handler pointed at the
// Lyceum base URL (overridden via LYCEUM_BASE_URL for the stub), authed by
// LYCEUM_API_KEY. This is the load-bearing acceptance test for
// M-LYCEUM-PROVIDER M1: built-in enum → dispatch case → openai transport →
// custom base URL → response → effect handler text.
func TestE2E_SetupAIHandlerFromConfig_DispatchesToLyceum(t *testing.T) {
	var gotPath, gotAuth, gotModel string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		var body struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			gotModel = body.Model
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{"message": {"role": "assistant", "content": "hello from lyceum"}}],
			"usage": {"prompt_tokens": 5, "completion_tokens": 4, "total_tokens": 9}
		}`))
	}))
	defer server.Close()

	t.Setenv("LYCEUM_API_KEY", "lyceum-test-key")
	t.Setenv("LYCEUM_BASE_URL", server.URL)

	effCtx := &effects.EffContext{}
	model := &eval_harness.ModelConfig{
		Provider: "lyceum",
		APIName:  "z-ai/glm-5.3-flash",
		EnvVar:   "LYCEUM_API_KEY",
	}
	if err := setupAIHandlerFromConfig(effCtx, model, "lyceum-glm-5-3-flash", nil, nil); err != nil {
		t.Fatalf("setupAIHandlerFromConfig failed: %v", err)
	}
	if effCtx.AI == nil {
		t.Fatal("effCtx.AI is nil after setup")
	}

	out, err := effCtx.AI.Call("Ping")
	if err != nil {
		t.Fatalf("AI.Call failed: %v", err)
	}
	if out != "hello from lyceum" {
		t.Errorf("output = %q, want hello from lyceum", out)
	}
	if !strings.HasSuffix(gotPath, "/chat/completions") {
		t.Errorf("request path = %q, want suffix /chat/completions", gotPath)
	}
	if gotAuth != "Bearer lyceum-test-key" {
		t.Errorf("server saw Authorization=%q, want Bearer lyceum-test-key", gotAuth)
	}
	if gotModel != "z-ai/glm-5.3-flash" {
		t.Errorf("request model = %q, want z-ai/glm-5.3-flash (api_name from the row)", gotModel)
	}
}

// Missing LYCEUM_API_KEY must fail loudly with the env var NAMED — no silent
// fallback (Critical Principle #2).
func TestSetupAIHandlerFromConfig_LyceumMissingKey(t *testing.T) {
	t.Setenv("LYCEUM_API_KEY", "")
	effCtx := &effects.EffContext{}
	model := &eval_harness.ModelConfig{
		Provider: "lyceum",
		APIName:  "z-ai/glm-5.3-flash",
		EnvVar:   "LYCEUM_API_KEY",
	}
	err := setupAIHandlerFromConfig(effCtx, model, "lyceum-glm-5-3-flash", nil, nil)
	if err == nil {
		t.Fatal("missing LYCEUM_API_KEY: want error, got nil")
	}
	if !strings.Contains(err.Error(), "LYCEUM_API_KEY") {
		t.Errorf("error %q does not name LYCEUM_API_KEY", err.Error())
	}
}

// LYCEUM_BASE_URL unset → the handler must target the real Lyceum endpoint.
// Verified at the unit level: the dispatch must NOT fall back to the default
// OpenAI base URL when the override is absent.
func TestLyceumBaseURLConstant(t *testing.T) {
	t.Setenv("LYCEUM_BASE_URL", "")
	if got := ai.LyceumBaseURL(); got != "https://api.lyceum.technology/openai/v1" {
		t.Errorf("LyceumBaseURL() = %q, want https://api.lyceum.technology/openai/v1", got)
	}
	t.Setenv("LYCEUM_BASE_URL", "http://stub:1234/v1")
	if got := ai.LyceumBaseURL(); got != "http://stub:1234/v1" {
		t.Errorf("LyceumBaseURL() with override = %q, want http://stub:1234/v1", got)
	}
}
