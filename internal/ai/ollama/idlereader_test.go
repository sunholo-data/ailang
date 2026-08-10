package ollama

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// errWatchdogCanceled stands in for what the real transport does when the
// request context is cancelled: it fails the pending Read with *some* error.
// The point of the reader is that this generic error never reaches the caller.
var errWatchdogCanceled = errors.New("test transport: canceled by watchdog")

// newPipeIdleReader drives an idleReader from an io.Pipe. The cancel func closes
// the write half with an error, which is what unblocks a Read parked on the
// pipe — the same shape as a transport failing a read on context cancellation.
func newPipeIdleReader(t *testing.T, cfg idleReaderConfig) (*idleReader, *io.PipeWriter) {
	t.Helper()
	pr, pw := io.Pipe()
	var once sync.Once
	cancel := func() {
		once.Do(func() { _ = pw.CloseWithError(errWatchdogCanceled) })
	}
	r := newIdleReader(pr, cancel, cfg)
	t.Cleanup(func() { _ = r.Close() })
	return r, pw
}

type readResult struct {
	n   int
	err error
}

// readWithin runs one Read off-goroutine so a watchdog that never fires shows up
// as a fast, legible failure instead of a ten-minute package timeout.
func readWithin(t *testing.T, r io.Reader, limit time.Duration) (readResult, bool) {
	t.Helper()
	ch := make(chan readResult, 1)
	go func() {
		buf := make([]byte, 256)
		n, err := r.Read(buf)
		ch <- readResult{n: n, err: err}
	}()
	select {
	case res := <-ch:
		return res, true
	case <-time.After(limit):
		return readResult{}, false
	}
}

// countingBody records how many times Close was called on it.
type countingBody struct {
	r      io.Reader
	closes atomic.Int32
}

func (c *countingBody) Read(p []byte) (int, error) { return c.r.Read(p) }
func (c *countingBody) Close() error {
	c.closes.Add(1)
	return nil
}

// ---------------------------------------------------------------------------
// T1 — idle window
// ---------------------------------------------------------------------------

func TestIdleReader_IdleWindow(t *testing.T) {
	t.Run("bytes_then_silence_yields_ErrIdleTimeout", func(t *testing.T) {
		const idle = 80 * time.Millisecond
		r, pw := newPipeIdleReader(t, idleReaderConfig{
			TTFT: 2 * time.Second,
			Idle: idle,
			Tick: 5 * time.Millisecond,
		})

		go func() {
			for i := 0; i < 3; i++ {
				if _, err := pw.Write([]byte("chunk")); err != nil {
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
			// Then go silent, holding the pipe open. This is the S2 hang.
		}()

		for i := 0; i < 3; i++ {
			res, ok := readWithin(t, r, 2*time.Second)
			if !ok {
				t.Fatalf("chunk %d: Read did not return within 2s", i)
			}
			if res.err != nil {
				t.Fatalf("chunk %d: unexpected error: %v", i, res.err)
			}
			if res.n == 0 {
				t.Fatalf("chunk %d: expected bytes, got 0", i)
			}
		}

		start := time.Now()
		res, ok := readWithin(t, r, 3*time.Second)
		if !ok {
			t.Fatal("Read never returned: the idle watchdog did not fire")
		}
		if res.err == nil {
			t.Fatal("expected an idle timeout error, got nil")
		}
		if !errors.Is(res.err, ErrIdleTimeout) {
			t.Fatalf("expected ErrIdleTimeout, got %v", res.err)
		}
		if elapsed := time.Since(start); elapsed > idle+2*time.Second {
			t.Fatalf("idle timeout fired far too late: %v (window %v)", elapsed, idle)
		}
	})

	t.Run("generic_transport_error_is_not_surfaced", func(t *testing.T) {
		r, pw := newPipeIdleReader(t, idleReaderConfig{
			TTFT: 2 * time.Second,
			Idle: 60 * time.Millisecond,
			Tick: 5 * time.Millisecond,
		})
		go func() { _, _ = pw.Write([]byte("x")) }()

		if res, ok := readWithin(t, r, 2*time.Second); !ok || res.err != nil {
			t.Fatalf("first read: ok=%v err=%v", ok, res.err)
		}
		res, ok := readWithin(t, r, 3*time.Second)
		if !ok {
			t.Fatal("Read never returned: the idle watchdog did not fire")
		}
		if errors.Is(res.err, errWatchdogCanceled) {
			t.Fatalf("raw transport error leaked to the caller: %v", res.err)
		}
		if !errors.Is(res.err, ErrIdleTimeout) {
			t.Fatalf("expected ErrIdleTimeout, got %v", res.err)
		}
	})
}

// ---------------------------------------------------------------------------
// T2 — TTFT window
// ---------------------------------------------------------------------------

func TestIdleReader_TTFTWindow(t *testing.T) {
	t.Run("silence_over_idle_but_under_ttft_completes", func(t *testing.T) {
		r, pw := newPipeIdleReader(t, idleReaderConfig{
			TTFT: 600 * time.Millisecond,
			Idle: 60 * time.Millisecond, // deliberately shorter than the wait below
			Tick: 5 * time.Millisecond,
		})

		go func() {
			time.Sleep(200 * time.Millisecond) // > idle, < TTFT
			_, _ = pw.Write([]byte("first token"))
		}()

		res, ok := readWithin(t, r, 3*time.Second)
		if !ok {
			t.Fatal("Read never returned")
		}
		if res.err != nil {
			t.Fatalf("pre-first-byte silence inside the TTFT window must not fail; got %v", res.err)
		}
		if res.n == 0 {
			t.Fatal("expected the first token to be delivered")
		}
	})

	t.Run("silence_over_ttft_yields_ErrTTFTTimeout", func(t *testing.T) {
		r, _ := newPipeIdleReader(t, idleReaderConfig{
			TTFT: 80 * time.Millisecond,
			Idle: 2 * time.Second, // idle must not be what fires here
			Tick: 5 * time.Millisecond,
		})

		res, ok := readWithin(t, r, 3*time.Second)
		if !ok {
			t.Fatal("Read never returned: the TTFT watchdog did not fire")
		}
		if res.err == nil {
			t.Fatal("expected a TTFT timeout error, got nil")
		}
		if !errors.Is(res.err, ErrTTFTTimeout) {
			t.Fatalf("expected ErrTTFTTimeout, got %v", res.err)
		}
	})
}

// ---------------------------------------------------------------------------
// T3 — which window fired is preserved
// ---------------------------------------------------------------------------

func TestIdleReader_WhichWindowFiredIsPreserved(t *testing.T) {
	fireTTFT := func(t *testing.T) error {
		t.Helper()
		r, _ := newPipeIdleReader(t, idleReaderConfig{
			TTFT: 60 * time.Millisecond,
			Idle: 5 * time.Second,
			Tick: 5 * time.Millisecond,
		})
		res, ok := readWithin(t, r, 3*time.Second)
		if !ok {
			t.Fatal("TTFT case: Read never returned")
		}
		return res.err
	}

	fireIdle := func(t *testing.T) error {
		t.Helper()
		r, pw := newPipeIdleReader(t, idleReaderConfig{
			TTFT: 5 * time.Second,
			Idle: 60 * time.Millisecond,
			Tick: 5 * time.Millisecond,
		})
		go func() { _, _ = pw.Write([]byte("token")) }()
		if res, ok := readWithin(t, r, 2*time.Second); !ok || res.err != nil {
			t.Fatalf("idle case: priming read failed ok=%v err=%v", ok, res.err)
		}
		res, ok := readWithin(t, r, 3*time.Second)
		if !ok {
			t.Fatal("idle case: Read never returned")
		}
		return res.err
	}

	fireDeadline := func(t *testing.T) error {
		t.Helper()
		r, _ := newPipeIdleReader(t, idleReaderConfig{
			TTFT: 5 * time.Second,
			Idle: 5 * time.Second,
			Hard: 60 * time.Millisecond,
			Tick: 5 * time.Millisecond,
		})
		res, ok := readWithin(t, r, 3*time.Second)
		if !ok {
			t.Fatal("deadline case: Read never returned")
		}
		return res.err
	}

	var ttftErr, idleErr, deadlineErr error

	t.Run("ttft_case_yields_ErrTTFTTimeout", func(t *testing.T) {
		ttftErr = fireTTFT(t)
		if !errors.Is(ttftErr, ErrTTFTTimeout) {
			t.Fatalf("expected ErrTTFTTimeout, got %v", ttftErr)
		}
	})
	t.Run("idle_case_yields_ErrIdleTimeout", func(t *testing.T) {
		idleErr = fireIdle(t)
		if !errors.Is(idleErr, ErrIdleTimeout) {
			t.Fatalf("expected ErrIdleTimeout, got %v", idleErr)
		}
	})
	t.Run("deadline_case_yields_ErrStreamDeadlineExceeded", func(t *testing.T) {
		deadlineErr = fireDeadline(t)
		if !errors.Is(deadlineErr, ErrStreamDeadlineExceeded) {
			t.Fatalf("expected ErrStreamDeadlineExceeded, got %v", deadlineErr)
		}
	})

	// The whole point: one generic error for all three must NOT pass.
	t.Run("the_three_sentinels_are_pairwise_distinct", func(t *testing.T) {
		if ttftErr == nil || idleErr == nil || deadlineErr == nil {
			t.Fatalf("a prior subtest produced no error: ttft=%v idle=%v deadline=%v",
				ttftErr, idleErr, deadlineErr)
		}
		cases := []struct {
			name string
			err  error
			not  []error
		}{
			{"ttft", ttftErr, []error{ErrIdleTimeout, ErrStreamDeadlineExceeded}},
			{"idle", idleErr, []error{ErrTTFTTimeout, ErrStreamDeadlineExceeded}},
			{"deadline", deadlineErr, []error{ErrTTFTTimeout, ErrIdleTimeout}},
		}
		for _, c := range cases {
			for _, other := range c.not {
				if errors.Is(c.err, other) {
					t.Errorf("%s error %v must NOT match %v", c.name, c.err, other)
				}
			}
		}
	})
}

// ---------------------------------------------------------------------------
// T4 — leak-free, idempotent Close
// ---------------------------------------------------------------------------

func TestIdleReader_CloseStopsWatchdogAndIsIdempotent(t *testing.T) {
	const payload = "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\ndata: [DONE]\n\n"

	body := &countingBody{r: strings.NewReader(payload)}
	// Windows generous enough that nothing fires during the happy path.
	r := newIdleReader(body, func() { t.Error("cancel must not fire on the happy path") },
		idleReaderConfig{TTFT: 5 * time.Second, Idle: 5 * time.Second, Tick: 50 * time.Millisecond})

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("happy-path stream failed: %v", err)
	}
	if string(got) != payload {
		t.Fatalf("payload mismatch:\n got %q\nwant %q", got, payload)
	}

	t.Run("close_shuts_the_watchdog_down_within_100ms", func(t *testing.T) {
		if err := r.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		select {
		case <-r.watchdogDone():
		case <-time.After(100 * time.Millisecond):
			t.Fatal("watchdog goroutine still running 100ms after Close")
		}
	})

	t.Run("close_closes_the_underlying_body_exactly_once", func(t *testing.T) {
		if n := body.closes.Load(); n != 1 {
			t.Fatalf("underlying body Close count = %d, want 1", n)
		}
	})

	t.Run("second_close_is_a_no_op", func(t *testing.T) {
		if err := r.Close(); err != nil {
			t.Fatalf("second Close returned %v, want nil", err)
		}
		if n := body.closes.Load(); n != 1 {
			t.Fatalf("underlying body Close count after second Close = %d, want 1", n)
		}
	})
}

// ---------------------------------------------------------------------------
// T5 — hard-deadline resolver
// ---------------------------------------------------------------------------

func TestOllamaStreamHardDeadline(t *testing.T) {
	const key = "AILANG_OLLAMA_HTTP_TIMEOUT_SEC"

	tests := []struct {
		name    string
		set     bool
		value   string
		want    time.Duration
		wantErr error
	}{
		{name: "unset_defaults_to_3600s", set: false, want: 3600 * time.Second},
		{name: "7200_is_honoured", set: true, value: "7200", want: 7200 * time.Second},
		{name: "zero_is_rejected", set: true, value: "0", wantErr: ErrStreamDeadlineInvalid},
		{name: "negative_is_rejected", set: true, value: "-1", wantErr: ErrStreamDeadlineInvalid},
		{name: "garbage_is_rejected", set: true, value: "banana", wantErr: ErrStreamDeadlineInvalid},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// t.Setenv registers the restore; unsetting after it gives a truly
			// absent variable for the default row.
			t.Setenv(key, "placeholder")
			if tc.set {
				t.Setenv(key, tc.value)
			} else if err := os.Unsetenv(key); err != nil {
				t.Fatalf("unsetenv: %v", err)
			}

			got, err := ollamaStreamHardDeadline()
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("want error %v, got %v", tc.wantErr, err)
				}
				if got != 0 {
					t.Fatalf("want zero duration on rejection, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}

	// The legacy resolver keeps its own semantics; this file must not have
	// changed it. "0" means "disabled" there and "invalid" here, on purpose.
	t.Run("legacy_ollamaV1Timeout_still_treats_0_as_disabled", func(t *testing.T) {
		t.Setenv(key, "0")
		if d := ollamaV1Timeout(); d != 0 {
			t.Fatalf("ollamaV1Timeout() with %s=0 = %v, want 0 (disabled)", key, d)
		}
	})
}

func TestOllamaSoftWindowResolvers(t *testing.T) {
	t.Run("idle_defaults_to_120s", func(t *testing.T) {
		t.Setenv("AILANG_OLLAMA_IDLE_TIMEOUT_SEC", "")
		if got := ollamaIdleTimeout(); got != 120*time.Second {
			t.Fatalf("got %v, want 120s", got)
		}
	})
	t.Run("idle_override_is_honoured", func(t *testing.T) {
		t.Setenv("AILANG_OLLAMA_IDLE_TIMEOUT_SEC", "45")
		if got := ollamaIdleTimeout(); got != 45*time.Second {
			t.Fatalf("got %v, want 45s", got)
		}
	})
	t.Run("ttft_defaults_to_600s", func(t *testing.T) {
		t.Setenv("AILANG_OLLAMA_TTFT_TIMEOUT_SEC", "")
		if got := ollamaTTFTTimeout(); got != 600*time.Second {
			t.Fatalf("got %v, want 600s", got)
		}
	})
	t.Run("ttft_override_is_honoured", func(t *testing.T) {
		t.Setenv("AILANG_OLLAMA_TTFT_TIMEOUT_SEC", "900")
		if got := ollamaTTFTTimeout(); got != 900*time.Second {
			t.Fatalf("got %v, want 900s", got)
		}
	})
}

// ---------------------------------------------------------------------------
// T6 — transport construction
// ---------------------------------------------------------------------------

type stubRoundTripper struct {
	resp *http.Response
	err  error
}

func (s *stubRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return s.resp, s.err
}

func TestIdleTimeoutTransport(t *testing.T) {
	t.Run("wraps_the_response_body_in_the_watchdog_reader", func(t *testing.T) {
		body := &countingBody{r: strings.NewReader("payload")}
		rt := &idleTimeoutTransport{
			base: &stubRoundTripper{resp: &http.Response{StatusCode: 200, Body: body}},
			cfg:  idleReaderConfig{TTFT: 5 * time.Second, Idle: 5 * time.Second, Tick: time.Second},
		}
		req, err := http.NewRequest(http.MethodGet, "http://example.invalid/v1/chat/completions", nil)
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}
		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("RoundTrip: %v", err)
		}
		wrapped, ok := resp.Body.(*idleReader)
		if !ok {
			t.Fatalf("response body is %T, want *idleReader", resp.Body)
		}
		if err := wrapped.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		if n := body.closes.Load(); n != 1 {
			t.Fatalf("underlying body Close count = %d, want 1", n)
		}
	})

	t.Run("transport_errors_pass_through_unwrapped", func(t *testing.T) {
		want := errors.New("dial failed")
		rt := &idleTimeoutTransport{base: &stubRoundTripper{err: want}}
		req, err := http.NewRequest(http.MethodGet, "http://example.invalid/", nil)
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}
		if _, err := rt.RoundTrip(req); !errors.Is(err, want) {
			t.Fatalf("got %v, want %v", err, want)
		}
	})

	t.Run("base_transport_is_built_once_cloned_and_does_not_mutate_the_default", func(t *testing.T) {
		a := streamBaseTransport()
		b := streamBaseTransport()
		if a != b {
			t.Fatal("streamBaseTransport must return the same instance (sync.Once)")
		}
		def, ok := http.DefaultTransport.(*http.Transport)
		if !ok {
			t.Skip("http.DefaultTransport is not an *http.Transport in this build")
		}
		if a == def {
			t.Fatal("streamBaseTransport must be a clone, not http.DefaultTransport itself")
		}
		if def.ResponseHeaderTimeout != 0 {
			t.Fatalf("http.DefaultTransport was mutated: ResponseHeaderTimeout=%v", def.ResponseHeaderTimeout)
		}
		if a.ResponseHeaderTimeout <= 0 {
			t.Fatalf("clone must carry a TTFT ResponseHeaderTimeout, got %v", a.ResponseHeaderTimeout)
		}
		if a.IdleConnTimeout != def.IdleConnTimeout {
			t.Fatalf("clone lost the inherited IdleConnTimeout: got %v, want %v",
				a.IdleConnTimeout, def.IdleConnTimeout)
		}
		if a.IdleConnTimeout == 0 {
			t.Fatal("IdleConnTimeout is unlimited; idle connections would accumulate forever")
		}
	})

	t.Run("newIdleTimeoutTransport_delegates_to_the_shared_base", func(t *testing.T) {
		rt := newIdleTimeoutTransport(func() {}, idleReaderConfig{})
		if rt.base != nil {
			t.Fatalf("per-request wrapper must not carry its own transport, got %T", rt.base)
		}
	})
}
