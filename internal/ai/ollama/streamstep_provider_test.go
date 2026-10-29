package ollama

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
)

// The caller of StreamStep sees each streamed delta WHILE the call runs — the
// property motoko's session log (and so the executor's idle watchdog) needs.
func TestStreamStep_ForwardsDeltasDuringTheCall(t *testing.T) {
	const gap = 400 * time.Millisecond
	f := newFakeV1(t, func(f *fakeV1, w http.ResponseWriter, r *http.Request) {
		sseHeaders(w)
		for _, s := range []string{"he", "ll", "o"} {
			if !sseWrite(w, contentChunk(s)) {
				return
			}
			time.Sleep(gap)
		}
		sseWrite(w, finishChunk("stop"))
		sseWrite(w, doneChunk)
	})
	streamEnv(t, f.srv.URL)
	c := newTestClient(t, f.srv.URL)

	var mu sync.Mutex
	var texts []string
	var firstAt time.Time
	resp, err := c.StreamStep(context.Background(), toolReq(), func(ch ai.StreamChunk) {
		if d, ok := ch.(ai.StreamContentDelta); ok {
			mu.Lock()
			if firstAt.IsZero() {
				firstAt = time.Now()
			}
			texts = append(texts, d.Text)
			mu.Unlock()
		}
	})
	returned := time.Now()
	if err != nil {
		t.Fatalf("StreamStep: %v", err)
	}
	if resp.Text != "hello" {
		t.Fatalf("Text = %q, want hello", resp.Text)
	}
	if len(texts) != 3 {
		t.Fatalf("caller got content deltas %q, want the 3 streamed ones", texts)
	}
	if returned.Sub(firstAt) < 2*gap {
		t.Fatalf("first delta reached the caller %v before the call returned; want >= %v (streamed, not replayed at the end)", returned.Sub(firstAt), 2*gap)
	}
}

// Off the streaming path, StreamStep does what the handler's fallback did:
// one ContentDelta with the whole text, then Usage.
func TestStreamStep_BufferedPathKeepsTheSyntheticChunks(t *testing.T) {
	f := newFakeV1(t, func(f *fakeV1, w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"c1","object":"chat.completion","model":"qwen3.6","choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":1,"total_tokens":6}}`)
	})
	streamEnv(t, f.srv.URL)
	t.Setenv("AILANG_OLLAMA_V1_STREAM", "")
	c := newTestClient(t, f.srv.URL)

	var got []ai.StreamChunk
	resp, err := c.StreamStep(context.Background(), toolReq(), func(ch ai.StreamChunk) { got = append(got, ch) })
	if err != nil {
		t.Fatalf("StreamStep: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("chunks = %#v, want ContentDelta + Usage", got)
	}
	if d, ok := got[0].(ai.StreamContentDelta); !ok || d.Text != resp.Text || d.Text != "hi" {
		t.Fatalf("first chunk = %#v, want ContentDelta %q", got[0], "hi")
	}
	if u, ok := got[1].(ai.StreamUsage); !ok || u.InputTokens != resp.InputTokens {
		t.Fatalf("second chunk = %#v, want Usage matching the response", got[1])
	}
}
