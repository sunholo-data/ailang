package anthropic

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// TestSystemField_Backcompat: empty/nil breakpoints produce wire bytes
// IDENTICAL to today (a bare JSON string). This is the load-bearing
// back-compat assertion from the design doc Conflict Surface section.
func TestSystemField_Backcompat_BareString(t *testing.T) {
	cases := []struct {
		name        string
		breakpoints []ai.CacheBreakpoint
	}{
		{"nil_breakpoints", nil},
		{"empty_breakpoints", []ai.CacheBreakpoint{}},
		{"unrelated_breakpoint", []ai.CacheBreakpoint{{Position: "tool_result", TTL: "ephemeral"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := systemFieldFromPrompt("You are helpful.", tc.breakpoints)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Bare JSON-quoted string — exactly the pre-v0.18.4 shape.
			want := `"You are helpful."`
			if string(out) != want {
				t.Errorf("system wire bytes = %s, want %s (back-compat)", string(out), want)
			}
		})
	}
}

func TestSystemField_EmptyPrompt_ReturnsNil(t *testing.T) {
	// Empty system prompt + cache hint = nothing to cache, nil/omit field.
	out, err := systemFieldFromPrompt("", []ai.CacheBreakpoint{
		{Position: "system", TTL: "ephemeral"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != nil {
		t.Errorf("empty prompt should produce nil RawMessage, got %s", string(out))
	}
}

func TestSystemField_WithSystemBreakpoint_ContentArray(t *testing.T) {
	out, err := systemFieldFromPrompt("Cache me.", []ai.CacheBreakpoint{
		{Position: "system", TTL: "ephemeral"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Decode + structural-check (avoid byte-equality on Go's JSON map ordering).
	var blocks []map[string]interface{}
	if err := json.Unmarshal(out, &blocks); err != nil {
		t.Fatalf("expected JSON array, got %s (err: %v)", string(out), err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	b := blocks[0]
	if b["type"] != "text" {
		t.Errorf("block.type = %v, want text", b["type"])
	}
	if b["text"] != "Cache me." {
		t.Errorf("block.text = %v, want \"Cache me.\"", b["text"])
	}
	cc, ok := b["cache_control"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cache_control object, got %T", b["cache_control"])
	}
	if cc["type"] != "ephemeral" {
		t.Errorf("cache_control.type = %v, want ephemeral", cc["type"])
	}
}

func TestHasSystemBreakpoint(t *testing.T) {
	cases := []struct {
		name        string
		breakpoints []ai.CacheBreakpoint
		want        bool
	}{
		{"nil", nil, false},
		{"empty", []ai.CacheBreakpoint{}, false},
		{"only_system", []ai.CacheBreakpoint{{Position: "system"}}, true},
		{"only_tool_result", []ai.CacheBreakpoint{{Position: "tool_result"}}, false},
		{"system_among_others", []ai.CacheBreakpoint{
			{Position: "tool_result"},
			{Position: "system"},
			{Position: "last_user"},
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasSystemBreakpoint(tc.breakpoints); got != tc.want {
				t.Errorf("hasSystemBreakpoint(%+v) = %v, want %v", tc.breakpoints, got, tc.want)
			}
		})
	}
}

// TestBuildStepRequest_NoCache_BackcompatGoldenWire: end-to-end through
// buildStepRequest with no breakpoints. Verifies the full request body's
// `system` field serializes identically to the pre-v0.18.4 wire shape.
func TestBuildStepRequest_NoCache_BackcompatGoldenWire(t *testing.T) {
	req := &ai.Request{
		Model:        "claude-3-5-haiku",
		SystemPrompt: "You are helpful.",
		Messages: []ai.Message{
			{Role: "user", Content: "Hi"},
		},
		MaxTokens: 100,
	}
	apiReq, aiErr := buildStepRequest(req)
	if aiErr != nil {
		t.Fatalf("buildStepRequest error: %v", aiErr)
	}
	body, err := json.Marshal(apiReq)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Must contain a bare JSON string for system, NOT a content array.
	if !strings.Contains(string(body), `"system":"You are helpful."`) {
		t.Errorf("expected bare-string system field, got: %s", string(body))
	}
	// Must NOT contain cache_control.
	if strings.Contains(string(body), "cache_control") {
		t.Errorf("expected no cache_control with empty breakpoints, got: %s", string(body))
	}
}

// TestBuildStepRequest_SystemCacheHint_StampsCacheControl: end-to-end with
// a system breakpoint. Verifies the wire body contains cache_control.
func TestBuildStepRequest_SystemCacheHint_StampsCacheControl(t *testing.T) {
	req := &ai.Request{
		Model:        "claude-3-5-haiku",
		SystemPrompt: "Long stable system prompt...",
		Messages: []ai.Message{
			{Role: "user", Content: "Hi"},
		},
		MaxTokens: 100,
		CacheBreakpoints: []ai.CacheBreakpoint{
			{Position: "system", TTL: "ephemeral"},
		},
	}
	apiReq, aiErr := buildStepRequest(req)
	if aiErr != nil {
		t.Fatalf("buildStepRequest error: %v", aiErr)
	}
	body, err := json.Marshal(apiReq)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	bodyStr := string(body)
	// Must contain a content array (not a bare string) for system.
	if !strings.Contains(bodyStr, `"system":[{`) {
		t.Errorf("expected system content array, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `"cache_control":{"type":"ephemeral"}`) {
		t.Errorf("expected cache_control:{type:ephemeral}, got: %s", bodyStr)
	}
	// Original system text must still appear inside the array.
	if !strings.Contains(bodyStr, "Long stable system prompt") {
		t.Errorf("system text missing from wire body: %s", bodyStr)
	}
}

// TestBuildStepRequest_NoSystemPrompt_NoFieldEvenWithCacheHint: empty system
// + cache hint = field omitted (nothing to cache).
func TestBuildStepRequest_NoSystemPrompt_NoFieldEvenWithCacheHint(t *testing.T) {
	req := &ai.Request{
		Model:    "claude-3-5-haiku",
		Messages: []ai.Message{{Role: "user", Content: "Hi"}},
		CacheBreakpoints: []ai.CacheBreakpoint{
			{Position: "system", TTL: "ephemeral"},
		},
	}
	apiReq, aiErr := buildStepRequest(req)
	if aiErr != nil {
		t.Fatalf("buildStepRequest error: %v", aiErr)
	}
	body, err := json.Marshal(apiReq)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(body), `"system"`) {
		t.Errorf("expected system field omitted for empty prompt, got: %s", string(body))
	}
}
