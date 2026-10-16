package motoko

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

// unsetEnv removes a variable for one test and restores it afterwards;
// t.Setenv cannot express "absent".
func unsetEnv(t *testing.T, name string) {
	t.Helper()
	prev, had := os.LookupEnv(name)
	os.Unsetenv(name)
	t.Cleanup(func() {
		if had {
			os.Setenv(name, prev)
		} else {
			os.Unsetenv(name)
		}
	})
}

func TestOllamaStreamEnv(t *testing.T) {
	var warn strings.Builder

	unsetEnv(t, config.EnvOllamaV1Stream)
	unsetEnv(t, config.EnvOllamaHTTPTimeoutSec)
	if got := ollamaStreamEnv("ollama/qwen3.8:27b-mxfp8", &warn); len(got) != 1 || got[0] != "AILANG_OLLAMA_V1_STREAM=1" {
		t.Fatalf("ollama model, nothing set: got %v, want streaming on", got)
	}
	if got := ollamaStreamEnv("openrouter/z-ai/glm-5.3-flash", &warn); got != nil {
		t.Fatalf("non-ollama model: got %v, want nothing", got)
	}

	t.Setenv(config.EnvOllamaHTTPTimeoutSec, "1800")
	if got := ollamaStreamEnv("ollama/m", &warn); len(got) != 1 {
		t.Fatalf("positive timeout: got %v, want streaming on", got)
	}

	// Streaming refuses a non-positive deadline at client construction.
	t.Setenv(config.EnvOllamaHTTPTimeoutSec, "0")
	warn.Reset()
	if got := ollamaStreamEnv("ollama/m", &warn); got != nil {
		t.Fatalf("timeout 0: got %v, want the buffered path kept", got)
	}
	if !strings.Contains(warn.String(), "buffered") {
		t.Fatalf("timeout 0 kept buffered without saying so: %q", warn.String())
	}

	// An explicit operator choice wins, including "0".
	t.Setenv(config.EnvOllamaHTTPTimeoutSec, "1800")
	t.Setenv(config.EnvOllamaV1Stream, "0")
	if got := ollamaStreamEnv("ollama/m", &warn); got != nil {
		t.Fatalf("explicit V1_STREAM=0: got %v, want nothing added", got)
	}
}

func TestOllamaCallBound(t *testing.T) {
	cases := []struct {
		env  []string
		want time.Duration
	}{
		{nil, 300 * time.Second},
		{[]string{"AILANG_OLLAMA_V1_STREAM=1"}, 3600 * time.Second},
		{[]string{"AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800"}, 1800 * time.Second},
		{[]string{"AILANG_OLLAMA_V1_STREAM=1", "AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800"}, 1800 * time.Second},
		{[]string{"AILANG_OLLAMA_HTTP_TIMEOUT_SEC=0"}, 0},
		{[]string{"AILANG_OLLAMA_V1_STREAM=0", "AILANG_OLLAMA_HTTP_TIMEOUT_SEC=junk"}, 300 * time.Second},
	}
	for _, c := range cases {
		if got := ollamaCallBound(c.env); got != c.want {
			t.Errorf("ollamaCallBound(%v) = %v, want %v", c.env, got, c.want)
		}
	}
}

func TestLastEventType(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.jsonl")
	body := `{"type":"thinking_stream_start","step":"0"}` + "\n" +
		`{"type":"provider_call_prepared","step":"0"}` + "\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := lastEventType(p); got != "provider_call_prepared" {
		t.Fatalf("lastEventType = %q, want provider_call_prepared", got)
	}
	if !inFlightEvents[lastEventType(p)] {
		t.Fatal("provider_call_prepared must count as a call in flight")
	}
	if inFlightEvents["thinking_stream_end"] || inFlightEvents["native_tool_results"] {
		t.Fatal("a closed stream or tool result must not count as in flight")
	}
	if got := lastEventType(filepath.Join(t.TempDir(), "missing")); got != "" {
		t.Fatalf("missing file: got %q, want empty", got)
	}
}

// The v0.54.9 shape: the log stops at provider_call_prepared while ollama
// generates. Past the idle limit but inside the call's bound, the run lives.
func TestWatchIdle_InFlightCallIsNotIdle(t *testing.T) {
	saved := callMargin
	callMargin = 0
	defer func() { callMargin = saved }()

	var fired atomic.Bool
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	watchIdle(ctx, 20*time.Millisecond, 5*time.Millisecond,
		func() int64 { return 7 }, func() bool { return true }, time.Hour,
		func() { fired.Store(true) })
	if fired.Load() {
		t.Fatal("killed a run whose provider call was still in flight, inside its bound")
	}
}

// An open call that outlives the provider's own deadline is a real hang.
func TestWatchIdle_InFlightPastBoundFires(t *testing.T) {
	saved := callMargin
	callMargin = 0
	defer func() { callMargin = saved }()

	done := make(chan struct{})
	var fired atomic.Int32
	go func() {
		watchIdle(context.Background(), 10*time.Millisecond, 5*time.Millisecond,
			func() int64 { return 7 }, func() bool { return true }, 40*time.Millisecond,
			func() { fired.Add(1) })
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("an in-flight call past its bound was never killed")
	}
	if fired.Load() != 1 {
		t.Fatalf("onIdle fired %d times, want 1", fired.Load())
	}
}
