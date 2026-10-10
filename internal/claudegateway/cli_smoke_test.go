package claudegateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// Opt-in protocol check against the locally installed Claude binary. The child
// has an empty HOME, synthetic credentials, no tools and a fake-only upstream.
func TestClaudeCLIProtocolSmoke(t *testing.T) {
	if os.Getenv("CLAUDE_CREDIT_CLI_SMOKE") != "1" {
		t.Skip("opt-in local CLI protocol fixture")
	}
	binary, err := exec.LookPath("claude")
	if err != nil {
		t.Skip("Claude CLI unavailable")
	}
	stream := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg-test\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-haiku-5-5\",\"content\":[],\"stop_reason\":null,\"stop_sequence\":null,\"usage\":{\"input_tokens\":100,\"output_tokens\":0}}}\n\nevent: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\nevent: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hello\"}}\n\nevent: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\",\"stop_sequence\":null},\"usage\":{\"output_tokens\":10}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	calls := 0
	var mu sync.Mutex
	var rejection string
	g, a, cap := fixture(t, func(*http.Request) (*http.Response, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(stream))}, nil
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := httptest.NewRecorder()
		g.ServeHTTP(recorder, r)
		if r.URL.Path == "/v1/messages" && recorder.Code != 200 {
			mu.Lock()
			rejection = "beta=" + r.Header.Get("anthropic-beta") + " " + recorder.Body.String()
			mu.Unlock()
		}
		for k, v := range recorder.Header() {
			w.Header()[k] = v
		}
		w.WriteHeader(recorder.Code)
		w.Write(recorder.Body.Bytes())
	}))
	defer server.Close()
	home := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "--print", "--model", "claude-haiku-5-5", "--tools", "", "--max-turns", "1", "--no-session-persistence", "say hello")
	cmd.Dir = home
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "ANTHROPIC_API_KEY=" + cap, "ANTHROPIC_BASE_URL=" + server.URL, "ANTHROPIC_CUSTOM_HEADERS=Authorization: Bearer job-token\nX-Serverless-Authorization: Bearer job-token", "ANTHROPIC_DEFAULT_HAIKU_MODEL=claude-haiku-5-5", "ANTHROPIC_DEFAULT_SONNET_MODEL=claude-haiku-5-5", "ANTHROPIC_DEFAULT_OPUS_MODEL=claude-haiku-5-5", "CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS=1", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1", "ENABLE_TOOL_SEARCH=false", "CLAUDE_CODE_AUTO_MODE_SERVER=0"}
	out, err := cmd.CombinedOutput()
	mu.Lock()
	rejected := rejection
	forwarded := calls
	mu.Unlock()
	if err != nil || !strings.Contains(string(out), "hello") {
		t.Fatalf("CLI fixture: %v; rejection=%s; output=%s", err, rejected, out)
	}
	s, err := a.Status(context.Background(), AccountID)
	if err != nil || forwarded != 1 || s.Reserved != 0 || s.Settled != 15 {
		t.Fatalf("CLI guard accounting %+v calls%d err%v", s, forwarded, err)
	}
}
