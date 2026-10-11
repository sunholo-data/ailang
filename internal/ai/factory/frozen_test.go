package factory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/testutil"
)

func TestFreezeConfiguration_EndpointCredentialAndOllama(t *testing.T) {
	isolate(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer selected-key" {
			t.Error("credential source changed")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"frozen"}}]}`))
	}))
	defer server.Close()
	t.Setenv("OPENAI_BASE_URL", server.URL)
	t.Setenv("OPENAI_API_KEY", "selected-key")
	freeze := FreezeConfiguration()
	t.Setenv("OPENAI_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("OPENAI_API_KEY", "late-key")
	c, err := New("openai", freeze)
	if err != nil {
		t.Fatal(err)
	}
	if response, err := c.Provider.Generate(context.Background(), &ai.Request{Model: "gpt-4o", UserPrompt: "hello"}); err != nil || response.Text != "frozen" {
		t.Fatalf("request: %+v %v", response, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}

	// Ollama readiness is not checked by preparation, and even the constructor's
	// ordinary OLLAMA_HOST override cannot move a prepared client afterwards.
	var probes atomic.Int32
	daemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		probes.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"models":[]}`))
	}))
	defer daemon.Close()
	t.Setenv("OLLAMA_HOST", daemon.URL)
	ollamaFreeze := FreezeConfiguration()
	if probes.Load() != 0 {
		t.Fatal("preparation probed Ollama")
	}
	t.Setenv("OLLAMA_HOST", "http://127.0.0.1:1")
	local, err := New("ollama", ollamaFreeze)
	if err != nil {
		t.Fatal(err)
	}
	if err := local.Provider.(interface{ CheckConnection(context.Context) error }).CheckConnection(context.Background()); err != nil {
		t.Fatal(err)
	}
	if probes.Load() != 1 {
		t.Fatalf("probes=%d", probes.Load())
	}
}

func TestFreezeConfiguration_ChatGPTPathAndRefresh(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	freeze := FreezeConfiguration() // auth.json deliberately absent until first use
	write := func(token string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(`{"auth_mode":"chatgpt","tokens":{"access_token":"`+token+`","account_id":"acct"}}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("first-token")
	t.Setenv("CODEX_HOME", t.TempDir())
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := "Bearer first-token"
		if calls.Load() == 2 {
			want = "Bearer refreshed-token"
		}
		if r.Header.Get("Authorization") != want {
			t.Errorf("credential %q want %q", r.Header.Get("Authorization"), want)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"ok\"}]}]}}\n\n"))
	}))
	defer server.Close()
	c, err := New("chatgpt", freeze, WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if i == 1 {
			write("refreshed-token")
		}
		if _, err := c.Provider.Generate(context.Background(), &ai.Request{Model: "gpt-4o", UserPrompt: "hi"}); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestFreezeConfiguration_AnthropicDeferredFileSource(t *testing.T) {
	isolate(t)
	fixtureHome := t.TempDir()
	testutil.SetHomeDir(t, fixtureHome)
	freeze := FreezeConfiguration()
	if err := os.MkdirAll(filepath.Join(fixtureHome, ".claude"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixtureHome, ".claude", ".credentials.json"), []byte(`{"claudeAiOauth":{"accessToken":"selected-file-token"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANTHROPIC_API_KEY", "late-metered-key")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "late-token")
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer selected-file-token" || r.Header.Get("x-api-key") != "" {
			t.Error("auth lane/source changed")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content":[{"type":"text","text":"frozen"}],"usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()
	client, err := New("anthropic", freeze, WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if client.Lane != LaneOAuth {
		t.Fatalf("lane=%s", client.Lane)
	}
	if _, err := client.Provider.Generate(context.Background(), &ai.Request{Model: "claude-test", UserPrompt: "hi"}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}
