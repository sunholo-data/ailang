package ollama

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// TestResolveOllamaNumCtx_DefaultOmits pins the fix for a config that could not
// work: both option maps in this package hardcoded num_ctx 8192 while the eval
// harness sends these models 28k-44k-token prompts, and asked for up to 32768
// tokens of generation inside that same 8192. Omitting the option lets ollama
// size context from the model (measured 262144 for qwen3.6:35b-a3b-mxfp8), which
// is what the /v1 lanes — opencode and pi — already get.
func TestResolveOllamaNumCtx_DefaultOmits(t *testing.T) {
	t.Setenv("AILANG_OLLAMA_NUM_CTX", "")
	if n, ok := resolveOllamaNumCtx(); ok {
		t.Errorf("default must send NO num_ctx so the server sizes it; got %d", n)
	}
}

func TestResolveOllamaNumCtx_EnvPins(t *testing.T) {
	t.Setenv("AILANG_OLLAMA_NUM_CTX", "65536")
	n, ok := resolveOllamaNumCtx()
	if !ok || n != 65536 {
		t.Errorf("resolveOllamaNumCtx() = (%d, %v), want (65536, true)", n, ok)
	}
}

// A junk or non-positive value must fall back to "let the server decide" rather
// than to some invented number — an unparseable override is not a request for a
// small context.
func TestResolveOllamaNumCtx_RejectsJunk(t *testing.T) {
	for _, v := range []string{"0", "-1", "lots", "8192.5"} {
		t.Setenv("AILANG_OLLAMA_NUM_CTX", v)
		if n, ok := resolveOllamaNumCtx(); ok {
			t.Errorf("AILANG_OLLAMA_NUM_CTX=%q should be ignored, got %d", v, n)
		}
	}
}

// captureOllamaBody stands up a fake ollama and returns the request body the
// client actually sent.
func captureOllamaBody(t *testing.T, send func(c *Client) error) string {
	t.Helper()
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"ok"},"done":true}` + "\n"))
	}))
	defer srv.Close()
	t.Setenv("OLLAMA_HOST", srv.URL)
	c, err := NewClient(WithEndpoint(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if err := send(c); err != nil {
		t.Fatalf("send: %v", err)
	}
	return body
}

// TestOllamaWire_NoNumCtxByDefault asserts on the BYTES, not the resolver: both
// option maps in this package used to pin num_ctx to 8192, which sat below the
// 28k-44k prompts the eval harness sends these models — so the model answered
// having seen a fraction of the task, while the same map asked for up to 32768
// tokens of generation inside that 8192. The wire must now carry no num_ctx, so
// ollama sizes context from the model itself.
func TestOllamaWire_NoNumCtxByDefault(t *testing.T) {
	t.Setenv("AILANG_OLLAMA_NUM_CTX", "")

	t.Run("generate path", func(t *testing.T) {
		body := captureOllamaBody(t, func(c *Client) error {
			_, err := c.Generate(context.Background(), &ai.Request{Model: "qwen3.5", UserPrompt: "hi"})
			return err
		})
		if strings.Contains(body, "num_ctx") {
			t.Errorf("Generate must not pin num_ctx; body: %s", body)
		}
	})

	t.Run("native chat path", func(t *testing.T) {
		t.Setenv("AILANG_OLLAMA_NATIVE_TOOLS", "1")
		body := captureOllamaBody(t, func(c *Client) error {
			_, err := c.Step(context.Background(), &ai.Request{
				Model:    "qwen3.5",
				Messages: []ai.Message{{Role: "user", Content: "hi"}},
				Tools:    []ai.ToolSchema{{Name: "noop", Description: "n", Parameters: `{"type":"object"}`}},
			})
			return err
		})
		if strings.Contains(body, "num_ctx") {
			t.Errorf("native /api/chat must not pin num_ctx; body: %s", body)
		}
	})
}

// The override still has to reach the wire, or it is not a control.
func TestOllamaWire_NumCtxOverrideReachesWire(t *testing.T) {
	t.Setenv("AILANG_OLLAMA_NUM_CTX", "65536")
	body := captureOllamaBody(t, func(c *Client) error {
		_, err := c.Generate(context.Background(), &ai.Request{Model: "qwen3.5", UserPrompt: "hi"})
		return err
	})
	if !strings.Contains(body, `"num_ctx":65536`) {
		t.Errorf("AILANG_OLLAMA_NUM_CTX=65536 must appear on the wire; body: %s", body)
	}
}
