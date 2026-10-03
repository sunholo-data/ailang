package ai_test

// #1496: providers without reference-conditioned image generation must fail
// loudly — before any network call — rather than silently dropping the
// reference images. Gemini and OpenRouter support it (tested in their own
// packages); every other provider is pinned here.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/anthropic"
	"github.com/sunholo-data/ailang/internal/ai/configdriven"
	"github.com/sunholo-data/ailang/internal/ai/ollama"
	"github.com/sunholo-data/ailang/internal/ai/openai"
	"github.com/sunholo-data/ailang/internal/pkg"
)

func TestImageRefs_UnsupportedProvidersFailLoudly(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ol, err := ollama.NewClient(ollama.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("ollama client: %v", err)
	}
	visionSpec := &pkg.AIProviderSpec{
		SchemaVersion: 1,
		Name:          "test-vision",
		Endpoint:      server.URL,
		RequestShape:  "openai_chat",
		ResponsePath:  "$.choices[0].message.content",
		Capabilities:  pkg.AIProviderCapabilities{Vision: true},
	}
	providers := []ai.Provider{
		anthropic.NewClient("k", anthropic.WithBaseURL(server.URL)),
		openai.NewClient("k", openai.WithBaseURL(server.URL)),
		ol,
		configdriven.New(visionSpec),
	}
	for _, p := range providers {
		t.Run(p.Name(), func(t *testing.T) {
			_, err := p.Generate(context.Background(), &ai.Request{
				Model:              "some-model",
				UserPrompt:         "same person, older",
				ResponseModalities: []string{"IMAGE"},
				InputImages:        []ai.ImagePart{{Source: "cmVm", Mime: "image/png"}},
			})
			if err == nil {
				t.Fatal("expected a not-supported error, got nil")
			}
			if !strings.Contains(err.Error(), "not supported") {
				t.Errorf("err = %q, want a 'not supported' message", err.Error())
			}
		})
	}
	if calls != 0 {
		t.Errorf("server called %d times; unsupported providers must refuse before dispatch", calls)
	}
}
