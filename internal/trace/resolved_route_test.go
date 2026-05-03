package trace

import (
	"encoding/json"
	"testing"
)

// TestEffectEvent_BackwardCompat verifies that a trace event serialized
// before ResolvedRoute existed (i.e. without a "route" field) still
// unmarshals correctly into the new EffectEvent shape — Route ends up nil
// and all other fields populate as before.
func TestEffectEvent_BackwardCompat(t *testing.T) {
	// Pre-M3 trace event: no "route" field at all.
	oldJSON := `{
		"version": "1.0",
		"event": "effect",
		"timestamp_ns": 12345,
		"effect": {
			"effect_name": "AI",
			"op_name": "call",
			"args": ["hello"],
			"result": "world"
		}
	}`

	var evt TraceEvent
	if err := json.Unmarshal([]byte(oldJSON), &evt); err != nil {
		t.Fatalf("unmarshal old trace event: %v", err)
	}
	if evt.Effect == nil {
		t.Fatal("Effect is nil")
	}
	if evt.Effect.EffectName != "AI" {
		t.Errorf("EffectName = %q, want AI", evt.Effect.EffectName)
	}
	if evt.Effect.OpName != "call" {
		t.Errorf("OpName = %q, want call", evt.Effect.OpName)
	}
	if evt.Effect.Route != nil {
		t.Errorf("Route = %+v, want nil for old-format events", evt.Effect.Route)
	}
}

// TestEffectEvent_RouteRoundTrip verifies that an EffectEvent with a
// populated ResolvedRoute round-trips through JSON unchanged.
func TestEffectEvent_RouteRoundTrip(t *testing.T) {
	original := EffectEvent{
		EffectName: "AI",
		OpName:     "call",
		Args:       []string{"summarize"},
		Result:     "ok",
		Route: &ResolvedRoute{
			RequestedModel:   "anthropic/claude-sonnet-4.5",
			ResolvedModel:    "anthropic/claude-sonnet-4.5",
			ResolvedProvider: "Anthropic",
			FallbackChain:    []string{"anthropic/claude-sonnet-4.5"},
			PromptTokens:     100,
			CompletionTokens: 25,
			CachedTokens:     60,
			ReasoningTokens:  0,
			CostUSD:          "0.000345",
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got EffectEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Route == nil {
		t.Fatal("Route is nil after round-trip")
	}
	if got.Route.RequestedModel != original.Route.RequestedModel {
		t.Errorf("RequestedModel = %q", got.Route.RequestedModel)
	}
	if got.Route.ResolvedProvider != "Anthropic" {
		t.Errorf("ResolvedProvider = %q", got.Route.ResolvedProvider)
	}
	if got.Route.CachedTokens != 60 {
		t.Errorf("CachedTokens = %d", got.Route.CachedTokens)
	}
	if got.Route.CostUSD != "0.000345" {
		t.Errorf("CostUSD = %q", got.Route.CostUSD)
	}
}

// TestEffectEvent_NilRouteOmitted verifies that an EffectEvent with a nil
// Route serializes without a "route" field (so old consumers don't see a
// new key). This is what makes the change additive on the wire.
func TestEffectEvent_NilRouteOmitted(t *testing.T) {
	evt := EffectEvent{
		EffectName: "IO",
		OpName:     "println",
		Result:     "hi",
	}
	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(data); contains(got, `"route"`) {
		t.Errorf("nil Route should be omitted, got: %s", got)
	}
}

// TestCollector_RecordAIEffect verifies the new collector method emits an
// EffectEvent with EffectName="AI" and the supplied route attached.
func TestCollector_RecordAIEffect(t *testing.T) {
	c := NewCollector()
	route := &ResolvedRoute{
		RequestedModel: "anthropic/claude-sonnet-4.5",
		ResolvedModel:  "anthropic/claude-sonnet-4.5",
		PromptTokens:   100,
		CostUSD:        "0.0001",
	}
	c.RecordAIEffect("call", []string{"hello"}, "world", route)

	events := c.Events()
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	got := events[0]
	if got.Event != EventEffect {
		t.Errorf("Event = %v, want %v", got.Event, EventEffect)
	}
	if got.Effect == nil {
		t.Fatal("Effect is nil")
	}
	if got.Effect.EffectName != "AI" {
		t.Errorf("EffectName = %q, want AI", got.Effect.EffectName)
	}
	if got.Effect.OpName != "call" {
		t.Errorf("OpName = %q, want call", got.Effect.OpName)
	}
	if got.Effect.Route == nil {
		t.Fatal("Route is nil — should be populated")
	}
	if got.Effect.Route.RequestedModel != "anthropic/claude-sonnet-4.5" {
		t.Errorf("RequestedModel = %q", got.Effect.Route.RequestedModel)
	}
	if got.Effect.Route.CostUSD != "0.0001" {
		t.Errorf("CostUSD = %q", got.Effect.Route.CostUSD)
	}
}

// TestCollector_RecordAIEffect_NilRoute verifies that recording with a nil
// route produces an event whose Effect.Route is nil (no fabrication).
func TestCollector_RecordAIEffect_NilRoute(t *testing.T) {
	c := NewCollector()
	c.RecordAIEffect("call", []string{"hi"}, "ok", nil)

	events := c.Events()
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if events[0].Effect.Route != nil {
		t.Errorf("Route = %+v, want nil for non-routed call", events[0].Effect.Route)
	}
}

// contains is a tiny helper to avoid pulling in strings just for one call.
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
