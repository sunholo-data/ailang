package protocol

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func gateRequest(token string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "https://svc.example/mcp/connect/", nil)
	r.Host = "svc.example"
	r.Header.Set("X-Forwarded-Proto", "https")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return r
}

func metadataFor(r *http.Request) string {
	return PublicBaseURL(r) + "/.well-known/oauth-protected-resource/mcp/connect/"
}

func TestBearerGate_Outcomes(t *testing.T) {
	cases := []struct {
		name       string
		token      string
		verify     VerifyFunc
		wantAdmit  bool
		wantStatus int
		wantHeader string // substring of WWW-Authenticate ("" = header absent)
	}{
		{"no token", "", func(context.Context, string) (bool, error) { return true, nil }, false, 401,
			`resource_metadata="https://svc.example/.well-known/oauth-protected-resource/mcp/connect/"`},
		{"rejected", "bad", func(context.Context, string) (bool, error) { return false, nil }, false, 401, `error="invalid_token"`},
		{"accepted", "good", func(_ context.Context, tok string) (bool, error) { return tok == "good", nil }, true, 200, ""},
		{"verifier error", "x", func(context.Context, string) (bool, error) { return true, errors.New("db down") }, false, 503, ""},
		{"verifier panic", "x", func(context.Context, string) (bool, error) { panic("boom") }, false, 503, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := NewBearerGate(c.verify, metadataFor, time.Second, 4)
			w := httptest.NewRecorder()
			admitted := g.Admit(w, gateRequest(c.token))
			status := w.Code
			if admitted {
				status = 200
			}
			if admitted != c.wantAdmit || status != c.wantStatus {
				t.Fatalf("admit=%v status=%d, want admit=%v status=%d (body %s)", admitted, status, c.wantAdmit, c.wantStatus, w.Body)
			}
			got := w.Header().Get("WWW-Authenticate")
			if c.wantHeader == "" && got != "" {
				t.Errorf("unexpected WWW-Authenticate %q", got)
			}
			if c.wantHeader != "" && !strings.Contains(got, c.wantHeader) {
				t.Errorf("WWW-Authenticate %q missing %q", got, c.wantHeader)
			}
			if status == 503 && w.Header().Get("Retry-After") == "" {
				t.Error("503 without Retry-After")
			}
		})
	}
}

// D7: a verifier that does not answer in time is refused with 503 at the
// deadline. The gate does not wait for it.
func TestBearerGate_TimeoutFailsClosed(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	g := NewBearerGate(func(context.Context, string) (bool, error) { <-release; return true, nil }, metadataFor, 100*time.Millisecond, 4)
	w := httptest.NewRecorder()
	start := time.Now()
	if g.Admit(w, gateRequest("slow")) {
		t.Fatal("slow verifier admitted")
	}
	if w.Code != 503 || time.Since(start) > time.Second {
		t.Fatalf("status %d after %v, want 503 near the 100ms deadline", w.Code, time.Since(start))
	}
}

// D7: verifiers stuck past the deadline keep their slots, so with the cap
// reached the next request is refused at once and goroutines stay bounded.
func TestBearerGate_SaturationBoundsGoroutines(t *testing.T) {
	const capN = 3
	release := make(chan struct{})
	var started atomic.Int32
	g := NewBearerGate(func(context.Context, string) (bool, error) {
		started.Add(1)
		<-release // ignores ctx, like a pure loop in AILANG
		return true, nil
	}, metadataFor, 20*time.Millisecond, capN)

	base := runtime.NumGoroutine()
	var wg sync.WaitGroup
	for i := 0; i < capN; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); g.Admit(httptest.NewRecorder(), gateRequest("t")) }()
	}
	wg.Wait() // each timed out with 503; their verifiers are still running
	w := httptest.NewRecorder()
	start := time.Now()
	if g.Admit(w, gateRequest("t")) || w.Code != 503 || time.Since(start) > 10*time.Millisecond {
		t.Fatalf("saturated gate: status %d after %v, want an immediate 503", w.Code, time.Since(start))
	}
	if n := started.Load(); n != capN {
		t.Errorf("%d verifiers started, want %d (the over-cap request must not start one)", n, capN)
	}
	if extra := runtime.NumGoroutine() - base; extra > capN+2 {
		t.Errorf("%d extra goroutines, want <= cap(%d)+2", extra, capN)
	}
	close(release)
	// Slots free once the stuck verifiers return.
	deadline := time.Now().Add(time.Second)
	for {
		w := httptest.NewRecorder()
		if g.Admit(w, gateRequest("t")) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("slots never freed: last status %d", w.Code)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestToolCallNames(t *testing.T) {
	cases := map[string][]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"parse"}}`:                                              {"parse"},
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`:                                                                        nil,
		`[{"method":"tools/list"},{"method":"tools/call","params":{"name":"a"}},{"method":"tools/call","params":{"name":"b"}}]`: {"a", "b"},
		`not json`: nil,
	}
	for body, want := range cases {
		got := ToolCallNames([]byte(body))
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("ToolCallNames(%s) = %v, want %v", body, got, want)
		}
	}
}

func TestBearerToken(t *testing.T) {
	for in, want := range map[string]string{"Bearer abc": "abc", "bearer abc": "abc", "Basic abc": "", "Bearer": "", "": "", "Bearer a b": ""} {
		if got := BearerToken(in); got != want {
			t.Errorf("BearerToken(%q) = %q, want %q", in, got, want)
		}
	}
}
