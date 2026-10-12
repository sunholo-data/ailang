package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/configdriven"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/modelreg"
	"github.com/sunholo-data/ailang/internal/pkg"
)

func TestPrepareRunAIHandler_KeySourceFrozenAndFailureCached(t *testing.T) {
	t.Setenv("AILANG_AI_KEY_FILE", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	missing := filepath.Join(t.TempDir(), "missing.key")
	t.Setenv("AILANG_AI_KEY_FILE", missing)
	ctx := effects.NewEffContext(nil)
	if err := prepareRunAIHandler(ctx, aiSetup{Model: "z-ai/glm-5.3-flash"}, nil, nil); err != nil {
		t.Fatalf("preparation read credential: %v", err)
	}
	t.Setenv("AILANG_AI_KEY_FILE", filepath.Join(t.TempDir(), "different.key"))
	_, first := ctx.AI.Call("hi")
	if first == nil || !strings.Contains(first.Error(), missing) {
		t.Fatalf("selected source changed: %v", first)
	}
	if err := os.WriteFile(missing, []byte("secret-never-log\ninvalid-extra-line"), 0600); err != nil {
		t.Fatal(err)
	}
	_, second := ctx.AI.Call("hi")
	ctx.Grant(effects.NewCapability("AI"))
	result, resultErr := effects.Call(ctx, "AI", "callResult", []eval.Value{&eval.StringValue{Value: "hi"}})
	if resultErr != nil || !strings.Contains(result.String(), "AuthFailed") {
		t.Fatalf("first-effect auth lost Result error shape: %v %v", result, resultErr)
	}
	if first != second || strings.Contains(second.Error(), "secret-never-log") {
		t.Fatalf("failure not cached or secret exposed: %v", second)
	}
	// A separate run has its own initializer/error, with its newly selected path.
	other := effects.NewEffContext(nil)
	if err := prepareRunAIHandler(other, aiSetup{Model: "z-ai/glm-5.3-flash"}, nil, nil); err != nil {
		t.Fatal(err)
	}
	_, err := other.AI.Call("hi")
	if err == first || !strings.Contains(err.Error(), "different.key") {
		t.Fatal("runs shared initialization state")
	}
}

func TestPrepareRunAIHandler_ModelProviderAndOptionsFrozen(t *testing.T) {
	t.Setenv("AILANG_AI_KEY_FILE", "")
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["max_tokens"] != float64(37) {
			t.Errorf("output budget changed: %v", body)
		}
		if body["model"] != "frozen-model" {
			t.Errorf("model changed: %v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"text":"mock response"}`))
	}))
	defer server.Close()
	oldRegistry, oldModels := ai.GlobalProviderRegistry, modelreg.GlobalModelsConfig
	t.Cleanup(func() { ai.GlobalProviderRegistry = oldRegistry; modelreg.GlobalModelsConfig = oldModels })
	ai.GlobalProviderRegistry = ai.NewProviderRegistry()
	spec := &pkg.AIProviderSpec{Name: "lazy-fixture", Endpoint: server.URL, RequestShape: "simple_completion", ResponsePath: "$.text", Auth: pkg.AIProviderAuth{Type: "none"}, Models: pkg.AIProviderModels{Allowed: []string{"frozen-model"}}}
	if err := ai.GlobalProviderRegistry.Register(spec.Name, configdriven.New(spec), "test fixture"); err != nil {
		t.Fatal(err)
	}
	modelsPath := filepath.Join(t.TempDir(), "models.yml")
	if err := os.WriteFile(modelsPath, []byte(`models:
  lazy-selected:
    provider: lazy-fixture
    api_name: frozen-model
    max_output_tokens: 37
  lazy-step:
    provider: lazy-fixture
    api_name: frozen-model
  other:
    provider: openai
    api_name: other
`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(modelreg.ModelsPathEnv, modelsPath)
	ctx := effects.NewEffContext(nil)
	routing := &ai.AIRoutingPolicy{}
	if err := prepareRunAIHandler(ctx, aiSetup{Model: "lazy-selected"}, routing, nil); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 0 {
		t.Fatal("preparation called provider")
	}
	// Replace global configuration and mutate the old spec before first use.
	spec.Endpoint = "http://127.0.0.1:1"
	spec.Models.Allowed[0] = "forbidden"
	routing.AllowFallback = true
	modelreg.GlobalModelsConfig.Models["lazy-step"] = modelreg.ModelConfig{Provider: "openai", APIName: "changed"}
	ai.GlobalProviderRegistry = ai.NewProviderRegistry()
	if got, err := ctx.AI.Call("hi"); err != nil || got != "mock response" {
		t.Fatalf("first effect: %q %v", got, err)
	}
	// Config-driven Step is deliberately unsupported today. A frozen same-provider
	// name still reaches that existing refusal, rather than changing provider.
	if _, err := ctx.AI.Step("lazy-step", nil, nil); err == nil || !strings.Contains(err.Error(), "Step not yet implemented") {
		t.Fatalf("frozen resolver: %v", err)
	}
	if got, err := ctx.AI.Call("again"); err != nil || got != "mock response" {
		t.Fatalf("second mock request: %q %v", got, err)
	}
	if requests.Load() != 2 {
		t.Fatalf("expected exactly two local requests, got %d", requests.Load())
	}
	if _, err := ctx.AI.Step("other", nil, nil); err == nil {
		t.Fatal("cross-provider model accepted")
	}
	if requests.Load() != 2 {
		t.Fatal("cross-provider refusal called mock")
	}
}

func TestPrepareRunAIHandler_NoADCAndStructuralValidation(t *testing.T) {
	for _, name := range []string{"GOOGLE_API_KEY", "GEMINI_API_KEY", "AILANG_AI_KEY_FILE"} {
		t.Setenv(name, "")
	}
	t.Setenv("AILANG_AI_NO_ADC", "1")
	ctx := effects.NewEffContext(nil)
	if err := prepareRunAIHandler(ctx, aiSetup{Model: "gemini-2-5-flash"}, nil, nil); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_AI_NO_ADC", "")
	t.Setenv("GOOGLE_API_KEY", "late-key-must-not-select-new-lane")
	_, err := ctx.AI.Call("hi")
	var typed *ai.AIError
	if !errors.As(err, &typed) || typed.Code != ai.CodeAuthFailed {
		t.Fatalf("frozen NoADC/auth choice: %v", err)
	}
	for _, cfg := range []aiSetup{{Model: "cannot-guess-this-model"}, {Model: "gpt-4o", StubFixtures: "missing"}, {Stub: true, StubFixtures: "missing"}} {
		if err := prepareRunAIHandler(effects.NewEffContext(nil), cfg, nil, nil); err == nil {
			t.Fatalf("structural configuration accepted: %+v", cfg)
		}
	}
}

func TestPrepareRunAIHandler_RelativeKeyPathFrozen(t *testing.T) {
	t.Setenv("AILANG_AI_KEY_FILE", "")
	selected := t.TempDir()
	t.Chdir(selected)
	ctx := effects.NewEffContext(nil)
	if err := prepareRunAIHandler(ctx, aiSetup{Model: "z-ai/glm-5.3-flash", KeyFile: "missing.key"}, nil, nil); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	_, err := ctx.AI.Call("hi")
	if err == nil || !strings.Contains(err.Error(), filepath.Join(selected, "missing.key")) {
		t.Fatalf("relative key changed source: %v", err)
	}
}

func TestPrepareRunAIHandler_ConfigDrivenAuthFrozen(t *testing.T) {
	for _, header := range []bool{false, true} {
		t.Run(map[bool]string{false: "bearer", true: "headers"}[header], func(t *testing.T) {
			t.Setenv("AILANG_AI_KEY_FILE", "")
			t.Setenv("LAZY_FIXTURE_CREDENTIAL", "selected-secret")
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				name, want := "Authorization", "Bearer selected-secret"
				if header {
					name, want = "X-Fixture-Auth", "prefix selected-secret"
				}
				if r.Header.Get(name) != want {
					t.Errorf("configured auth value changed: %s=%q", name, r.Header.Get(name))
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"text":"mock"}`))
			}))
			defer server.Close()
			previous := ai.GlobalProviderRegistry
			t.Cleanup(func() { ai.GlobalProviderRegistry = previous })
			ai.GlobalProviderRegistry = ai.NewProviderRegistry()
			spec := &pkg.AIProviderSpec{SchemaVersion: 1, Name: "auth-fixture", Endpoint: server.URL, RequestShape: "simple_completion", ResponsePath: "$.text", Auth: pkg.AIProviderAuth{Type: "bearer", Env: "LAZY_FIXTURE_CREDENTIAL"}}
			if header {
				spec.Auth = pkg.AIProviderAuth{}
				spec.AuthHeaders = map[string]string{"X-Fixture-Auth": "prefix ${LAZY_FIXTURE_CREDENTIAL}"}
			}
			if err := ai.GlobalProviderRegistry.Register(spec.Name, configdriven.New(spec), "test fixture"); err != nil {
				t.Fatal(err)
			}
			ctx := effects.NewEffContext(nil)
			if err := prepareRunAIHandler(ctx, aiSetup{Model: "auth-fixture/test"}, nil, nil); err != nil {
				t.Fatal(err)
			}
			t.Setenv("LAZY_FIXTURE_CREDENTIAL", "late-secret")
			if _, err := ctx.AI.Call("hi"); err != nil {
				t.Fatal(err)
			}
			if calls.Load() != 1 {
				t.Fatalf("calls=%d", calls.Load())
			}
		})
	}
}
