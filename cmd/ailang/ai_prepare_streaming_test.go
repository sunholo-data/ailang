package main

import (
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/configdriven"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

func TestPrepareRunAIHandler_NamedStreamingFrozen(t *testing.T) {
	for _, mode := range []string{"none", "stub", "lazy"} {
		for _, op := range []string{"streamCall", "callStream"} {
			for _, mutation := range []string{"auth", "registry"} {
				t.Run(mode+"/"+op+"/"+mutation, func(t *testing.T) {
					old := ai.GlobalProviderRegistry
					t.Cleanup(func() { ai.GlobalProviderRegistry = old })
					ai.GlobalProviderRegistry = ai.NewProviderRegistry()
					t.Setenv("FROZEN_STREAM_KEY", "prepared-secret")
					t.Setenv("AILANG_AI_KEY_FILE", "")
					var requests atomic.Int32
					server := newOpenAISSEServer(t, func(r *http.Request, _ []byte) {
						requests.Add(1)
						if got := r.Header.Get("Authorization"); got != "Bearer prepared-secret" {
							t.Errorf("late authentication lookup: %q", got)
						}
						// callStream must use the same captured delta parser after opening.
						if mutation == "registry" {
							ai.GlobalProviderRegistry.Reset()
						}
					})
					defer server.Close()
					registerOpenAITestProvider(t, "named-frozen", server.URL, "FROZEN_STREAM_KEY")
					provider, _ := ai.GlobalProviderRegistry.Lookup("named-frozen")
					spec := provider.(*configdriven.Provider).Spec()
					ctx := streamTestContext(t)
					defer ctx.Stream.CloseAll()
					cfg := aiSetup{}
					if mode == "stub" {
						cfg.Stub = true
					} else if mode == "lazy" {
						// Named streaming must not initialize this unrelated provider.
						cfg.Model = "z-ai/glm-5.3-flash"
						cfg.KeyFile = filepath.Join(t.TempDir(), "missing.key")
					}
					if err := prepareRunAIHandler(ctx, cfg, nil, nil); err != nil {
						t.Fatal(err)
					}
					if mode == "none" && ctx.AI != nil {
						t.Fatal("named-provider snapshot installed a selected handler")
					}
					// Function budget views must preserve the same run's lookup.
					ctx = ctx.WithBudget(nil)
					t.Setenv("FROZEN_STREAM_KEY", "late-secret")
					if mutation == "registry" {
						spec.Endpoint = "http://127.0.0.1:1"
						spec.Streaming.DeltaPath = "$.changed"
						ai.GlobalProviderRegistry.Reset()
					}
					result, err := effects.Call(ctx, "AI", op, []eval.Value{
						&eval.StringValue{Value: "named-frozen"},
						&eval.StringValue{Value: "fixture-model"},
						&eval.StringValue{Value: `[{"role":"user","content":"hi"}]`},
					})
					if err != nil || result.(*eval.TaggedValue).CtorName != "Ok" {
						t.Fatalf("frozen named provider: %v %v", result, err)
					}
					if op == "callStream" && !strings.Contains(result.String(), "Hello world") {
						t.Fatalf("delta parser snapshot lost: %v", result)
					}
					if requests.Load() != 1 {
						t.Fatalf("local requests = %d", requests.Load())
					}
				})
			}
		}
	}
}

func TestNamedStreaming_AuthorityBeforeProviderLookup(t *testing.T) {
	for _, op := range []string{"streamCall", "callStream"} {
		for _, denial := range []string{"capability", "budget"} {
			t.Run(op+"/"+denial, func(t *testing.T) {
				ctx := effects.NewEffContext(nil)
				lookups := 0
				ctx.AI = effects.NewLazyAIContext(func() (*effects.AIContext, error) {
					t.Fatal("named streaming initialized selected provider")
					return nil, nil
				})
				ctx.WithProviderLookup(func(string) (ai.Provider, bool) {
					lookups++
					return nil, false
				})
				if denial == "budget" {
					ctx.Grant(effects.NewCapability("AI"))
					zero := 0
					ctx.Budget = effects.NewBudgetContext(map[string]*int{"AI": &zero})
				}
				_, err := effects.Call(ctx, "AI", op, []eval.Value{
					&eval.StringValue{Value: "fixture"}, &eval.StringValue{Value: "model"}, &eval.StringValue{Value: "[]"},
				})
				if err == nil || lookups != 0 {
					t.Fatalf("authorization boundary: err=%v lookups=%d", err, lookups)
				}
			})
		}
	}
}

func TestPrepareRunAIHandler_NamedMissingAuthStaysMissing(t *testing.T) {
	old := ai.GlobalProviderRegistry
	t.Cleanup(func() { ai.GlobalProviderRegistry = old })
	ai.GlobalProviderRegistry = ai.NewProviderRegistry()
	t.Setenv("FROZEN_MISSING_STREAM_KEY", "")
	var requests atomic.Int32
	server := newOpenAISSEServer(t, func(*http.Request, []byte) { requests.Add(1) })
	defer server.Close()
	registerOpenAITestProvider(t, "missing-auth", server.URL, "FROZEN_MISSING_STREAM_KEY")
	ctx := streamTestContext(t)
	defer ctx.Stream.CloseAll()
	if err := prepareRunAIHandler(ctx, aiSetup{}, nil, nil); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FROZEN_MISSING_STREAM_KEY", "late-secret")
	for _, op := range []string{"streamCall", "callStream"} {
		result, err := effects.Call(ctx, "AI", op, []eval.Value{
			&eval.StringValue{Value: "missing-auth"}, &eval.StringValue{Value: "model"},
			&eval.StringValue{Value: `[{"role":"user","content":"hi"}]`},
		})
		if err != nil || result.(*eval.TaggedValue).CtorName != "Err" || !strings.Contains(result.String(), "FROZEN_MISSING_STREAM_KEY is unset") {
			t.Fatalf("missing auth mapping for %s: %v %v", op, result, err)
		}
	}
	if requests.Load() != 0 {
		t.Fatal("missing-at-preparation credential reached network")
	}
}
