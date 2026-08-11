package ollama

// streamstep.go — M2 of M-OLLAMA-V1-STREAMING-IDLE-TIMEOUT (ailang#618).
//
// Split out of step.go rather than appended to it so the flag-off diff to
// Step is exactly two hunks (the outerCtx capture and one early return), and
// so M3's response-parity work has room without crossing the 800-line gate.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/openai"
)

// --- Streaming /v1 path (M-OLLAMA-V1-STREAMING-IDLE-TIMEOUT, ailang#618 M2) ---
//
// The buffered /v1 path above reads the whole response with io.ReadAll under a
// single whole-call clock. That clock cannot tell "the model is thinking hard"
// from "the connection is wedged", so it is either too short (a long-but-healthy
// turn is killed at 4m59.97s) or too long (a wedged stream hangs for hours).
//
// The streaming path replaces one coarse clock with three sharp ones:
// TTFT (silence before the first byte), IDLE (silence between bytes) and a
// MANDATORY hard deadline (total budget). The first two live in the response
// body wrapper (idlereader.go); the third is the request context.

// ollamaStreamEnabled reports whether the flag-gated streaming /v1 path is on.
//
// Exactly "1" opts in. Deliberately NOT `!= "0"`: default-off is a contract
// (doc S5 — with the flag unset the wire bytes must be identical to today's),
// so this must be an opt-IN test, not an opt-out one.
func ollamaStreamEnabled() bool {
	return os.Getenv("AILANG_OLLAMA_V1_STREAM") == "1"
}

// streamCallContext derives the streaming call's context from outer — the
// context as it was BEFORE Step's ollamaCallContext wrap — and returns the
// resolved hard deadline alongside it.
//
// Deriving from outer is the entire point of Step's outerCtx capture.
// ollamaCallContext bounds the buffered paths at AILANG_OLLAMA_HTTP_TIMEOUT_SEC
// with a 300s default. A streaming branch derived from THAT context would be
// capped at 300s no matter how large the new hard deadline was configured to
// be: the effective bound is the minimum of the two, so the feature would ship
// inert and every long turn would still die at ~4m59s — the exact signature
// this work exists to remove.
//
// The hard deadline is MANDATORY. A <= 0 or unparseable configuration is
// REJECTED here, before any HTTP request is made, rather than silently meaning
// "unbounded" (which is what the same value means to the legacy resolver
// ollamaV1Timeout, whose semantics TestOllamaV1Timeout pins and which this
// must not change). An unbounded stream is the hang the guard exists to stop.
func streamCallContext(outer context.Context) (context.Context, context.CancelFunc, time.Duration, error) {
	hard, err := ollamaStreamHardDeadline()
	if err != nil {
		return nil, nil, 0, err
	}
	base, stopHard := outer, context.CancelFunc(func() {})
	if hard > 0 {
		base, stopHard = context.WithTimeout(outer, hard)
	}
	// A cancellable child of the hard-deadline context so the TTFT/idle
	// watchdog can fail a pending Read immediately instead of waiting out the
	// total budget. Cancelling the child leaves the parent's deadline intact,
	// so streamCtx.Deadline() still reads back the hard deadline.
	ctx, cancel := context.WithCancel(base)
	return ctx, func() { cancel(); stopHard() }, hard, nil
}

// streamMetrics is the per-request falsifier record for Design Freeze #1/#2/#3:
// the observed TTFT and max inter-chunk gap are what say whether the 600s/120s
// window defaults are right, and the read-back effective deadline is what says
// whether the hard deadline actually reached the wire.
//
// Read-level timings (ttft, maxGap, bytes) are recorded by meteredReader;
// delta counts come from the StreamStep callback. In M2 the callback only
// counts — M3 gives it the response-parity job.
type streamMetrics struct {
	mu             sync.Mutex
	start          time.Time
	ttft           time.Duration
	lastRead       time.Time
	maxGap         time.Duration
	bytes          int64
	sawFirst       bool
	contentDeltas  int
	thinkingDeltas int
}

// observe records a non-empty read: TTFT on the first one, the running maximum
// inter-read gap thereafter.
func (m *streamMetrics) observe(n int) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bytes += int64(n)
	if !m.sawFirst {
		m.sawFirst = true
		m.ttft = now.Sub(m.start)
	} else if gap := now.Sub(m.lastRead); gap > m.maxGap {
		m.maxGap = gap
	}
	m.lastRead = now
}

// onChunk is the StreamStep callback. Non-nil on purpose: it is the seam M3
// uses to recover Response.Reasoning without touching the shared SSE parser.
// In M2 it only counts deltas for the debug log.
func (m *streamMetrics) onChunk(chunk ai.StreamChunk) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch chunk.(type) {
	case ai.StreamContentDelta:
		m.contentDeltas++
	case ai.StreamThinkingDelta:
		m.thinkingDeltas++
	}
}

// meteredReader times reads without touching the watchdog wrapper it sits on
// top of. Placed OUTSIDE the idleReader so it observes exactly the reads the
// SSE scanner performs, and so idlereader.go stays a pure M1 artifact.
type meteredReader struct {
	inner io.ReadCloser
	m     *streamMetrics
}

var _ io.ReadCloser = (*meteredReader)(nil)

func (r *meteredReader) Read(p []byte) (int, error) {
	n, err := r.inner.Read(p)
	if n > 0 {
		r.m.observe(n)
	}
	return n, err
}

func (r *meteredReader) Close() error { return r.inner.Close() }

// streamWatchTransport wraps the per-request idleTimeoutTransport so the caller
// can ask, after the fact, WHICH window fired.
//
// It is needed because the typed sentinel the idleReader returns from Read is
// consumed by bufio.Scanner inside ParseChatStepSSEStream and re-wrapped as an
// *ai.AIError, which errors.Is cannot see through. Holding the reader lets the
// error mapping below reconstruct the typed sentinel from the wrapper's own
// first-writer-wins CAS rather than sniffing error strings.
type streamWatchTransport struct {
	inner http.RoundTripper
	m     *streamMetrics

	mu     sync.Mutex
	reader *idleReader
}

var _ http.RoundTripper = (*streamWatchTransport)(nil)

func (t *streamWatchTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.inner.RoundTrip(req)
	if resp == nil || resp.Body == nil {
		return resp, err
	}
	if ir, ok := resp.Body.(*idleReader); ok {
		t.mu.Lock()
		t.reader = ir
		t.mu.Unlock()
	}
	resp.Body = &meteredReader{inner: resp.Body, m: t.m}
	return resp, err
}

// firedError returns the typed sentinel for the window that fired, or nil if
// none did. cause is kept as context so the transport-level error is not lost.
func (t *streamWatchTransport) firedError(cause error) error {
	t.mu.Lock()
	r := t.reader
	t.mu.Unlock()
	if r == nil {
		return nil
	}
	code := r.fired.Load()
	if code == firedNone {
		return nil
	}
	return r.sentinel(code, cause)
}

// stepV1Stream is the streaming twin of Step's buffered /v1 delegation.
//
// outerCtx is Step's PRE-ollamaCallContext context; see streamCallContext.
func (c *Client) stepV1Stream(outerCtx context.Context, req *ai.Request) (*ai.Response, error) {
	streamCtx, cancel, hard, cfgErr := streamCallContext(outerCtx)
	if cfgErr != nil {
		// Refuse before anything leaves the process. Returning the error here
		// rather than after a request is what makes "0 is rejected, not
		// disabled" observable: the server sees zero requests.
		return nil, ai.NewProviderError("ollama", 0, cfgErr.Error(), cfgErr)
	}
	defer cancel()

	metrics := &streamMetrics{start: time.Now()}
	// cancel is the watchdog's escape hatch: a blocked Read cannot interrupt
	// itself, so the watchdog cancels the request context and the transport
	// fails the pending Read.
	//
	// Hard is deliberately left at 0 here: the total budget is enforced ONCE,
	// by streamCtx. Arming the reader's hard window with the same value would
	// be a second timer for the same budget, and a redundant guard is one that
	// no test can hold accountable — neutering either half would leave every
	// deadline test green. The typed ErrStreamDeadlineExceeded still reaches
	// the caller, via rule (b) in streamStepError.
	watch := &streamWatchTransport{
		m: metrics,
		inner: newIdleTimeoutTransport(cancel, idleReaderConfig{
			TTFT: ollamaTTFTTimeout(),
			Idle: ollamaIdleTimeout(),
		}),
	}

	v1 := openai.NewClient("ollama",
		openai.WithBaseURL(strings.TrimRight(c.endpoint, "/")+"/v1"),
		// Timeout: 0 on purpose. A whole-call client timeout is exactly the
		// trap this path removes — it fires mid-stream on a long but healthily
		// progressing response, which is indistinguishable to the caller from
		// a wedge. The bound is streamCtx's hard deadline plus the per-read
		// TTFT/idle watchdogs in the transport.
		openai.WithHTTPClient(&http.Client{Timeout: 0, Transport: watch}),
	)

	// Same request shaping as the buffered branch. Deliberately duplicated
	// rather than factored out: the flag-off diff to Step must stay reviewable
	// as "one early return", with no edits to the code it guards.
	r2 := *req
	r2.Model = bareModel(req.Model)
	r2.Temperature = resolveOllamaTemperature(req.Temperature)
	r2.MaxTokens = resolveOllamaMaxTokens(req.MaxTokens)

	resp, err := v1.StreamStep(streamCtx, &r2, metrics.onChunk)
	logOllamaStreamMetrics(metrics, streamCtx, hard, err)
	if err != nil {
		return nil, streamStepError(err, watch, streamCtx)
	}
	logOllamaResponse(resp)
	return resp, nil
}

// streamStepError maps a StreamStep failure to a typed error with EXPLICIT
// precedence. Precedence matters because when the hard deadline expires the
// idle timer is usually racing it, and a caller asking "was this idle?" must
// not get a yes from a deadline.
//
//	(a) a window fired  -> that window's sentinel (first-writer-wins CAS)
//	(b) the context hit its deadline -> ErrStreamDeadlineExceeded
//	(c) anything else   -> StreamStep's own classification, unchanged
//
// The typed sentinel is carried by an *ai.ProviderError so errors.Is still
// finds it (ai.AIError cannot wrap a sentinel from outside its own package),
// and the message embeds the sentinel text so string-based retry classifiers
// downstream keep seeing "timeout".
func streamStepError(err error, watch *streamWatchTransport, streamCtx context.Context) error {
	if fired := watch.firedError(err); fired != nil {
		return ai.NewProviderError("ollama", 0, fired.Error(), fired)
	}
	if errors.Is(streamCtx.Err(), context.DeadlineExceeded) {
		hardErr := fmt.Errorf("%w (transport: %v)", ErrStreamDeadlineExceeded, err)
		return ai.NewProviderError("ollama", 0, hardErr.Error(), hardErr)
	}
	return err
}

// logOllamaStreamMetrics appends the per-request stream record to the same
// JSONL sink as logOllamaRequest/logOllamaResponse (AILANG_OLLAMA_LOG_REQUESTS,
// or the HOME sentinel that reaches harnesses which drop our env).
//
// effective_deadline_sec is READ BACK from streamCtx.Deadline() rather than
// reported from the configured value. That distinction is the whole point: a
// configured 3600s that arrives on the wire as 300s because an outer context
// deadline survived is precisely the inert-feature bug, and only the read-back
// value can see it. Compare it with hard_deadline_sec: if they disagree,
// something upstream is capping the stream.
func logOllamaStreamMetrics(m *streamMetrics, streamCtx context.Context, hard time.Duration, err error) {
	m.mu.Lock()
	rec := map[string]any{
		"kind":              "stream_metrics",
		"ts":                time.Now().UTC().Format(time.RFC3339Nano),
		"ttft_ms":           m.ttft.Milliseconds(),
		"max_gap_ms":        m.maxGap.Milliseconds(),
		"total_ms":          time.Since(m.start).Milliseconds(),
		"bytes":             m.bytes,
		"content_deltas":    m.contentDeltas,
		"thinking_deltas":   m.thinkingDeltas,
		"hard_deadline_sec": hard.Seconds(),
		"idle_window_sec":   ollamaIdleTimeout().Seconds(),
		"ttft_window_sec":   ollamaTTFTTimeout().Seconds(),
		"saw_first_byte":    m.sawFirst,
		"effective_deadline_sec": func() float64 {
			dl, ok := streamCtx.Deadline()
			if !ok {
				return 0
			}
			return dl.Sub(m.start).Seconds()
		}(),
	}
	m.mu.Unlock()
	if err != nil {
		rec["error"] = err.Error()
	}
	appendOllamaLog(rec)
}
