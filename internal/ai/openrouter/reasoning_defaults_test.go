package openrouter

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// sentReasoning returns the reasoning block of the captured request body.
func sentReasoning(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var v struct {
		Reasoning map[string]any `json:"reasoning"`
	}
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	return v.Reasoning
}

// The registry's effort reaches a request that sets none — on Step (motoko's
// path), Generate and StreamStep alike — and a caller's own choice still wins.
func TestReasoningDefault_AppliesWhenCallerSetsNone(t *testing.T) {
	SetReasoningDefaults(map[string]string{"z-ai/glm-5.3-flash": "high"})
	t.Cleanup(func() { SetReasoningDefaults(nil) })

	cases := []struct {
		name  string
		model string
		opts  map[string]any
		want  any // effort on the wire; nil = no reasoning block
	}{
		{"registry default", "z-ai/glm-5.3-flash", nil, "high"},
		{"openrouter/ prefix", "openrouter/z-ai/glm-5.3-flash", nil, "high"},
		// The caller's own control replaces the default rather than merging:
		// reasoning_max_tokens alone goes out as reasoning.max_tokens, no effort.
		{"caller wins", "z-ai/glm-5.3-flash", map[string]any{"reasoning_max_tokens": 2000}, "max_tokens"},
		{"other model untouched", "deepseek/deepseek-v4.1-flash", nil, nil},
	}
	calls := map[string]func(*Client, *ai.Request) error{
		"Step": func(c *Client, r *ai.Request) error { _, err := c.Step(context.Background(), r); return err },
		"Generate": func(c *Client, r *ai.Request) error {
			_, err := c.Generate(context.Background(), r)
			return err
		},
		"StreamStep": func(c *Client, r *ai.Request) error {
			_, err := c.StreamStep(context.Background(), r, func(ai.StreamChunk) {})
			return err
		},
	}
	for path, call := range calls {
		for _, tc := range cases {
			t.Run(path+"/"+tc.name, func(t *testing.T) {
				cap := &captureHandler{}
				server := httptest.NewServer(cap)
				defer server.Close()
				client := NewClient("k", WithBaseURL(server.URL))
				err := call(client, &ai.Request{
					Model:    tc.model,
					Messages: []ai.Message{{Role: "user", Content: "hi"}},
					Options:  tc.opts,
				})
				if len(cap.captured) == 0 {
					t.Fatalf("no request reached the server (err: %v)", err)
				}
				got := sentReasoning(t, cap.captured)
				if tc.want == nil {
					if got != nil {
						t.Fatalf("reasoning = %v, want none", got)
					}
					return
				}
				if tc.want == "max_tokens" {
					if got == nil || got["effort"] != nil || got["max_tokens"] == nil {
						t.Fatalf("reasoning = %v, want the caller's max_tokens and no effort", got)
					}
					return
				}
				if got == nil || got["effort"] != tc.want {
					t.Fatalf("reasoning = %v, want effort %v", got, tc.want)
				}
			})
		}
	}
}

// A declared effort the model is not verified to honour must fail loudly,
// not ship: the default is subject to the same capability gate as a caller's.
func TestReasoningDefault_UnverifiedEffortFailsLoud(t *testing.T) {
	SetReasoningDefaults(map[string]string{"z-ai/glm-5.3-flash": "low"})
	t.Cleanup(func() { SetReasoningDefaults(nil) })
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()
	_, err := NewClient("k", WithBaseURL(server.URL)).Step(context.Background(), &ai.Request{
		Model:    "z-ai/glm-5.3-flash",
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})
	if err == nil || len(cap.captured) != 0 {
		t.Fatalf("unverified default effort: err=%v, sent=%d bytes; want an error and nothing sent", err, len(cap.captured))
	}
}

// The registry's EffectiveOpenRouterProvider is what pi is told to send; it
// must equal what this client actually sends for the same model, or the two
// harnesses route differently again.
func TestEffectiveProviderMatchesClient(t *testing.T) {
	c, err := modelreg.LoadModelsConfig("../../modelreg/models.yml")
	if err != nil {
		t.Fatalf("load models.yml: %v", err)
	}
	blocks := c.OpenRouterProviderByAPIName()
	if len(blocks) == 0 {
		t.Fatal("no openrouter_provider blocks in models.yml — nothing compared")
	}
	SetProviderDefaults(blocks)
	t.Cleanup(func() { SetProviderDefaults(nil) })
	t.Setenv("AILANG_OPENROUTER_PROVIDER", "")
	for api := range blocks {
		got, err := routeFor(&ai.Request{Model: api})
		if err != nil {
			t.Fatalf("routeFor(%s): %v", api, err)
		}
		gotJSON, _ := json.Marshal(got)
		wantJSON, _ := json.Marshal(c.EffectiveOpenRouterProvider(api))
		if string(gotJSON) != string(wantJSON) {
			t.Errorf("%s: client sends %s, registry says %s", api, gotJSON, wantJSON)
		}
	}
}
