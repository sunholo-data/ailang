package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

func TestTranslatePolicy_Nil(t *testing.T) {
	if got := translatePolicy(nil); got != nil {
		t.Errorf("translatePolicy(nil) = %+v, want nil", got)
	}
}

func TestTranslatePolicy_ZeroValue(t *testing.T) {
	if got := translatePolicy(&ai.AIRoutingPolicy{}); got != nil {
		t.Errorf("translatePolicy(zero) = %+v, want nil", got)
	}
}

func TestTranslatePolicy_OrderAndFallback(t *testing.T) {
	pol := &ai.AIRoutingPolicy{
		Order:         []string{"anthropic", "openai", "google"},
		AllowFallback: true,
	}
	got := translatePolicy(pol)
	if got == nil {
		t.Fatal("translatePolicy returned nil for non-zero policy")
	}
	if !reflect.DeepEqual(got.Order, []string{"anthropic", "openai", "google"}) {
		t.Errorf("Order = %v", got.Order)
	}
	if got.AllowFallbacks == nil || *got.AllowFallbacks != true {
		t.Errorf("AllowFallbacks = %v, want pointer to true", got.AllowFallbacks)
	}
}

// TestTranslatePolicy_AllowFallbackFalseWithOrder ensures the pointer trick
// causes an explicit false to ride along with an order list. Without the
// pointer we'd lose the explicit "no fallback" intent.
func TestTranslatePolicy_AllowFallbackFalseWithOrder(t *testing.T) {
	pol := &ai.AIRoutingPolicy{
		Order:         []string{"anthropic"},
		AllowFallback: false,
	}
	got := translatePolicy(pol)
	if got == nil {
		t.Fatal("translatePolicy returned nil")
	}
	if got.AllowFallbacks == nil {
		t.Fatalf("AllowFallbacks must be non-nil when Order is set")
	}
	if *got.AllowFallbacks != false {
		t.Errorf("AllowFallbacks = %v, want false", *got.AllowFallbacks)
	}

	// Round-trip through JSON to confirm `false` actually serialises.
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var roundTrip map[string]any
	if err := json.Unmarshal(b, &roundTrip); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if roundTrip["allow_fallbacks"] != false {
		t.Errorf("JSON allow_fallbacks = %v, want false", roundTrip["allow_fallbacks"])
	}
}

func TestTranslatePolicy_RequireParameters(t *testing.T) {
	pol := &ai.AIRoutingPolicy{
		Require: []ai.AICapability{
			ai.CapStructuredOutputs,
			ai.CapToolCalling,
			ai.CapJSONMode,
		},
	}
	got := translatePolicy(pol)
	if got == nil {
		t.Fatal("translatePolicy returned nil")
	}
	want := []string{"structured_outputs", "tool_calling", "json_mode"}
	if !reflect.DeepEqual(got.RequireParameters, want) {
		t.Errorf("RequireParameters = %v, want %v", got.RequireParameters, want)
	}
}

func TestTranslatePolicy_PreferMapping(t *testing.T) {
	cases := []struct {
		pref ai.RoutePreference
		want string
	}{
		{ai.PreferUnspecified, ""},
		{ai.PreferCheapest, "price"},
		{ai.PreferFastest, "throughput"},
		{ai.PreferMostReliable, "latency"},
	}
	for _, c := range cases {
		t.Run(string(c.pref)+"_to_"+c.want, func(t *testing.T) {
			pol := &ai.AIRoutingPolicy{
				// Add Order so the policy isn't zero-valued for PreferUnspecified case.
				Order:  []string{"anthropic"},
				Prefer: c.pref,
			}
			got := translatePolicy(pol)
			if got == nil {
				t.Fatal("nil result")
			}
			if got.Sort != c.want {
				t.Errorf("Sort = %q, want %q", got.Sort, c.want)
			}
		})
	}
}

// TestTranslatePolicy_MaxPriceIgnored documents the deferred-by-design
// behaviour: MaxPricePerMTok is silently dropped in M2. If this test starts
// failing it means someone wired the field through — update the test and
// the doc comment in routing.go together.
func TestTranslatePolicy_MaxPriceIgnored(t *testing.T) {
	pol := &ai.AIRoutingPolicy{
		Order:           []string{"anthropic"},
		MaxPricePerMTok: "0.005",
	}
	got := translatePolicy(pol)
	if got == nil {
		t.Fatal("nil result")
	}
	// Marshal the providerField — the JSON must NOT contain "max_price"
	// or any equivalent. translatePolicy is the contract here.
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for k := range m {
		if k == "max_price" || k == "max_price_per_mtok" {
			t.Errorf("unexpected field %q in serialised provider field; got: %s", k, string(b))
		}
	}
}

// TestGenerate_IncludesProviderField wires translatePolicy into the chat
// request and confirms the on-the-wire body contains the `provider` block.
func TestGenerate_IncludesProviderField(t *testing.T) {
	var capturedBody chatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "gen-1",
			"model": "x",
			"choices": [{"index":0, "message":{"role":"assistant","content":"ok"}, "finish_reason":"stop"}],
			"usage": {"prompt_tokens":1, "completion_tokens":1, "total_tokens":2}
		}`))
	}))
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:      "anthropic/claude-sonnet-4.5",
		UserPrompt: "hi",
		Routing: &ai.AIRoutingPolicy{
			Order:         []string{"anthropic", "openai"},
			AllowFallback: true,
			Require:       []ai.AICapability{ai.CapStructuredOutputs},
			Prefer:        ai.PreferCheapest,
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if capturedBody.Provider == nil {
		t.Fatal("captured request has no provider field; want non-nil")
	}
	if !reflect.DeepEqual(capturedBody.Provider.Order, []string{"anthropic", "openai"}) {
		t.Errorf("Order = %v", capturedBody.Provider.Order)
	}
	if capturedBody.Provider.AllowFallbacks == nil || !*capturedBody.Provider.AllowFallbacks {
		t.Errorf("AllowFallbacks = %v, want true", capturedBody.Provider.AllowFallbacks)
	}
	if !reflect.DeepEqual(capturedBody.Provider.RequireParameters, []string{"structured_outputs"}) {
		t.Errorf("RequireParameters = %v", capturedBody.Provider.RequireParameters)
	}
	if capturedBody.Provider.Sort != "price" {
		t.Errorf("Sort = %q, want price", capturedBody.Provider.Sort)
	}
}

// TestGenerate_NoProviderFieldWhenNoPolicy confirms the back-compat default:
// callers that don't set Routing get exactly the same request body as before.
func TestGenerate_NoProviderFieldWhenNoPolicy(t *testing.T) {
	var capturedBody chatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "gen-1",
			"model": "x",
			"choices": [{"index":0, "message":{"role":"assistant","content":"ok"}, "finish_reason":"stop"}],
			"usage": {"prompt_tokens":1, "completion_tokens":1, "total_tokens":2}
		}`))
	}))
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:      "anthropic/claude-sonnet-4.5",
		UserPrompt: "hi",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if capturedBody.Provider != nil {
		t.Errorf("Provider should be nil when no policy set, got %+v", capturedBody.Provider)
	}
}
