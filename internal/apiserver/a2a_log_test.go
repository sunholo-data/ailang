package apiserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuf is a goroutine-safe log sink: the handler logs after the response.
type syncBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuf) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// lines waits briefly for n lines: the log line is written after the handler
// returns, which can be just after the client has read the response.
func (b *syncBuf) lines(t *testing.T, n int) []map[string]any {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		b.mu.Lock()
		raw := strings.TrimSpace(b.buf.String())
		b.mu.Unlock()
		var out []map[string]any
		if raw != "" {
			for _, l := range strings.Split(raw, "\n") {
				var m map[string]any
				if err := json.Unmarshal([]byte(l), &m); err != nil {
					t.Fatalf("log line is not JSON: %q", l)
				}
				out = append(out, m)
			}
		}
		if len(out) >= n || time.Now().After(deadline) {
			return out
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func captureA2ALog(t *testing.T) *syncBuf {
	t.Helper()
	b := &syncBuf{}
	old := a2aLogOut
	a2aLogOut = b
	t.Cleanup(func() { a2aLogOut = old })
	return b
}

func TestA2ARequestLogLineSend(t *testing.T) {
	logs := captureA2ALog(t)
	ts := newA2AServer(t, "reply")
	a2aPost(t, ts.URL, "message/send", a2aUserMessage("secret words"))
	a2aPost(t, ts.URL, "message/send", a2aUserMessage("x"))
	a2aPost(t, ts.URL, "tasks/get", map[string]any{"id": "nope"})
	a2aPost(t, ts.URL, "no/such", map[string]any{})

	got := logs.lines(t, 4)
	want := [][2]string{{"message/send", "completed"}, {"message/send", "failed"}, {"tasks/get", "not_found"}, {"no/such", "unknown_method"}}
	if len(got) != len(want) {
		t.Fatalf("log lines = %#v, want %d", got, len(want))
	}
	for i, w := range want {
		l := got[i]
		if l["event"] != "a2a_request" || l["method"] != w[0] || l["outcome"] != w[1] {
			t.Errorf("line %d = %#v, want method %s outcome %s", i, l, w[0], w[1])
		}
		if _, ok := l["duration_ms"].(float64); !ok {
			t.Errorf("line %d has no duration_ms: %#v", i, l)
		}
		if _, ok := l["heartbeats"]; ok {
			t.Errorf("line %d: heartbeats on a non-stream method: %#v", i, l)
		}
	}
	logs.mu.Lock()
	all := logs.buf.String()
	logs.mu.Unlock()
	if strings.Contains(all, "secret words") || strings.Contains(all, "SECRET WORDS") {
		t.Fatalf("the request or answer text reached the log: %s", all)
	}
}

func TestA2ARequestLogLineStream(t *testing.T) {
	logs := captureA2ALog(t)
	old := a2aHeartbeat
	a2aHeartbeat = 20 * time.Millisecond
	t.Cleanup(func() { a2aHeartbeat = old })
	ts := newA2AServer(t, "slow")
	_, beats := a2aStream(t, ts.URL, "late")

	got := logs.lines(t, 1)
	if len(got) != 1 {
		t.Fatalf("log lines = %#v, want one", got)
	}
	l := got[0]
	if l["method"] != "message/stream" || l["outcome"] != "completed" {
		t.Fatalf("line = %#v, want message/stream completed", l)
	}
	if hb, _ := l["heartbeats"].(float64); int(hb) != beats || beats < 2 {
		t.Fatalf("logged heartbeats %v, client saw %d (want the same, at least 2)", l["heartbeats"], beats)
	}
	if d, _ := l["duration_ms"].(float64); d < 250 {
		t.Fatalf("duration_ms = %v for a 300ms handler", l["duration_ms"])
	}
}

func TestA2ARequestLogLineBadRequest(t *testing.T) {
	logs := captureA2ALog(t)
	ts := newA2AServer(t, "reply")
	resp, err := http.Post(ts.URL+"/a2a/", "application/json", strings.NewReader("{not json"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	long := strings.Repeat("m", 500)
	a2aPost(t, ts.URL, long, map[string]any{})

	got := logs.lines(t, 2)
	if len(got) != 2 || got[0]["outcome"] != "bad_request" || got[0]["method"] != "" {
		t.Fatalf("lines = %#v, want bad_request with no method first", got)
	}
	if m, _ := got[1]["method"].(string); len(m) != a2aMaxLoggedMethod {
		t.Fatalf("a %d-character method logged as %d characters, want clipped to %d", len(long), len(m), a2aMaxLoggedMethod)
	}
}
