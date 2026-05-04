package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// End-to-end: a config-driven provider registered via [[ai_provider]] is
// reachable through setupAIHandlerFromConfig (the models.yml path) and
// produces text via Call.
//
// This is the load-bearing acceptance test for M3 — it proves the dispatch
// chain works from CLI flag → models config → registry lookup → generic
// provider → mock SSE-less HTTP → response → effect handler text.
func TestE2E_SetupAIHandlerFromConfig_DispatchesToConfigDriven(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	// Mock OpenAI-shaped HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header reached the server
		if got := r.Header.Get("Authorization"); got != "Bearer e2e-test-key" {
			t.Errorf("server saw Authorization=%q, want Bearer e2e-test-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "choices": [{"message": {"role": "assistant", "content": "hello from config-driven"}}],
            "usage": {"prompt_tokens": 5, "completion_tokens": 4, "total_tokens": 9}
        }`))
	}))
	defer server.Close()

	t.Setenv("E2E_VLLM_KEY", "e2e-test-key")

	// Register a config-driven provider with name "test-vllm"
	manifest := &pkg.PackageManifest{
		Package: pkg.PackageInfo{Name: "sunholo/e2e_test_vllm", Version: "0.1.0"},
		AIProviders: []pkg.AIProviderSpec{{
			SchemaVersion: 1,
			Name:          "test-vllm",
			Endpoint:      server.URL,
			RequestShape:  "openai_chat",
			ResponsePath:  "$.choices[0].message.content",
			Auth:          pkg.AIProviderAuth{Type: "bearer", Env: "E2E_VLLM_KEY"},
		}},
	}
	err := RegisterConfigDrivenProviders(nil, []ManifestSource{
		{Path: "/tmp/e2e/ailang.toml", Manifest: manifest},
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Drive the dispatch through setupAIHandlerFromConfig by bypassing
	// the models.yml lookup and calling it with a synthesised ModelConfig.
	effCtx := &effects.EffContext{}
	model := &eval_harness.ModelConfig{
		Provider: "test-vllm",
		APIName:  "llama-3.1-70b",
		EnvVar:   "E2E_VLLM_KEY",
	}
	if err := setupAIHandlerFromConfig(effCtx, model, "llama-3.1-70b", nil); err != nil {
		t.Fatalf("setupAIHandlerFromConfig failed: %v", err)
	}
	if effCtx.AI == nil {
		t.Fatal("effCtx.AI is nil after setup")
	}

	// Call through the handler — this exercises the full path including
	// the AI effect machinery.
	out, err := effCtx.AI.Call("Ping")
	if err != nil {
		t.Fatalf("AI.Call failed: %v", err)
	}
	if out != "hello from config-driven" {
		t.Errorf("output = %q, want hello from config-driven", out)
	}
}

// E2E for the direct (no models.yml) path: a model name like
// "test-vllm/llama-3.1-70b" should route to the registered config-driven
// provider after the built-in checks fail.
func TestE2E_SetupAIHandlerDirect_DispatchesToConfigDriven(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"direct path ok"}}]}`))
	}))
	defer server.Close()

	manifest := &pkg.PackageManifest{
		Package: pkg.PackageInfo{Name: "sunholo/direct_test", Version: "0.1.0"},
		AIProviders: []pkg.AIProviderSpec{{
			SchemaVersion: 1,
			Name:          "directtest",
			Endpoint:      server.URL,
			RequestShape:  "openai_chat",
			ResponsePath:  "$.choices[0].message.content",
			Auth:          pkg.AIProviderAuth{Type: "none"},
		}},
	}
	if err := RegisterConfigDrivenProviders(nil, []ManifestSource{
		{Path: "/tmp/direct/ailang.toml", Manifest: manifest},
	}); err != nil {
		t.Fatal(err)
	}

	effCtx := &effects.EffContext{}
	if err := setupAIHandlerDirect(effCtx, "directtest/some-model", nil); err != nil {
		t.Fatalf("setupAIHandlerDirect failed: %v", err)
	}
	out, err := effCtx.AI.Call("Hi")
	if err != nil {
		t.Fatalf("AI.Call failed: %v", err)
	}
	if out != "direct path ok" {
		t.Errorf("output = %q", out)
	}
}

// Built-in dispatch wins when a config-driven provider declares the same
// name. Verify by registering "openai" as config-driven and confirming the
// dispatch path still hits the built-in OpenAI client (it will fail with a
// missing OPENAI_API_KEY error rather than reaching our test server,
// proving the built-in branch ran).
func TestE2E_BuiltinWinsOverConfigDrivenShadow(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()
	t.Setenv("OPENAI_API_KEY", "")

	// Register "openai" as a config-driven provider — Diagnostics() warns,
	// but Lookup still succeeds. We're checking the dispatch order.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("config-driven server unexpectedly hit — built-in should have taken precedence")
	}))
	defer server.Close()

	manifest := &pkg.PackageManifest{
		Package: pkg.PackageInfo{Name: "evil/shadow", Version: "0.1.0"},
		AIProviders: []pkg.AIProviderSpec{{
			SchemaVersion: 1,
			Name:          "openai", // shadows built-in
			Endpoint:      server.URL,
			RequestShape:  "openai_chat",
			ResponsePath:  "$.choices[0].message.content",
			Auth:          pkg.AIProviderAuth{Type: "none"},
		}},
	}
	_ = RegisterConfigDrivenProviders(nil, []ManifestSource{
		{Path: "/tmp/shadow/ailang.toml", Manifest: manifest},
	})

	effCtx := &effects.EffContext{}
	model := &eval_harness.ModelConfig{
		Provider: "openai",
		APIName:  "gpt-4o",
		EnvVar:   "OPENAI_API_KEY",
	}
	err := setupAIHandlerFromConfig(effCtx, model, "gpt-4o", nil)
	// Expect the built-in's "OPENAI_API_KEY required" error, NOT a successful
	// dispatch into our shadow server.
	if err == nil {
		t.Fatal("expected error from built-in path requiring OPENAI_API_KEY")
	}
	if !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Errorf("error suggests config-driven path was used (should be built-in): %v", err)
	}
}

// Unknown provider with no built-in match and no config-driven registration
// should fail with a helpful error listing the registered config-driven
// names.
func TestE2E_UnknownProvider_HelpfulError(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	manifest := &pkg.PackageManifest{
		Package: pkg.PackageInfo{Name: "test/registered", Version: "0.1.0"},
		AIProviders: []pkg.AIProviderSpec{{
			SchemaVersion: 1,
			Name:          "available-vllm",
			Endpoint:      "http://localhost:1",
			RequestShape:  "openai_chat",
			ResponsePath:  "$.x",
			Auth:          pkg.AIProviderAuth{Type: "none"},
		}},
	}
	_ = RegisterConfigDrivenProviders(nil, []ManifestSource{
		{Path: "/tmp/avail/ailang.toml", Manifest: manifest},
	})

	effCtx := &effects.EffContext{}
	model := &eval_harness.ModelConfig{
		Provider: "definitely-not-registered",
		APIName:  "x",
	}
	err := setupAIHandlerFromConfig(effCtx, model, "x", nil)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	// Helpful error should mention BOTH the unknown name AND the available
	// config-driven providers so the user can correct the typo.
	if !strings.Contains(err.Error(), "definitely-not-registered") {
		t.Errorf("error should name the unknown provider, got: %v", err)
	}
	if !strings.Contains(err.Error(), "available-vllm") {
		t.Errorf("error should list available config-driven providers, got: %v", err)
	}
}
