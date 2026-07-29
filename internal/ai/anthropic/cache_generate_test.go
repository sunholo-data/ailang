package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// M-ANTHROPIC-CACHE-HIT-RATE M1.
//
// These tests cover the gap recorded as V1 in the design doc: before this
// milestone, internal/ai/anthropic/client.go contained ZERO references to
// CacheBreakpoint, so the Generate path — which carries essentially all of our
// Anthropic traffic — could not request a prompt cache at all.
//
// The load-bearing assertion in this file is the back-compat one: a request
// that declares no breakpoints must produce wire bytes identical to v0.30.0.

// ---------------------------------------------------------------------------
// userContentFromPrompt — pure wire-shape builder
// ---------------------------------------------------------------------------

// TestUserContent_Backcompat_BareString: with no user_prefix breakpoint (or no
// CachedPrefix to cache), the user content field stays a bare JSON string —
// byte-identical to the pre-M1 shape.
func TestUserContent_Backcompat_BareString(t *testing.T) {
	cases := []struct {
		name         string
		cachedPrefix string
		userPrompt   string
		breakpoints  []ai.CacheBreakpoint
		want         string
	}{
		{
			name:        "nil_breakpoints",
			userPrompt:  "Hello",
			breakpoints: nil,
			want:        `"Hello"`,
		},
		{
			name:        "empty_breakpoints",
			userPrompt:  "Hello",
			breakpoints: []ai.CacheBreakpoint{},
			want:        `"Hello"`,
		},
		{
			name:        "unrelated_breakpoint_position",
			userPrompt:  "Hello",
			breakpoints: []ai.CacheBreakpoint{{Position: "system", TTL: "ephemeral"}},
			want:        `"Hello"`,
		},
		{
			// A prefix with no breakpoint must still reach the model — it is
			// concatenated, not dropped. This is the provider-neutral contract
			// (D2): CachedPrefix + UserPrompt is what the model sees.
			name:         "prefix_without_breakpoint_concatenates",
			cachedPrefix: "TEACHING",
			userPrompt:   "\n\n## Task\n\nDo it",
			breakpoints:  nil,
			want:         `"TEACHING\n\n## Task\n\nDo it"`,
		},
		{
			// Breakpoint declared but nothing to cache — degrade to the bare
			// string rather than emitting a one-block array.
			name:         "breakpoint_but_empty_prefix",
			cachedPrefix: "",
			userPrompt:   "Hello",
			breakpoints:  []ai.CacheBreakpoint{{Position: "user_prefix", TTL: "ephemeral"}},
			want:         `"Hello"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := userContentFromPrompt(tc.cachedPrefix, tc.userPrompt, tc.breakpoints)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(out) != tc.want {
				t.Errorf("user content wire bytes = %s, want %s (back-compat)", string(out), tc.want)
			}
		})
	}
}

// TestUserContent_TwoBlockArray: the caching shape. Block 0 carries the stable
// prefix plus cache_control; block 1 carries the volatile remainder and must
// NOT carry a marker (a marker there would key the cache to the varying task
// and never read).
func TestUserContent_TwoBlockArray(t *testing.T) {
	out, err := userContentFromPrompt(
		"TEACHING PROMPT",
		"\n\n## Task\n\nSolve it",
		[]ai.CacheBreakpoint{{Position: "user_prefix", TTL: "ephemeral"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var blocks []struct {
		Type         string `json:"type"`
		Text         string `json:"text"`
		CacheControl *struct {
			Type string `json:"type"`
		} `json:"cache_control"`
	}
	if err := json.Unmarshal(out, &blocks); err != nil {
		t.Fatalf("expected a content array, got %s (%v)", string(out), err)
	}
	if len(blocks) != 2 {
		t.Fatalf("got %d content blocks, want 2: %s", len(blocks), string(out))
	}
	if blocks[0].Text != "TEACHING PROMPT" {
		t.Errorf("block[0].text = %q, want the cached prefix", blocks[0].Text)
	}
	if blocks[0].CacheControl == nil || blocks[0].CacheControl.Type != "ephemeral" {
		t.Errorf("block[0] must carry cache_control{type:ephemeral}, got %+v", blocks[0].CacheControl)
	}
	if blocks[1].Text != "\n\n## Task\n\nSolve it" {
		t.Errorf("block[1].text = %q, want the volatile remainder", blocks[1].Text)
	}
	if blocks[1].CacheControl != nil {
		t.Error("block[1] must NOT carry cache_control — it varies per request, so a marker there never reads")
	}
	for i, b := range blocks {
		if b.Type != "text" {
			t.Errorf("block[%d].type = %q, want \"text\"", i, b.Type)
		}
	}
}

// TestUserContent_ConcatenationIsLossless: whatever the encoding, the text the
// model sees must equal cachedPrefix + userPrompt. This is what protects eval
// baseline comparability under D1 (cache in place, don't move to system role).
func TestUserContent_ConcatenationIsLossless(t *testing.T) {
	const prefix = "TEACHING PROMPT"
	const task = "\n\n## Task\n\nSolve it"

	cached, err := userContentFromPrompt(prefix, task,
		[]ai.CacheBreakpoint{{Position: "user_prefix", TTL: "ephemeral"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	plain, err := userContentFromPrompt(prefix, task, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Uncached form is a bare string.
	var plainText string
	if err := json.Unmarshal(plain, &plainText); err != nil {
		t.Fatalf("uncached form should be a JSON string: %v", err)
	}
	// Cached form is an array; concatenating its text must equal the same thing.
	var blocks []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(cached, &blocks); err != nil {
		t.Fatalf("cached form should be a JSON array: %v", err)
	}
	var sb strings.Builder
	for _, b := range blocks {
		sb.WriteString(b.Text)
	}
	if sb.String() != plainText {
		t.Errorf("cached encoding changes what the model sees:\n cached = %q\n plain  = %q", sb.String(), plainText)
	}
	if plainText != prefix+task {
		t.Errorf("plain text = %q, want %q", plainText, prefix+task)
	}
}

// ---------------------------------------------------------------------------
// Minimum cacheable prefix — the silent-no-op guard
// ---------------------------------------------------------------------------

// TestCachePrefixTooSmall: Anthropic silently declines to cache a prefix under
// the model's minimum (no error, cache_creation_input_tokens: 0). That silence
// is exactly how the eval-harness gap (design doc V8: a 70-token system prompt
// against a 1024-token minimum) stayed invisible. Detect it and warn.
func TestCachePrefixTooSmall(t *testing.T) {
	small := strings.Repeat("x", 400)     // ~100 tokens
	big := strings.Repeat("x", 64*1024)   // ~16k tokens
	medium := strings.Repeat("x", 8*1024) // ~2k tokens

	cases := []struct {
		name         string
		model        string
		prefix       string
		wantTooSmall bool
		wantMin      int
	}{
		{"opus5_small_prefix_too_small", "claude-opus-5", small, true, 512},
		{"opus5_big_prefix_ok", "claude-opus-5", big, false, 512},
		{"sonnet5_small_too_small", "claude-sonnet-5", small, true, 1024},
		{"sonnet5_big_ok", "claude-sonnet-5", big, false, 1024},
		{"haiku45_medium_too_small", "claude-haiku-4-5", medium, true, 4096},
		{"haiku45_big_ok", "claude-haiku-4-5", big, false, 4096},
		{"opus46_medium_too_small", "claude-opus-4-6", medium, true, 4096},
		{"unknown_model_defaults_conservative", "some-future-model", small, true, 1024},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			min, tooSmall := cachePrefixTooSmall(tc.model, tc.prefix)
			if tooSmall != tc.wantTooSmall {
				t.Errorf("cachePrefixTooSmall(%s, %d chars) tooSmall = %v, want %v",
					tc.model, len(tc.prefix), tooSmall, tc.wantTooSmall)
			}
			if min != tc.wantMin {
				t.Errorf("minimum for %s = %d, want %d", tc.model, min, tc.wantMin)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Generate wire shape — end to end through the HTTP client
// ---------------------------------------------------------------------------

// captureGenerateBody runs one Generate call against a stub server and returns
// the raw JSON request body the client actually sent.
func captureGenerateBody(t *testing.T, req *ai.Request) map[string]any {
	t.Helper()
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant",
			"content":[{"type":"text","text":"ok"}],"model":"claude-sonnet-5",
			"stop_reason":"end_turn","usage":{"input_tokens":10,"output_tokens":2}}`))
	}))
	defer srv.Close()

	c := NewClient("test-key", WithBaseURL(srv.URL))
	if _, err := c.Generate(context.Background(), req); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	return captured
}

// TestGenerate_NoBreakpoints_ByteIdenticalShape is the back-compat gate: a
// request that declares nothing must look exactly as it did in v0.30.0 —
// `system` a bare string, message content a bare string.
func TestGenerate_NoBreakpoints_ByteIdenticalShape(t *testing.T) {
	body := captureGenerateBody(t, &ai.Request{
		Model:        "claude-sonnet-5",
		SystemPrompt: "You are helpful.",
		UserPrompt:   "Hello",
		MaxTokens:    2048,
	})

	if got, ok := body["system"].(string); !ok || got != "You are helpful." {
		t.Errorf("system = %#v, want the bare string \"You are helpful.\" (back-compat)", body["system"])
	}
	msgs, _ := body["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}
	m, _ := msgs[0].(map[string]any)
	if got, ok := m["content"].(string); !ok || got != "Hello" {
		t.Errorf("message content = %#v, want the bare string \"Hello\" (back-compat)", m["content"])
	}
}

// TestGenerate_UserPrefixBreakpoint_EmitsCacheControl closes V1: the Generate
// path can now request a cache.
func TestGenerate_UserPrefixBreakpoint_EmitsCacheControl(t *testing.T) {
	teaching := strings.Repeat("TEACHING ", 4096) // comfortably over every minimum
	body := captureGenerateBody(t, &ai.Request{
		Model:            "claude-sonnet-5",
		SystemPrompt:     "You are helpful.",
		CachedPrefix:     teaching,
		UserPrompt:       "\n\n## Task\n\nSolve it",
		MaxTokens:        2048,
		CacheBreakpoints: []ai.CacheBreakpoint{{Position: "user_prefix", TTL: "ephemeral"}},
	})

	msgs, _ := body["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}
	m, _ := msgs[0].(map[string]any)
	blocks, ok := m["content"].([]any)
	if !ok {
		t.Fatalf("message content = %#v, want a content array", m["content"])
	}
	if len(blocks) != 2 {
		t.Fatalf("got %d content blocks, want 2", len(blocks))
	}
	b0, _ := blocks[0].(map[string]any)
	cc, ok := b0["cache_control"].(map[string]any)
	if !ok || cc["type"] != "ephemeral" {
		t.Errorf("block[0].cache_control = %#v, want {type: ephemeral}", b0["cache_control"])
	}
	b1, _ := blocks[1].(map[string]any)
	if _, present := b1["cache_control"]; present {
		t.Error("block[1] must not carry cache_control")
	}
}

// TestGenerate_SystemBreakpoint_EmitsCacheControl: the system position works on
// Generate too, not just on Step.
func TestGenerate_SystemBreakpoint_EmitsCacheControl(t *testing.T) {
	body := captureGenerateBody(t, &ai.Request{
		Model:            "claude-sonnet-5",
		SystemPrompt:     strings.Repeat("SYSTEM ", 4096),
		UserPrompt:       "Hello",
		MaxTokens:        2048,
		CacheBreakpoints: []ai.CacheBreakpoint{{Position: "system", TTL: "ephemeral"}},
	})

	sys, ok := body["system"].([]any)
	if !ok {
		t.Fatalf("system = %#v, want a content array when a system breakpoint is declared", body["system"])
	}
	if len(sys) != 1 {
		t.Fatalf("got %d system blocks, want 1", len(sys))
	}
	s0, _ := sys[0].(map[string]any)
	cc, ok := s0["cache_control"].(map[string]any)
	if !ok || cc["type"] != "ephemeral" {
		t.Errorf("system block cache_control = %#v, want {type: ephemeral}", s0["cache_control"])
	}
}

// TestGenerate_CachedPrefixWithoutBreakpoint_StillReachesModel: a caller that
// sets CachedPrefix but declares no breakpoint must still have that text sent —
// silently dropping it would corrupt the prompt.
func TestGenerate_CachedPrefixWithoutBreakpoint_StillReachesModel(t *testing.T) {
	body := captureGenerateBody(t, &ai.Request{
		Model:        "claude-sonnet-5",
		CachedPrefix: "TEACHING",
		UserPrompt:   "\n\n## Task\n\nSolve it",
		MaxTokens:    2048,
	})

	msgs, _ := body["messages"].([]any)
	m, _ := msgs[0].(map[string]any)
	got, ok := m["content"].(string)
	if !ok {
		t.Fatalf("message content = %#v, want a bare string", m["content"])
	}
	if want := "TEACHING\n\n## Task\n\nSolve it"; got != want {
		t.Errorf("message content = %q, want %q — the prefix must not be dropped", got, want)
	}
}
