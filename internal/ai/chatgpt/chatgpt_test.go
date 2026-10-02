package chatgpt

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
)

func jwtWithExp(exp time.Time) string {
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d}`, exp.Unix())))
	return "h." + payload + ".s"
}

func writeAuth(t *testing.T, mode, token, account string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	body := fmt.Sprintf(`{"auth_mode":%q,"tokens":{"access_token":%q,"account_id":%q}}`, mode, token, account)
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadCredential(t *testing.T) {
	writeAuth(t, "chatgpt", jwtWithExp(time.Now().Add(time.Hour)), "acct")
	if c, err := LoadCredential(); err != nil || c.AccountID != "acct" {
		t.Fatalf("valid ChatGPT login refused: %+v %v", c, err)
	}

	// An API-key login bills per token: not this lane.
	writeAuth(t, "apikey", jwtWithExp(time.Now().Add(time.Hour)), "acct")
	if _, err := LoadCredential(); err == nil || !strings.Contains(err.Error(), "apikey") {
		t.Fatalf("apikey-mode auth.json must be refused, got %v", err)
	}

	// An expired token fails loudly with the refresh command, not a bare 401 later.
	writeAuth(t, "chatgpt", jwtWithExp(time.Now().Add(-time.Minute)), "acct")
	if _, err := LoadCredential(); err == nil || !strings.Contains(err.Error(), "run any `codex` command") {
		t.Fatalf("expired token must be refused with the refresh hint, got %v", err)
	}

	writeAuth(t, "chatgpt", jwtWithExp(time.Now().Add(time.Hour)), "")
	if _, err := LoadCredential(); err == nil {
		t.Fatal("a login without an account id must be refused")
	}

	t.Setenv("CODEX_HOME", t.TempDir())
	if _, err := LoadCredential(); err == nil || !strings.Contains(err.Error(), "codex login") {
		t.Fatalf("missing auth.json must name `codex login`, got %v", err)
	}
}

func TestBuildRequest_MapsTurnsAndTools(t *testing.T) {
	req := &ai.Request{
		Model:        "chatgpt/gpt-6.1-sol",
		SystemPrompt: "base system",
		Messages: []ai.Message{
			{Role: "system", Content: "extra system"},
			{Role: "user", Content: "read a.txt"},
			{Role: "assistant", Content: "reading", ToolCalls: []ai.ToolCall{{ID: "call_1", Name: "ReadFile", Arguments: `{"path":"a.txt"}`}}},
			{Role: "tool", ToolCallID: "call_1", Content: "hello"},
		},
		Tools:           []ai.ToolSchema{{Name: "ReadFile", Description: "read", Parameters: `{"type":"object","properties":{"path":{"type":"string"}}}`}},
		ReasoningEffort: "high",
	}
	got := buildRequest(req)
	if got.Model != "gpt-6.1-sol" || !got.Stream || got.Store {
		t.Fatalf("model/stream/store wrong: %+v", got)
	}
	if got.Instructions != "base system\n\nextra system" {
		t.Fatalf("instructions = %q", got.Instructions)
	}
	types := []string{}
	for _, it := range got.Input {
		types = append(types, it["type"].(string))
	}
	if strings.Join(types, ",") != "message,message,function_call,function_call_output" {
		t.Fatalf("input item order = %v", types)
	}
	if got.Input[2]["call_id"] != "call_1" || got.Input[3]["call_id"] != "call_1" {
		t.Fatalf("call_id must pair the call with its output: %v / %v", got.Input[2], got.Input[3])
	}
	if len(got.Tools) != 1 || got.ToolChoice != "auto" || got.Reasoning["effort"] != "high" {
		t.Fatalf("tools/choice/reasoning wrong: %+v", got)
	}
	if _, ok := got.Tools[0]["parameters"].(map[string]any); !ok {
		t.Fatalf("tool parameters must be a JSON object, got %T", got.Tools[0]["parameters"])
	}
}

func sse(events ...string) string {
	var b strings.Builder
	for _, e := range events {
		b.WriteString("event: x\ndata: " + e + "\n\n")
	}
	return b.String()
}

func TestParseStream(t *testing.T) {
	completed := `{"type":"response.completed","response":{"model":"gpt-6.1-sol","status":"completed","usage":{"input_tokens":23,"output_tokens":5,"total_tokens":28,"input_tokens_details":{"cached_tokens":3},"output_tokens_details":{"reasoning_tokens":2}}}}`
	t.Run("text and usage", func(t *testing.T) {
		var deltas []string
		var usage ai.StreamUsage
		resp, err := parseStream(strings.NewReader(sse(
			`{"type":"response.output_text.delta","delta":"O"}`,
			`{"type":"response.output_text.delta","delta":"K"}`, completed)), "gpt-6.1-sol",
			func(c ai.StreamChunk) {
				switch v := c.(type) {
				case ai.StreamContentDelta:
					deltas = append(deltas, v.Text)
				case ai.StreamUsage:
					usage = v
				}
			})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Text != "OK" || resp.FinishReason != "stop" || resp.InputTokens != 23 || resp.OutputTokens != 5 ||
			resp.ReasonTokens != 2 || resp.CacheReadInputTokens != 3 || strings.Join(deltas, "") != "OK" || usage.InputTokens != 23 {
			t.Fatalf("resp = %+v deltas=%v usage=%+v", resp, deltas, usage)
		}
	})
	t.Run("tool call", func(t *testing.T) {
		resp, err := parseStream(strings.NewReader(sse(
			`{"type":"response.output_item.added","item":{"type":"function_call","call_id":"call_9","name":"ReadFile","arguments":""}}`,
			`{"type":"response.output_item.done","item":{"type":"function_call","call_id":"call_9","name":"ReadFile","arguments":"{\"path\":\"a\"}"}}`,
			completed)), "m", nil)
		if err != nil {
			t.Fatal(err)
		}
		if resp.FinishReason != "tool_calls" || len(resp.ToolCalls) != 1 || resp.ToolCalls[0].ID != "call_9" || resp.ToolCalls[0].Arguments != `{"path":"a"}` {
			t.Fatalf("resp = %+v", resp)
		}
	})
	t.Run("cut off at the output cap", func(t *testing.T) {
		resp, err := parseStream(strings.NewReader(sse(
			`{"type":"response.incomplete","response":{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"}}}`)), "m", nil)
		if err != nil || resp.FinishReason != "length" {
			t.Fatalf("want finish length, got %+v %v", resp, err)
		}
	})
	t.Run("failure and truncation are errors", func(t *testing.T) {
		for name, body := range map[string]string{
			"failed":    sse(`{"type":"response.failed","response":{"error":{"code":"server_error","message":"boom"}}}`),
			"error":     sse(`{"type":"error","code":"rate_limit","message":"slow down"}`),
			"truncated": sse(`{"type":"response.output_text.delta","delta":"partial"}`),
		} {
			if _, err := parseStream(strings.NewReader(body), "m", nil); err == nil {
				t.Errorf("%s: want an error, got none", name)
			}
		}
	})
}

// The request on the wire: the subscription token and account, an honest
// originator, and the bare model id.
func TestStreamStep_WireRequest(t *testing.T) {
	var gotHeaders http.Header
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		if r.URL.Path != "/responses" {
			http.Error(w, "wrong path "+r.URL.Path, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, sse(`{"type":"response.output_text.delta","delta":"hi"}`,
			`{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":1,"output_tokens":1}}}`))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithCredential(Credential{AccessToken: "tok", AccountID: "acct"}))
	resp, err := c.Step(context.Background(), &ai.Request{Model: "chatgpt/gpt-6.1-sol", UserPrompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "hi" {
		t.Fatalf("text = %q", resp.Text)
	}
	if gotHeaders.Get("Authorization") != "Bearer tok" || gotHeaders.Get("chatgpt-account-id") != "acct" || gotHeaders.Get("originator") != "ailang" {
		t.Fatalf("headers = %v", gotHeaders)
	}
	if gotBody["model"] != "gpt-6.1-sol" || gotBody["store"] != false || gotBody["stream"] != true {
		t.Fatalf("body = %v", gotBody)
	}

	// A non-2xx answer is a typed error carrying the status, not a parse attempt.
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":{"message":"usage limit reached"}}`, http.StatusTooManyRequests)
	}))
	defer bad.Close()
	_, err = NewClient(WithBaseURL(bad.URL), WithCredential(Credential{AccessToken: "t", AccountID: "a"})).
		Step(context.Background(), &ai.Request{Model: "gpt-6.1-sol", UserPrompt: "x"})
	var aiErr *ai.AIError
	if err == nil || !errors.As(err, &aiErr) || aiErr.Code != ai.CodeRateLimit {
		t.Fatalf("429 must classify as RateLimit, got %v", err)
	}
}

func TestGuessProvider_ChatGPTPrefix(t *testing.T) {
	if got := ai.GuessProvider("chatgpt/gpt-6.1-sol"); got != ai.ProviderChatGPT {
		t.Fatalf("GuessProvider = %q, want chatgpt (the vendor/model check would send it to OpenRouter)", got)
	}
}
