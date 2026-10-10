package claudegateway

import (
	"bytes"
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

	"github.com/sunholo-data/ailang/internal/creditbudget"
)

// Opt-in protocol check against the locally installed Claude binary. The child
// has an empty HOME, synthetic credentials and a fake-only upstream. The tool
// fixture requests only printf inside that empty temporary workspace.
func TestClaudeCLIProtocolSmoke(t *testing.T) {
	if os.Getenv("CLAUDE_CREDIT_CLI_SMOKE") != "1" {
		t.Skip("opt-in local CLI protocol fixture")
	}
	binary, err := exec.LookPath("claude")
	if err != nil {
		t.Skip("Claude CLI unavailable")
	}
	for _, tool := range []string{"", "Bash"} {
		t.Run("tools="+tool, func(t *testing.T) { runClaudeCLIProtocolSmoke(t, binary, tool) })
	}
}

func runClaudeCLIProtocolSmoke(t *testing.T, binary, tool string) {
	t.Helper()
	stream := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg-test\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-haiku-5-5\",\"content\":[],\"stop_reason\":null,\"stop_sequence\":null,\"usage\":{\"input_tokens\":100,\"output_tokens\":0}}}\n\nevent: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\nevent: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hello\"}}\n\nevent: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\",\"stop_sequence\":null},\"usage\":{\"output_tokens\":10}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	calls := 0
	var mu sync.Mutex
	var rejection string
	g, a, cap := fixture(t, func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		calls++
		current := calls
		mu.Unlock()
		if r.Header.Get("anthropic-beta") != "" {
			t.Error("client compatibility mode forwarded a beta capability")
		}
		response := stream
		if tool != "" && current == 1 {
			response = strings.Replace(stream, `"type":"text","text":""`, `"type":"tool_use","id":"tool-fixture","name":"Bash","input":{}`, 1)
			response = strings.Replace(response, `"type":"text_delta","text":"hello"`, `"type":"input_json_delta","partial_json":"{\"command\":\"printf fixture-success\"}"`, 1)
			response = strings.Replace(response, `"stop_reason":"end_turn"`, `"stop_reason":"tool_use"`, 1)
		} else if tool != "" {
			body, err := io.ReadAll(r.Body)
			if err != nil || !bytes.Contains(body, []byte("tool_result")) || !bytes.Contains(body, []byte("fixture-success")) {
				t.Errorf("tool result missing from follow-up: %v", err)
			}
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(response))}, nil
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var contractErr error
		if r.URL.Path == "/v1/messages" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			_, _, _, contractErr = validateMessage(body)
		}
		recorder := httptest.NewRecorder()
		g.ServeHTTP(recorder, r)
		if r.URL.Path == "/v1/messages" && recorder.Code != 200 {
			mu.Lock()
			rejection = "beta=" + r.Header.Get("anthropic-beta") + " " + recorder.Body.String()
			if contractErr != nil {
				rejection += " body contract: " + contractErr.Error()
			}
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
	cmd := exec.CommandContext(ctx, binary, "--print", "--model", "claude-haiku-5-5", "--tools", tool, "--permission-mode", "bypassPermissions", "--max-turns", "2", "--no-session-persistence", "say hello")
	cmd.Dir = home
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "ANTHROPIC_API_KEY=" + cap, "ANTHROPIC_BASE_URL=" + server.URL, "ANTHROPIC_CUSTOM_HEADERS=Authorization: Bearer job-token\nX-Serverless-Authorization: Bearer job-token", "ANTHROPIC_DEFAULT_HAIKU_MODEL=claude-haiku-5-5", "ANTHROPIC_DEFAULT_SONNET_MODEL=claude-haiku-5-5", "ANTHROPIC_DEFAULT_OPUS_MODEL=claude-haiku-5-5", "CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS=1", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1", "ENABLE_TOOL_SEARCH=false", "CLAUDE_CODE_AUTO_MODE_SERVER=0", "CLAUDE_CODE_SIMULATE_PROXY_USAGE=1"}
	out, err := cmd.CombinedOutput()
	mu.Lock()
	rejected := rejection
	forwarded := calls
	mu.Unlock()
	if err != nil || !strings.Contains(string(out), "hello") {
		t.Fatalf("CLI fixture: %v; rejection=%s; output=%s", err, rejected, out)
	}
	s, err := a.Status(context.Background(), AccountID)
	wantCalls := 1
	if tool != "" {
		wantCalls = 2
	}
	if err != nil || forwarded != wantCalls || s.Reserved != 0 || s.Settled != 15*creditbudget.MicroUSD(wantCalls) {
		t.Fatalf("CLI guard accounting %+v calls%d err%v", s, forwarded, err)
	}
}
