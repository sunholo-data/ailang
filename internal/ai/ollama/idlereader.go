package ollama

// idlereader.go — M1 of M-OLLAMA-V1-STREAMING-IDLE-TIMEOUT (ailang#618).
//
// A streaming /v1 response can stall in three distinct ways, and the three
// need distinct diagnoses:
//
//	TTFT   — the model never emitted a first byte (cold load, GPU contention).
//	IDLE   — bytes flowed, then stopped mid-stream (the S2 hang we actually see).
//	HARD   — the stream is alive but has run past its total budget.
//
// A blocked Read cannot interrupt itself, so a watchdog goroutine samples the
// last-activity timestamp and, on expiry, calls a supplied context.CancelFunc;
// the transport then fails the pending Read. The watchdog records WHICH window
// fired via an atomic CAS (first writer wins) so the error surfaced to the
// caller is a typed sentinel, never a generic "context canceled".
//
// M1 adds no call sites — nothing in this file is wired into Step yet, so
// runtime behaviour is unchanged by construction. M2 does the wiring.

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Typed sentinels. All four are distinguishable with errors.Is, which is what
// lets the caller tell "model never started" from "model went quiet mid-stream"
// from "we blew the total budget".
var (
	// ErrTTFTTimeout — no first byte arrived within the TTFT window.
	ErrTTFTTimeout = errors.New("ollama stream: time-to-first-token timeout")
	// ErrIdleTimeout — bytes flowed, then the stream went silent past the idle window.
	ErrIdleTimeout = errors.New("ollama stream: idle timeout")
	// ErrStreamDeadlineExceeded — the stream ran past its total hard deadline.
	ErrStreamDeadlineExceeded = errors.New("ollama stream: hard deadline exceeded")
	// ErrStreamDeadlineInvalid — the configured hard deadline is unusable.
	ErrStreamDeadlineInvalid = errors.New("ollama stream: invalid hard deadline configuration")
)

// Window defaults, in seconds.
const (
	// defaultOllamaIdleTimeoutSec bounds silence *between* bytes. Short, because
	// a healthy stream emits tokens continuously.
	defaultOllamaIdleTimeoutSec = 120
	// defaultOllamaTTFTTimeoutSec bounds silence *before* the first byte. Long,
	// because a cold 35B load under GPU contention legitimately takes minutes.
	defaultOllamaTTFTTimeoutSec = 600
	// defaultOllamaStreamDeadlineSec bounds the whole stream.
	defaultOllamaStreamDeadlineSec = 3600
)

// ollamaIdleTimeout resolves the inter-byte silence window.
// Override with AILANG_OLLAMA_IDLE_TIMEOUT_SEC (a positive integer, seconds).
func ollamaIdleTimeout() time.Duration {
	return envSeconds("AILANG_OLLAMA_IDLE_TIMEOUT_SEC", defaultOllamaIdleTimeoutSec)
}

// ollamaTTFTTimeout resolves the pre-first-byte window.
// Override with AILANG_OLLAMA_TTFT_TIMEOUT_SEC (a positive integer, seconds).
func ollamaTTFTTimeout() time.Duration {
	return envSeconds("AILANG_OLLAMA_TTFT_TIMEOUT_SEC", defaultOllamaTTFTTimeoutSec)
}

// envSeconds reads a positive-integer seconds value, falling back to def when
// unset or unusable. Only used for the two *soft* windows, where a bad value
// costs nothing but a wrong watchdog period.
func envSeconds(key string, defSec int) time.Duration {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return time.Duration(defSec) * time.Second
}

// ollamaStreamHardDeadline resolves the total-stream budget for the STREAMING
// path from AILANG_OLLAMA_HTTP_TIMEOUT_SEC, defaulting to 3600s.
//
// Deliberately NOT ollamaV1Timeout(). That resolver maps "0" (and negatives) to
// "no timeout at all" for the legacy buffered path, and TestOllamaV1Timeout pins
// that behaviour. An unbounded *stream* is precisely the failure mode this work
// exists to remove, so here <= 0 is rejected rather than silently honoured — no
// silent fallback on a value that decides whether a rig job can hang forever.
func ollamaStreamHardDeadline() (time.Duration, error) {
	v := os.Getenv("AILANG_OLLAMA_HTTP_TIMEOUT_SEC")
	if v == "" {
		return defaultOllamaStreamDeadlineSec * time.Second, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%w: AILANG_OLLAMA_HTTP_TIMEOUT_SEC=%q is not an integer", ErrStreamDeadlineInvalid, v)
	}
	if n <= 0 {
		return 0, fmt.Errorf("%w: AILANG_OLLAMA_HTTP_TIMEOUT_SEC=%d must be > 0 in streaming mode "+
			"(an unbounded stream is the hang this guard exists to prevent)", ErrStreamDeadlineInvalid, n)
	}
	return time.Duration(n) * time.Second, nil
}

// Which window fired. Stored in an atomic int32 so the watchdog and the reader
// agree without a lock on the hot Read path.
const (
	firedNone int32 = iota
	firedTTFT
	firedIdle
	firedDeadline
)

// idleReaderConfig carries the three windows. Tests construct this directly at
// millisecond scale rather than going through the env resolvers.
type idleReaderConfig struct {
	// TTFT bounds the wait for the first byte. Zero disables the TTFT window.
	TTFT time.Duration
	// Idle bounds silence after at least one byte. Zero disables the idle window.
	Idle time.Duration
	// Hard bounds the whole stream. Zero disables the hard deadline.
	Hard time.Duration
	// Tick is the watchdog sampling period. Zero derives one from the windows.
	Tick time.Duration
}

// idleReader wraps a streaming response body with TTFT/idle/hard watchdogs.
//
// It is an io.ReadCloser, not an io.Reader, on purpose: StreamStep does
// `defer httpResp.Body.Close()` (internal/ai/openai/streamstep.go:100), and that
// Close is what shuts the watchdog down. A bare Reader would lose the underlying
// Close and leave the timer armed after the request finished.
type idleReader struct {
	body   io.ReadCloser
	cancel func()
	cfg    idleReaderConfig
	hardAt time.Time

	mu       sync.Mutex
	last     time.Time
	sawBytes bool

	fired   atomic.Int32
	stopped atomic.Bool
	stop    chan struct{}
	done    chan struct{}
}

var _ io.ReadCloser = (*idleReader)(nil)

// newIdleReader wraps body and starts the watchdog. cancel is invoked once, from
// the watchdog goroutine, when a window expires; in production it is the request
// context's CancelFunc, which makes the transport fail the pending Read.
func newIdleReader(body io.ReadCloser, cancel func(), cfg idleReaderConfig) *idleReader {
	now := time.Now()
	r := &idleReader{
		body:   body,
		cancel: cancel,
		cfg:    cfg,
		last:   now,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	if cfg.Hard > 0 {
		r.hardAt = now.Add(cfg.Hard)
	}
	go r.watch(watchdogTick(cfg))
	return r
}

// watchdogTick picks a sampling period roughly a tenth of the tightest window,
// clamped so tests at millisecond scale stay responsive and production does not
// spin.
func watchdogTick(cfg idleReaderConfig) time.Duration {
	if cfg.Tick > 0 {
		return cfg.Tick
	}
	tightest := time.Duration(0)
	for _, w := range []time.Duration{cfg.TTFT, cfg.Idle, cfg.Hard} {
		if w > 0 && (tightest == 0 || w < tightest) {
			tightest = w
		}
	}
	if tightest == 0 {
		return time.Second
	}
	tick := tightest / 10
	if tick < time.Millisecond {
		tick = time.Millisecond
	}
	if tick > 250*time.Millisecond {
		tick = 250 * time.Millisecond
	}
	return tick
}

// watchdogDone is closed when the watchdog goroutine has exited. Tests assert
// shutdown against it; nothing in production needs to wait on it.
func (r *idleReader) watchdogDone() <-chan struct{} { return r.done }

// watch samples last-activity until a window expires or Close stops it.
func (r *idleReader) watch(tick time.Duration) {
	defer close(r.done)

	t := time.NewTicker(tick)
	defer t.Stop()

	for {
		select {
		case <-r.stop:
			return
		case now := <-t.C:
			r.mu.Lock()
			firstRead := !r.sawBytes
			last := r.last
			r.mu.Unlock()

			elapsed := now.Sub(last)

			if !r.hardAt.IsZero() && now.After(r.hardAt) {
				r.fire(firedDeadline)
				return
			}
			if firstRead && r.cfg.TTFT > 0 && elapsed > r.cfg.TTFT {
				r.fire(firedTTFT)
				return
			}
			if !firstRead && r.cfg.Idle > 0 && elapsed > r.cfg.Idle {
				r.fire(firedIdle)
				return
			}
		}
	}
}

// fire records which window expired (first writer wins) and unblocks the reader.
func (r *idleReader) fire(code int32) {
	r.fired.CompareAndSwap(firedNone, code)
	if r.cancel != nil {
		r.cancel()
	}
}

// Read proxies the underlying body, refreshing the activity clock on every
// non-empty read. Once a window has fired, the error is translated to the
// matching typed sentinel so the caller never sees a bare "context canceled".
func (r *idleReader) Read(p []byte) (int, error) {
	if code := r.fired.Load(); code != firedNone {
		return 0, r.sentinel(code, nil)
	}

	n, err := r.body.Read(p)
	if n > 0 {
		r.mu.Lock()
		r.sawBytes = true
		r.last = time.Now()
		r.mu.Unlock()
	}
	if err != nil {
		if code := r.fired.Load(); code != firedNone {
			return n, r.sentinel(code, err)
		}
		return n, err
	}
	return n, nil
}

// sentinel maps a fired-window code to its typed error, keeping the transport's
// own error as context.
func (r *idleReader) sentinel(code int32, cause error) error {
	var s error
	switch code {
	case firedTTFT:
		s = ErrTTFTTimeout
	case firedIdle:
		s = ErrIdleTimeout
	case firedDeadline:
		s = ErrStreamDeadlineExceeded
	default:
		return cause
	}
	if cause == nil {
		return s
	}
	// %v, not %w, for the cause: the sentinel must be the only thing errors.Is
	// matches, so a caller asking "was this idle?" cannot get a yes from TTFT.
	return fmt.Errorf("%w (transport: %v)", s, cause)
}

// Close stops the watchdog first, then closes the underlying body. It is
// idempotent: a second Close does not close the body again.
func (r *idleReader) Close() error {
	if r.stopped.CompareAndSwap(false, true) {
		close(r.stop)
		return r.body.Close()
	}
	return nil
}

// --- RoundTripper -----------------------------------------------------------

// The base transport is built ONCE, cloned from http.DefaultTransport.
//
// Two things it must not do: mutate http.DefaultTransport (process-global), or
// build a fresh &http.Transport{} per call. The /v1 client is constructed per
// request (step.go:293), and a bare http.Transport has IdleConnTimeout: 0 —
// unlimited — so a per-call transport would accumulate idle connections forever
// across thousands of rig requests. Cloning inherits IdleConnTimeout: 90s and
// the proxy configuration; only ResponseHeaderTimeout is set on the clone.
var (
	streamTransportOnce sync.Once
	streamTransportBase *http.Transport
)

func streamBaseTransport() *http.Transport {
	streamTransportOnce.Do(func() {
		if base, ok := http.DefaultTransport.(*http.Transport); ok {
			streamTransportBase = base.Clone()
		} else {
			streamTransportBase = &http.Transport{IdleConnTimeout: 90 * time.Second}
		}
		streamTransportBase.ResponseHeaderTimeout = ollamaTTFTTimeout()
	})
	return streamTransportBase
}

// idleTimeoutTransport is the cheap per-request wrapper: it holds this request's
// cancel func and windows, and delegates the actual round trip to the shared
// base transport.
type idleTimeoutTransport struct {
	// base is the delegate; nil means the shared package-level base transport.
	base   http.RoundTripper
	cancel func()
	cfg    idleReaderConfig
}

var _ http.RoundTripper = (*idleTimeoutTransport)(nil)

// newIdleTimeoutTransport builds the per-request wrapper over the shared base.
func newIdleTimeoutTransport(cancel func(), cfg idleReaderConfig) *idleTimeoutTransport {
	return &idleTimeoutTransport{cancel: cancel, cfg: cfg}
}

// RoundTrip performs the request and wraps the response body in the watchdog.
func (t *idleTimeoutTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = streamBaseTransport()
	}
	resp, err := base.RoundTrip(req)
	if err != nil || resp == nil || resp.Body == nil {
		return resp, err
	}
	resp.Body = newIdleReader(resp.Body, t.cancel, t.cfg)
	return resp, nil
}
