package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// These tests exercise the pure-Go conversion helpers in effects_helpers.go.
// They run on the host (no WASM env required) — the goal is to lock in the
// JS-compat shape contracts that the browser-side handlers depend on:
// empty inputs become empty arrays (NOT nil), tool_calls and cache
// breakpoints round-trip cleanly, and message field names match what
// OpenAI / Anthropic shims expect.

func TestMessagesToJSCompat_Empty(t *testing.T) {
	out := messagesToJSCompat(nil)
	if out == nil {
		t.Fatal("messagesToJSCompat(nil) returned nil; want empty slice")
	}
	if len(out) != 0 {
		t.Fatalf("messagesToJSCompat(nil) len=%d; want 0", len(out))
	}

	out = messagesToJSCompat([]ai.Message{})
	if out == nil {
		t.Fatal("messagesToJSCompat([]) returned nil; want empty slice")
	}
	if len(out) != 0 {
		t.Fatalf("messagesToJSCompat([]) len=%d; want 0", len(out))
	}
}

func TestMessagesToJSCompat_SingleMessage(t *testing.T) {
	in := []ai.Message{
		{Role: "user", Content: "hi", ToolCalls: nil, ToolCallID: ""},
	}
	out := messagesToJSCompat(in)
	if len(out) != 1 {
		t.Fatalf("len(out)=%d; want 1", len(out))
	}
	m := out[0].(map[string]interface{})
	if m["role"] != "user" {
		t.Errorf("role=%v; want user", m["role"])
	}
	if m["content"] != "hi" {
		t.Errorf("content=%v; want hi", m["content"])
	}
	if m["tool_call_id"] != "" {
		t.Errorf("tool_call_id=%v; want empty string", m["tool_call_id"])
	}
	tc := m["tool_calls"].([]interface{})
	if tc == nil {
		t.Fatal("tool_calls is nil; want empty slice (JS empty array, not null)")
	}
	if len(tc) != 0 {
		t.Errorf("len(tool_calls)=%d; want 0", len(tc))
	}
}

func TestMessagesToJSCompat_WithToolCalls(t *testing.T) {
	in := []ai.Message{
		{
			Role:    "assistant",
			Content: "",
			ToolCalls: []ai.ToolCall{
				{ID: "call_1", Name: "search", Arguments: `{"q":"go"}`},
				{ID: "call_2", Name: "fetch", Arguments: `{"url":"x"}`},
			},
			ToolCallID: "",
		},
		{Role: "tool", Content: "ok", ToolCallID: "call_1"},
	}
	out := messagesToJSCompat(in)
	if len(out) != 2 {
		t.Fatalf("len(out)=%d; want 2", len(out))
	}

	m0 := out[0].(map[string]interface{})
	tc := m0["tool_calls"].([]interface{})
	if len(tc) != 2 {
		t.Fatalf("len(tool_calls)=%d; want 2", len(tc))
	}
	tc0 := tc[0].(map[string]interface{})
	if tc0["id"] != "call_1" || tc0["name"] != "search" || tc0["arguments"] != `{"q":"go"}` {
		t.Errorf("tool_call[0] mismatch: %+v", tc0)
	}
	tc1 := tc[1].(map[string]interface{})
	if tc1["id"] != "call_2" || tc1["name"] != "fetch" {
		t.Errorf("tool_call[1] mismatch: %+v", tc1)
	}

	m1 := out[1].(map[string]interface{})
	if m1["role"] != "tool" || m1["tool_call_id"] != "call_1" {
		t.Errorf("tool message mismatch: %+v", m1)
	}
}

func TestToolsToJSCompat_Empty(t *testing.T) {
	out := toolsToJSCompat(nil)
	if out == nil || len(out) != 0 {
		t.Fatalf("nil → out=%v len=%d; want non-nil empty slice", out, len(out))
	}
}

func TestToolsToJSCompat_TwoTools(t *testing.T) {
	in := []ai.ToolSchema{
		{Name: "search", Description: "Web search", Parameters: `{"type":"object"}`},
		{Name: "fetch", Description: "URL fetch", Parameters: `{"type":"object","required":["url"]}`},
	}
	out := toolsToJSCompat(in)
	if len(out) != 2 {
		t.Fatalf("len(out)=%d; want 2", len(out))
	}
	t0 := out[0].(map[string]interface{})
	if t0["name"] != "search" || t0["description"] != "Web search" {
		t.Errorf("tool[0] mismatch: %+v", t0)
	}
	if t0["parameters"] != `{"type":"object"}` {
		t.Errorf("tool[0] parameters not raw JSON string: %v", t0["parameters"])
	}
	t1 := out[1].(map[string]interface{})
	if t1["name"] != "fetch" {
		t.Errorf("tool[1] name=%v; want fetch", t1["name"])
	}
}

func TestCacheBreakpointsToJSCompat_Empty(t *testing.T) {
	// Acceptance criterion from M2: empty cache_breakpoints serializes to JS
	// empty array, not null. This is the contract the JS shim relies on
	// when forwarding to providers that don't support caching — they need
	// an iterable, not a null check.
	out := cacheBreakpointsToJSCompat(nil)
	if out == nil {
		t.Fatal("nil input returned nil; want empty slice (JS empty array)")
	}
	if len(out) != 0 {
		t.Fatalf("nil input len=%d; want 0", len(out))
	}

	out = cacheBreakpointsToJSCompat([]ai.CacheBreakpoint{})
	if out == nil || len(out) != 0 {
		t.Fatalf("[] input → out=%v len=%d; want non-nil empty slice", out, len(out))
	}
}

func TestCacheBreakpointsToJSCompat_SystemEphemeral(t *testing.T) {
	in := []ai.CacheBreakpoint{
		{Position: "system", TTL: "ephemeral"},
	}
	out := cacheBreakpointsToJSCompat(in)
	if len(out) != 1 {
		t.Fatalf("len(out)=%d; want 1", len(out))
	}
	bp := out[0].(map[string]interface{})
	if bp["position"] != "system" {
		t.Errorf("position=%v; want system", bp["position"])
	}
	if bp["ttl"] != "ephemeral" {
		t.Errorf("ttl=%v; want ephemeral", bp["ttl"])
	}
}

func TestCacheBreakpointsToJSCompat_MultipleBreakpoints(t *testing.T) {
	in := []ai.CacheBreakpoint{
		{Position: "system", TTL: "ephemeral"},
		{Position: "tools", TTL: "1h"},
	}
	out := cacheBreakpointsToJSCompat(in)
	if len(out) != 2 {
		t.Fatalf("len(out)=%d; want 2", len(out))
	}
	bp1 := out[1].(map[string]interface{})
	if bp1["position"] != "tools" || bp1["ttl"] != "1h" {
		t.Errorf("breakpoint[1] mismatch: %+v", bp1)
	}
}
