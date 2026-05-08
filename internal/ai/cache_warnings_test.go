package ai

import (
	"bytes"
	"io"
	"os"
	"sync"
	"testing"
)

// captureStderr redirects os.Stderr to a pipe and returns a function that
// restores it and reads what was written.
func captureStderr(t *testing.T) func() string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	originalStderr := os.Stderr
	os.Stderr = w
	return func() string {
		_ = w.Close()
		os.Stderr = originalStderr
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		return buf.String()
	}
}

func TestWarnOnceCacheHintIgnored_FirstCallEmits(t *testing.T) {
	resetCacheWarningsForTesting()
	read := captureStderr(t)

	WarnOnceCacheHintIgnored("openai", "auto_cache")
	out := read()

	if !bytes.Contains([]byte(out), []byte("cache_hint_ignored_openai_auto_cache")) {
		t.Errorf("expected greppable token in stderr, got: %s", out)
	}
}

func TestWarnOnceCacheHintIgnored_SecondCallSilent(t *testing.T) {
	resetCacheWarningsForTesting()
	WarnOnceCacheHintIgnored("openai", "auto_cache") // priming call (output discarded)

	read := captureStderr(t)
	WarnOnceCacheHintIgnored("openai", "auto_cache")
	WarnOnceCacheHintIgnored("openai", "auto_cache")
	out := read()

	if out != "" {
		t.Errorf("expected silent on repeat, got: %s", out)
	}
}

func TestWarnOnceCacheHintIgnored_DifferentKeysAllEmit(t *testing.T) {
	resetCacheWarningsForTesting()
	read := captureStderr(t)

	WarnOnceCacheHintIgnored("openai", "auto_cache")
	WarnOnceCacheHintIgnored("gemini", "no_explicit_api")
	WarnOnceCacheHintIgnored("openrouter_routed_to_openai", "auto_cache")
	out := read()

	for _, want := range []string{
		"cache_hint_ignored_openai_auto_cache",
		"cache_hint_ignored_gemini_no_explicit_api",
		"cache_hint_ignored_openrouter_routed_to_openai_auto_cache",
	} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Errorf("missing %q in stderr; got: %s", want, out)
		}
	}
}

// TestWarnOnceCacheHintIgnored_ConcurrentCallsEmitOnce: 100 goroutines all
// calling with the same key produce exactly 1 line on stderr.
func TestWarnOnceCacheHintIgnored_ConcurrentCallsEmitOnce(t *testing.T) {
	resetCacheWarningsForTesting()
	read := captureStderr(t)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			WarnOnceCacheHintIgnored("openai", "auto_cache")
		}()
	}
	wg.Wait()
	out := read()

	count := bytes.Count([]byte(out), []byte("cache_hint_ignored_"))
	if count != 1 {
		t.Errorf("expected exactly 1 emission across 100 concurrent calls, got %d (output: %s)", count, out)
	}
}
