package chatgpt

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
)

func TestClientTimeoutDefaultAndOverride(t *testing.T) {
	if got := NewClient().httpClient.Timeout; got != 10*time.Minute {
		t.Fatalf("default: %v", got)
	}
	if got := NewClient(WithTimeout(time.Second)).httpClient.Timeout; got != time.Second {
		t.Fatalf("override: %v", got)
	}
}

// Continuous output must not reset the total response deadline. All three
// provider entry points share it, and caller cancellation must still win.
func TestClientStreamingDeadline(t *testing.T) {
	for _, mode := range []string{"generate", "step", "stream", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			stopped := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer close(stopped)
				w.Header().Set("Content-Type", "text/event-stream")
				tick := time.NewTicker(5 * time.Millisecond)
				defer tick.Stop()
				for {
					if _, err := fmt.Fprint(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"x\"}\n\n"); err != nil {
						return
					}
					w.(http.Flusher).Flush()
					select {
					case <-r.Context().Done():
						return
					case <-tick.C:
					}
				}
			}))
			defer server.Close()
			timeout := 100 * time.Millisecond
			if mode == "cancel" {
				timeout = 5 * time.Second
			}
			c := NewClient(WithBaseURL(server.URL), WithTimeout(timeout), WithCredential(Credential{AccessToken: "fixture", AccountID: "acct"}))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			req := &ai.Request{Model: "chatgpt/gpt-6.1-sol", UserPrompt: "hello"}
			start := time.Now()
			var err error
			var response *ai.Response
			chunks := 0
			switch mode {
			case "generate":
				response, err = c.Generate(ctx, req)
			case "step":
				response, err = c.Step(ctx, req)
			default:
				response, err = c.StreamStep(ctx, req, func(ai.StreamChunk) {
					chunks++
					if mode == "cancel" {
						cancel()
					}
				})
			}
			var ae *ai.AIError
			if response != nil || !errors.As(err, &ae) || ae.Code != ai.CodeTimeout {
				t.Fatalf("want timeout without partial success: %+v %v", response, err)
			}
			if elapsed := time.Since(start); elapsed > 2*time.Second {
				t.Fatalf("deadline not respected: %v", elapsed)
			}
			if (mode == "stream" || mode == "cancel") && chunks == 0 {
				t.Fatal("fixture must continuously stream before termination")
			}
			if mode == "cancel" && ctx.Err() != context.Canceled {
				t.Fatalf("caller cancellation: %v", ctx.Err())
			}
			select {
			case <-stopped:
			case <-time.After(time.Second):
				t.Fatal("server did not observe canceled request")
			}
		})
	}
}
