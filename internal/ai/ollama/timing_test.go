package ollama

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
)

// TestClient_Generate_Timing pins the M-LYCEUM-PROVIDER M3 latency capture on
// the streaming native path: TTFT = call start → first stream callback, wall =
// full stream. The server delays the first NDJSON chunk so TTFT is measurably
// greater than the inter-chunk gap.
func TestClient_Generate_Timing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		time.Sleep(30 * time.Millisecond) // TTFT window
		_, _ = w.Write([]byte(`{"model":"test","created_at":"2026-09-03T10:00:00Z","response":"hel","done":false}` + "\n"))
		_, _ = w.Write([]byte(`{"model":"test","created_at":"2026-09-03T10:00:00Z","response":"lo","done":true,"prompt_eval_count":3,"eval_count":2}` + "\n"))
	}))
	defer srv.Close()
	// NewClient lets OLLAMA_HOST override WithEndpoint, and other tests in this
	// package leave it set process-wide — pin it or the request leaks to the
	// real server on localhost:11434.
	t.Setenv("OLLAMA_HOST", srv.URL)
	c, err := NewClient(WithEndpoint(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.Generate(context.Background(), &ai.Request{
		Model:      "test",
		UserPrompt: "say hello",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.Text != "hello" {
		t.Errorf("Text = %q, want \"hello\"", resp.Text)
	}
	if resp.WallMS <= 0 {
		t.Errorf("WallMS = %d, want > 0 (streaming round trip)", resp.WallMS)
	}
	if resp.TTFTMS <= 0 {
		t.Errorf("TTFTMS = %d, want > 0 (streaming transport observes first byte)", resp.TTFTMS)
	}
	if resp.TTFTMS > resp.WallMS {
		t.Errorf("TTFTMS (%d) must not exceed WallMS (%d)", resp.TTFTMS, resp.WallMS)
	}
	// The 30ms first-chunk delay must dominate: TTFT should account for most
	// of a two-chunk stream. Generous floor to stay robust on loaded CIs.
	if resp.TTFTMS < 20 {
		t.Errorf("TTFTMS = %d, want >= 20 (first chunk delayed 30ms)", resp.TTFTMS)
	}
}
