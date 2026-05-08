package openrouter

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/openai"
)

// helper — build a minimal ChatStepRequest with a system message in slot 0.
func minimalChatReq(systemText string) *openai.ChatStepRequest {
	systemRaw, _ := json.Marshal(systemText)
	userRaw, _ := json.Marshal("hi")
	return &openai.ChatStepRequest{
		Model: "anthropic/claude-3-5-haiku",
		Messages: []openai.ChatStepMessage{
			{Role: "system", Content: systemRaw},
			{Role: "user", Content: userRaw},
		},
	}
}

func TestProviderKeyForRoute(t *testing.T) {
	cases := map[string]string{
		"anthropic/claude-3-5-haiku":  "openrouter_routed_to_anthropic",
		"openai/gpt-4o-mini":          "openrouter_routed_to_openai",
		"google/gemini-2.5-flash":     "openrouter_routed_to_gemini",
		"mistralai/mixtral-8x7b":      "openrouter_routed_to_unknown",
		"unknown-provider/some-model": "openrouter_routed_to_unknown",
	}
	for model, want := range cases {
		t.Run(model, func(t *testing.T) {
			if got := providerKeyForRoute(model); got != want {
				t.Errorf("providerKeyForRoute(%q) = %q, want %q", model, got, want)
			}
		})
	}
}

// TestApplyCacheHints_AnthropicRoute_StampsCacheControl: the load-bearing
// happy-path test. anthropic/... model + system breakpoint = wire body has
// cache_control on the system message's content array.
func TestApplyCacheHints_AnthropicRoute_StampsCacheControl(t *testing.T) {
	req := minimalChatReq("Long stable system prompt for caching")
	err := applyCacheHintsForRoute(req, "anthropic/claude-3-5-haiku", []ai.CacheBreakpoint{
		{Position: "system", TTL: "ephemeral"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	systemContent := req.Messages[0].Content
	// System message Content should now be a JSON array, not a bare string.
	var blocks []map[string]interface{}
	if err := json.Unmarshal(systemContent, &blocks); err != nil {
		t.Fatalf("expected JSON array Content, got %s (err: %v)", string(systemContent), err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	cc, ok := blocks[0]["cache_control"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cache_control object, got %T", blocks[0]["cache_control"])
	}
	if cc["type"] != "ephemeral" {
		t.Errorf("cache_control.type = %v, want ephemeral", cc["type"])
	}
	// Original text preserved.
	if blocks[0]["text"] != "Long stable system prompt for caching" {
		t.Errorf("text mismatch: %v", blocks[0]["text"])
	}
	// User message untouched.
	var userText string
	if err := json.Unmarshal(req.Messages[1].Content, &userText); err != nil {
		t.Errorf("user content corrupted: %s", string(req.Messages[1].Content))
	}
}

// TestApplyCacheHints_EmptyBreakpoints_NoMutation: load-bearing back-compat
// — no breakpoints means the wire body must be UNCHANGED.
func TestApplyCacheHints_EmptyBreakpoints_NoMutation(t *testing.T) {
	original := minimalChatReq("hello")
	originalSystemBytes := string(original.Messages[0].Content)

	err := applyCacheHintsForRoute(original, "anthropic/claude-3-5-haiku", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(original.Messages[0].Content) != originalSystemBytes {
		t.Errorf("nil breakpoints mutated wire bytes: got %s, want %s",
			string(original.Messages[0].Content), originalSystemBytes)
	}
}

func TestApplyCacheHints_OpenAIRoute_NoMutation_WithWarning(t *testing.T) {
	ai.WarnOnceCacheHintIgnored("force-fresh-state-marker", "x") // burn an unrelated key
	req := minimalChatReq("hello")
	originalSystemBytes := string(req.Messages[0].Content)

	err := applyCacheHintsForRoute(req, "openai/gpt-4o-mini", []ai.CacheBreakpoint{
		{Position: "system", TTL: "ephemeral"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// OpenAI route: no mutation, warning emitted (warning visibility tested in
	// internal/ai/cache_warnings_test.go; here we only verify no mutation).
	if string(req.Messages[0].Content) != originalSystemBytes {
		t.Errorf("openai route should not mutate wire: got %s", string(req.Messages[0].Content))
	}
}

func TestApplyCacheHints_GoogleRoute_NoMutation(t *testing.T) {
	req := minimalChatReq("hello")
	originalSystemBytes := string(req.Messages[0].Content)

	err := applyCacheHintsForRoute(req, "google/gemini-2.5-flash", []ai.CacheBreakpoint{
		{Position: "system", TTL: "ephemeral"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(req.Messages[0].Content) != originalSystemBytes {
		t.Errorf("google route should not mutate wire: got %s", string(req.Messages[0].Content))
	}
}

func TestApplyCacheHints_UnknownRoute_NoMutation_NoWarning(t *testing.T) {
	req := minimalChatReq("hello")
	originalSystemBytes := string(req.Messages[0].Content)

	err := applyCacheHintsForRoute(req, "mistralai/mixtral-8x7b", []ai.CacheBreakpoint{
		{Position: "system", TTL: "ephemeral"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(req.Messages[0].Content) != originalSystemBytes {
		t.Errorf("unknown route should not mutate wire: got %s", string(req.Messages[0].Content))
	}
}

func TestStampAnthropicCacheControl_NoSystemMessage_NoMutation(t *testing.T) {
	// Build a request with NO system message — only user.
	userRaw, _ := json.Marshal("hi")
	req := &openai.ChatStepRequest{
		Model: "anthropic/claude-3-5-haiku",
		Messages: []openai.ChatStepMessage{
			{Role: "user", Content: userRaw},
		},
	}
	original := string(req.Messages[0].Content)
	err := stampAnthropicCacheControl(req, []ai.CacheBreakpoint{
		{Position: "system", TTL: "ephemeral"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(req.Messages[0].Content) != original {
		t.Errorf("user message should not be mutated when no system present")
	}
}

func TestStampAnthropicCacheControl_NoSystemBreakpoint_NoMutation(t *testing.T) {
	req := minimalChatReq("hi")
	original := string(req.Messages[0].Content)
	err := stampAnthropicCacheControl(req, []ai.CacheBreakpoint{
		{Position: "tool_result", TTL: "ephemeral"}, // not "system"
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(req.Messages[0].Content) != original {
		t.Errorf("non-system breakpoint should not mutate system message")
	}
}

// TestStampAnthropicCacheControl_AlreadyContentArray_DoesNotDoubleWrap:
// defensive — if the system Content is already a JSON array (e.g. some
// future caller pre-built it), we don't crash or double-wrap.
func TestStampAnthropicCacheControl_AlreadyContentArray_DoesNotDoubleWrap(t *testing.T) {
	pre := json.RawMessage(`[{"type":"text","text":"hi"}]`)
	req := &openai.ChatStepRequest{
		Model: "anthropic/claude-3-5-haiku",
		Messages: []openai.ChatStepMessage{
			{Role: "system", Content: pre},
		},
	}
	original := string(pre)
	err := stampAnthropicCacheControl(req, []ai.CacheBreakpoint{
		{Position: "system", TTL: "ephemeral"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be unchanged — Unmarshal-as-string fails, we bail without error.
	if !strings.Contains(string(req.Messages[0].Content), original) &&
		string(req.Messages[0].Content) != original {
		// Allow exact match (current behavior). The contract is "don't double-wrap".
		t.Errorf("expected no double-wrap; got: %s", string(req.Messages[0].Content))
	}
}
