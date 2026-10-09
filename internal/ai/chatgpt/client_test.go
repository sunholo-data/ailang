package chatgpt

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
)

func TestDeadlineDefaults(t *testing.T) {
	if got := NewClient().httpClient.Timeout; got != 10*time.Minute {
		t.Fatalf("default deadline: %v", got)
	}
	if got := NewClient(WithTimeout(time.Second)).httpClient.Timeout; got != time.Second {
		t.Fatalf("override: %v", got)
	}
}
func TestContinuouslyStreamingDeadline(t *testing.T) {
	var heartbeats atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
					return
				}
				w.(http.Flusher).Flush()
				heartbeats.Add(1)
			}
		}
	}))
	defer server.Close()
	for _, cancelEarlier := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelEarlier), func(t *testing.T) {
			timeout := 100 * time.Millisecond
			ctx, guard := context.WithTimeout(context.Background(), 2*time.Second)
			defer guard()
			if cancelEarlier {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 30*time.Millisecond)
				defer cancel()
				timeout = time.Second
			}
			c := NewClient(WithBaseURL(server.URL), WithCredential(Credential{AccessToken: "token", AccountID: "acct"}), WithTimeout(timeout))
			before := heartbeats.Load()
			start := time.Now()
			_, err := c.Generate(ctx, &ai.Request{Model: "chatgpt/gpt-6.1-sol", UserPrompt: "hello"})
			if err == nil {
				t.Fatal("unfinished streaming response succeeded")
			}
			if heartbeats.Load() <= before {
				t.Fatal("server did not stream before deadline")
			}
			if !strings.Contains(err.Error(), "deadline") && !strings.Contains(err.Error(), "Timeout") {
				t.Fatalf("unexpected termination: %v", err)
			}
			if elapsed := time.Since(start); elapsed > 700*time.Millisecond {
				t.Fatalf("deadline did not bound stream: %v, %v", elapsed, err)
			}
		})
	}
}
