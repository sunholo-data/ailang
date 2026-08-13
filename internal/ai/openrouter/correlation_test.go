package openrouter

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// M-OPENROUTER-BROADCAST-INGEST M3.
//
// The headline property is NEGATIVE: a request that sets no correlation must
// marshal byte-identically to one built before these fields existed. Every
// OpenRouter call in the project — evals, agent mode, the `ai` effect — flows
// through these three build sites, so a wire-shape change here is a change to
// everything.

func TestCorrelation_NilIsWireIdentical_ChatRequest(t *testing.T) {
	base := chatRequest{
		Model:     "anthropic/claude-sonnet-4.5",
		Messages:  []chatMessage{{Role: "user", Content: "hi"}},
		MaxTokens: 4096,
	}

	before, err := json.Marshal(base)
	if err != nil {
		t.Fatalf("marshal baseline: %v", err)
	}

	withCorrelation := base
	if err := applyCorrelation(&withCorrelation, nil); err != nil {
		t.Fatalf("applyCorrelation(nil): %v", err)
	}
	after, err := json.Marshal(withCorrelation)
	if err != nil {
		t.Fatalf("marshal after: %v", err)
	}

	if string(before) != string(after) {
		t.Errorf("nil correlation changed the wire bytes:\n  before = %s\n  after  = %s", before, after)
	}
	if strings.Contains(string(after), "session_id") || strings.Contains(string(after), `"trace"`) {
		t.Errorf("nil correlation emitted correlation keys: %s", after)
	}
}

// TestCorrelation_EmptyStructIsWireIdentical covers the non-nil-but-empty case,
// which is what a caller that constructs a Correlation and fills nothing in
// produces.
func TestCorrelation_EmptyStructIsWireIdentical(t *testing.T) {
	base := chatRequest{Model: "m", Messages: []chatMessage{{Role: "user", Content: "hi"}}}
	before, _ := json.Marshal(base)

	withEmpty := base
	if err := applyCorrelation(&withEmpty, &ai.Correlation{}); err != nil {
		t.Fatalf("applyCorrelation(empty): %v", err)
	}
	after, _ := json.Marshal(withEmpty)

	if string(before) != string(after) {
		t.Errorf("empty correlation changed the wire bytes:\n  before = %s\n  after  = %s", before, after)
	}
}

// TestCorrelationExtras_EmptyProducesNoFragments is the same guarantee for the
// splice-based sites (step.go, streamstep.go). No fragments means
// marshalStepBodyWithExtras returns the OpenAI body untouched.
func TestCorrelationExtras_EmptyProducesNoFragments(t *testing.T) {
	for _, tt := range []struct {
		name string
		c    *ai.Correlation
	}{
		{"nil", nil},
		{"empty struct", &ai.Correlation{}},
		{"empty trace map", &ai.Correlation{Trace: map[string]any{}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			extras, err := correlationExtras(tt.c)
			if err != nil {
				t.Fatalf("correlationExtras: %v", err)
			}
			if len(extras) != 0 {
				t.Errorf("got %d fragments, want 0: %q", len(extras), extras)
			}
		})
	}
}

// TestCorrelation_PopulatedFieldsReachTheWire verifies the positive case by
// unmarshalling the ACTUAL bytes rather than inspecting the struct.
func TestCorrelation_PopulatedFieldsReachTheWire(t *testing.T) {
	req := chatRequest{Model: "m", Messages: []chatMessage{{Role: "user", Content: "hi"}}}
	err := applyCorrelation(&req, &ai.Correlation{
		SessionID: "b1df1f0e-3cfe-4783-8712-9c5f73fe5a50",
		Trace:     map[string]any{"trace_name": "eval:fizzbuzz", "benchmark": "fizzbuzz"},
	})
	if err != nil {
		t.Fatalf("applyCorrelation: %v", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got := decoded["session_id"]; got != "b1df1f0e-3cfe-4783-8712-9c5f73fe5a50" {
		t.Errorf("session_id = %v", got)
	}
	trace, ok := decoded["trace"].(map[string]any)
	if !ok {
		t.Fatalf("trace is %T, want an object", decoded["trace"])
	}
	if trace["trace_name"] != "eval:fizzbuzz" {
		t.Errorf("trace.trace_name = %v", trace["trace_name"])
	}
	// `user` was not set, so it must be absent rather than empty.
	if _, present := decoded["user"]; present {
		t.Errorf("user should be omitted when unset, got %v", decoded["user"])
	}
}

// TestCorrelationExtras_ProducesSplicableFragments checks the fragments are
// valid object members, since they are spliced in as raw bytes.
func TestCorrelationExtras_ProducesSplicableFragments(t *testing.T) {
	extras, err := correlationExtras(&ai.Correlation{
		User:      "user_12345",
		SessionID: "session_abc",
		Trace:     map[string]any{"trace_name": "t"},
	})
	if err != nil {
		t.Fatalf("correlationExtras: %v", err)
	}
	if len(extras) != 3 {
		t.Fatalf("got %d fragments, want 3", len(extras))
	}

	// Splice them into an object exactly as marshalStepBodyWithExtras does and
	// confirm the result parses.
	spliced := `{"model":"m",` + string(joinFragments(extras)) + `}`
	var decoded map[string]any
	if err := json.Unmarshal([]byte(spliced), &decoded); err != nil {
		t.Fatalf("spliced body is not valid JSON: %v\n%s", err, spliced)
	}
	if decoded["user"] != "user_12345" || decoded["session_id"] != "session_abc" {
		t.Errorf("spliced fields wrong: %v", decoded)
	}
}

func joinFragments(frags [][]byte) []byte {
	var out []byte
	for i, f := range frags {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, f...)
	}
	return out
}

// TestCorrelation_OverCapRejected pins the no-truncation rule: an over-long id
// must fail before dispatch, never be silently shortened into an id that joins
// to nothing.
func TestCorrelation_OverCapRejected(t *testing.T) {
	tests := []struct {
		name string
		c    *ai.Correlation
	}{
		{"session_id over 256", &ai.Correlation{SessionID: strings.Repeat("s", ai.MaxSessionIDLen+1)}},
		{"user over 128", &ai.Correlation{User: strings.Repeat("u", ai.MaxUserLen+1)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := chatRequest{Model: "m"}
			if err := applyCorrelation(&req, tt.c); err == nil {
				t.Error("applyCorrelation accepted an over-cap value, want a typed error")
			}
			if _, err := correlationExtras(tt.c); err == nil {
				t.Error("correlationExtras accepted an over-cap value, want a typed error")
			}
			// And nothing was written on the way out.
			if req.SessionID != "" || req.User != "" {
				t.Errorf("rejected correlation still mutated the request: %+v", req)
			}
		})
	}
}

// TestCorrelation_AtCapAccepted is the boundary control, so the cap test above
// cannot pass by rejecting everything.
func TestCorrelation_AtCapAccepted(t *testing.T) {
	req := chatRequest{Model: "m"}
	err := applyCorrelation(&req, &ai.Correlation{
		User:      strings.Repeat("u", ai.MaxUserLen),
		SessionID: strings.Repeat("s", ai.MaxSessionIDLen),
	})
	if err != nil {
		t.Errorf("at-cap values must be accepted, got: %v", err)
	}
}
