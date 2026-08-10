package ollama

// streambranch_test.go — acceptance tests for M2 of
// M-OLLAMA-V1-STREAMING-IDLE-TIMEOUT (ailang#618).
//
// A separate file from step_test.go on purpose: AC-M2.1 requires that the
// existing bodies in step_test.go stay UNMODIFIED, and a new file makes that
// property hold by construction rather than by review.
//
// Window granularity note: the two soft windows are configured through env
// vars that take whole SECONDS, so the plan's "500ms idle / 900ms delay" shapes
// are realised here as "1s idle / 1.5s delay". The ORDERING the plan relies on
// (emission interval < idle window < delay < TTFT window < hard deadline) is
// preserved exactly; only the scale moves.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
)

// --- fake /v1 server --------------------------------------------------------

// fakeV1 is an Ollama-compatible /v1 endpoint under test control. It counts
// requests (AC-M2.3 needs "exactly zero requests left the process") and
// captures the raw request bytes + headers (AC-M2.1 needs the wire shape).
type fakeV1 struct {
	srv      *httptest.Server
	requests atomic.Int32
	stop     chan struct{}

	mu     sync.Mutex
	body   []byte
	header http.Header
	path   string
}

// newFakeV1 starts the server and registers cleanups so a handler that is
// deliberately stalling cannot wedge srv.Close(). Cleanups run LIFO, so the
// stop channel is closed BEFORE the server is closed.
func newFakeV1(t *testing.T, h func(f *fakeV1, w http.ResponseWriter, r *http.Request)) *fakeV1 {
	t.Helper()
	f := &fakeV1{stop: make(chan struct{})}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.requests.Add(1)
		b, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.body = b
		f.header = r.Header.Clone()
		f.path = r.URL.Path
		f.mu.Unlock()
		h(f, w, r)
	}))
	t.Cleanup(f.srv.Close)
	t.Cleanup(func() { close(f.stop) })
	return f
}

func (f *fakeV1) capturedBody() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return string(f.body)
}

func (f *fakeV1) capturedHeader(k string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.header == nil {
		return ""
	}
	return f.header.Get(k)
}

// sseHeaders writes the SSE response header and flushes it, so the response
// HEADERS are always on the wire immediately. Everything the tests delay is
// therefore a BODY delay, which is what the TTFT/idle windows measure.
func sseHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	flush(w)
}

func flush(w http.ResponseWriter) {
	if fl, ok := w.(http.Flusher); ok {
		fl.Flush()
	}
}

// sseWrite returns false once the client has gone away, so a "forever" handler
// terminates instead of spinning.
func sseWrite(w http.ResponseWriter, s string) bool {
	if _, err := io.WriteString(w, s); err != nil {
		return false
	}
	flush(w)
	return true
}

func sseData(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err) // test-only: the literals below are always marshalable
	}
	return "data: " + string(b) + "\n\n"
}

func contentChunk(text string) string {
	return sseData(map[string]any{
		"id": "chatcmpl-1", "object": "chat.completion.chunk", "model": "qwen3.6",
		"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"content": text}}},
	})
}

// toolFragChunk emits ONE tool_calls fragment and nothing else — no content,
// no reasoning. This is the shape that fires zero onChunk callbacks
// (streamstep.go has no tool-call callback site), which is exactly why the
// idle clock has to live at the READ level.
func toolFragChunk(name, args string) string {
	fn := map[string]any{}
	if name != "" {
		fn["name"] = name
	}
	if args != "" {
		fn["arguments"] = args
	}
	return sseData(map[string]any{
		"id": "chatcmpl-1", "object": "chat.completion.chunk", "model": "qwen3.6",
		"choices": []any{map[string]any{"index": 0, "delta": map[string]any{
			"tool_calls": []any{map[string]any{
				"index": 0, "id": "call_1", "type": "function", "function": fn,
			}},
		}}},
	})
}

func finishChunk(reason string) string {
	return sseData(map[string]any{
		"id": "chatcmpl-1", "object": "chat.completion.chunk", "model": "qwen3.6",
		"choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": reason}},
	})
}

const doneChunk = "data: [DONE]\n\n"

// --- harness ----------------------------------------------------------------

// streamEnv puts the process in "streaming /v1, no native tools" mode.
func streamEnv(t *testing.T, endpoint string) {
	t.Helper()
	t.Setenv("AILANG_OLLAMA_NATIVE_TOOLS", "")
	t.Setenv("AILANG_OLLAMA_V1_STREAM", "1")
	t.Setenv("OLLAMA_HOST", endpoint)
}

func toolReq() *ai.Request {
	return &ai.Request{
		Model:    "ollama:qwen3.6",
		Messages: []ai.Message{{Role: "user", Content: "write x.ail"}},
		Tools: []ai.ToolSchema{{
			Name: "write_file", Description: "write a file",
			Parameters: `{"type":"object","properties":{"path":{"type":"string"}}}`,
		}},
	}
}

func newTestClient(t *testing.T, endpoint string) *Client {
	t.Helper()
	c, err := NewClient(WithEndpoint(endpoint))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

type stepResult struct {
	resp *ai.Response
	err  error
}

// stepWithin runs Step on a goroutine and fails the test if it has not
// returned within limit. Every timeout AC needs this: the failure mode of a
// broken watchdog is an infinite block, and a bare synchronous call would
// surface that as a 10-minute package timeout rather than a named failure.
func stepWithin(t *testing.T, c *Client, req *ai.Request, limit time.Duration) stepResult {
	t.Helper()
	done := make(chan stepResult, 1)
	go func() {
		resp, err := c.Step(context.Background(), req)
		done <- stepResult{resp, err}
	}()
	select {
	case r := <-done:
		return r
	case <-time.After(limit):
		t.Fatalf("Step did not return within %v — the watchdog/deadline never fired", limit)
		return stepResult{}
	}
}

// --- AC-M2.1 — default-off is byte-identical (doc S5) ------------------------

// TestStreamBranch_DefaultOffIsByteIdentical pins the release-gating property:
// with AILANG_OLLAMA_V1_STREAM unset, the /v1 request is the buffered one, with
// no streaming markers anywhere on the wire.
//
// The flag_on_control subtest is the anti-vacuity arm: it proves the three
// assertions can SEE a streaming request, so their absence in the flag-off arm
// is a measurement rather than a broken instrument.
func TestStreamBranch_DefaultOffIsByteIdentical(t *testing.T) {
	t.Run("flag_unset_sends_a_buffered_request", func(t *testing.T) {
		f := newFakeV1(t, func(_ *fakeV1, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"chatcmpl-1","object":"chat.completion","model":"qwen3.6","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
		})
		t.Setenv("AILANG_OLLAMA_NATIVE_TOOLS", "")
		t.Setenv("AILANG_OLLAMA_V1_STREAM", "") // the default: unset
		t.Setenv("OLLAMA_HOST", f.srv.URL)

		c := newTestClient(t, f.srv.URL)
		resp, err := c.Step(context.Background(), toolReq())
		if err != nil {
			t.Fatalf("Step: %v", err)
		}
		if resp.Text != "ok" {
			t.Errorf("Text = %q, want %q", resp.Text, "ok")
		}
		body := f.capturedBody()
		if strings.Contains(body, `"stream":true`) {
			t.Errorf("flag-off request set stream:true; body = %s", body)
		}
		if strings.Contains(body, "stream_options") {
			t.Errorf("flag-off request carried stream_options; body = %s", body)
		}
		if got := f.capturedHeader("Accept"); got == "text/event-stream" {
			t.Errorf("flag-off request asked for SSE: Accept = %q", got)
		}
	})

	t.Run("flag_on_control_the_assertions_can_see_a_stream", func(t *testing.T) {
		f := newFakeV1(t, func(_ *fakeV1, w http.ResponseWriter, _ *http.Request) {
			sseHeaders(w)
			sseWrite(w, contentChunk("ok"))
			sseWrite(w, finishChunk("stop"))
			sseWrite(w, doneChunk)
		})
		streamEnv(t, f.srv.URL)
		c := newTestClient(t, f.srv.URL)
		if _, err := c.Step(context.Background(), toolReq()); err != nil {
			t.Fatalf("Step: %v", err)
		}
		body := f.capturedBody()
		if !strings.Contains(body, `"stream":true`) {
			t.Errorf("CONTROL FAILED: flag-on request has no stream:true, so the flag-off assertion proves nothing; body = %s", body)
		}
		if !strings.Contains(body, "stream_options") {
			t.Errorf("CONTROL FAILED: flag-on request has no stream_options; body = %s", body)
		}
		if got := f.capturedHeader("Accept"); got != "text/event-stream" {
			t.Errorf("CONTROL FAILED: flag-on Accept = %q, want text/event-stream", got)
		}
	})
}

// --- AC-M2.2 — the MANDATORY hard deadline (doc S8) --------------------------

// TestStreamBranch_HardDeadlineBeatsKeepAlive is the highest-risk requirement
// in the sprint: a server that emits keep-alive BYTES forever but never a
// parseable chunk defeats an idle-only guard completely, because every
// keep-alive resets the idle clock. Only a total budget terminates it.
//
// Interval (100ms) << idle window (1s) << hard deadline (2s): the idle timer is
// provably reset throughout and provably never fires, so the terminating error
// can only have come from the deadline. The !ErrIdleTimeout assertion pins
// which of the two racing clocks is reported.
func TestStreamBranch_HardDeadlineBeatsKeepAlive(t *testing.T) {
	f := newFakeV1(t, func(f *fakeV1, w http.ResponseWriter, r *http.Request) {
		sseHeaders(w)
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-f.stop:
				return
			case <-tick.C:
				if !sseWrite(w, ": keep-alive\n") {
					return
				}
			}
		}
	})
	streamEnv(t, f.srv.URL)
	t.Setenv("AILANG_OLLAMA_IDLE_TIMEOUT_SEC", "1")
	t.Setenv("AILANG_OLLAMA_HTTP_TIMEOUT_SEC", "2") // the hard deadline

	c := newTestClient(t, f.srv.URL)
	start := time.Now()
	got := stepWithin(t, c, toolReq(), 3*time.Second)
	elapsed := time.Since(start)

	if got.err == nil {
		t.Fatalf("expected a hard-deadline error from an infinite keep-alive stream, got resp=%+v", got.resp)
	}
	if !errors.Is(got.err, ErrStreamDeadlineExceeded) {
		t.Fatalf("err = %v, want errors.Is(err, ErrStreamDeadlineExceeded)", got.err)
	}
	if errors.Is(got.err, ErrIdleTimeout) {
		t.Errorf("err reported as idle timeout, but bytes flowed every 100ms under a 1s idle window: %v", got.err)
	}
	if elapsed < 1500*time.Millisecond {
		t.Errorf("returned after %v — too early for a 2s deadline; something other than the deadline fired", elapsed)
	}
}

// --- AC-M2.3 — a configured 0 is REJECTED, not "disabled" (doc S8) -----------

// TestStreamBranch_ZeroDeadlineIsRejectedBeforeAnyRequest pins the semantics
// split. For the legacy buffered resolver, AILANG_OLLAMA_HTTP_TIMEOUT_SEC=0
// means "no cap" and TestOllamaV1Timeout pins that. For the streaming path an
// uncapped stream is the hang the feature exists to prevent, so the same value
// is a configuration ERROR — and the request-counter assertion is what stops an
// implementation that merely fails "eventually" from passing.
func TestStreamBranch_ZeroDeadlineIsRejectedBeforeAnyRequest(t *testing.T) {
	for _, v := range []string{"0", "-1"} {
		t.Run("value_"+v, func(t *testing.T) {
			f := newFakeV1(t, func(_ *fakeV1, w http.ResponseWriter, _ *http.Request) {
				sseHeaders(w)
				sseWrite(w, contentChunk("should never be reached"))
				sseWrite(w, doneChunk)
			})
			streamEnv(t, f.srv.URL)
			t.Setenv("AILANG_OLLAMA_HTTP_TIMEOUT_SEC", v)

			c := newTestClient(t, f.srv.URL)
			_, err := c.Step(context.Background(), toolReq())
			if err == nil {
				t.Fatal("expected ErrStreamDeadlineInvalid, got nil")
			}
			if !errors.Is(err, ErrStreamDeadlineInvalid) {
				t.Fatalf("err = %v, want errors.Is(err, ErrStreamDeadlineInvalid)", err)
			}
			if n := f.requests.Load(); n != 0 {
				t.Fatalf("server saw %d requests, want 0 — the refusal must happen before anything leaves the process", n)
			}
		})
	}

	// The mirror case: the SAME value keeps its legacy meaning on the buffered
	// path, so this change is a split of semantics and not a redefinition.
	t.Run("flag_off_zero_still_means_uncapped_on_the_legacy_path", func(t *testing.T) {
		f := newFakeV1(t, func(_ *fakeV1, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"c","object":"chat.completion","model":"qwen3.6","choices":[{"index":0,"message":{"role":"assistant","content":"legacy"},"finish_reason":"stop"}]}`)
		})
		t.Setenv("AILANG_OLLAMA_NATIVE_TOOLS", "")
		t.Setenv("AILANG_OLLAMA_V1_STREAM", "")
		t.Setenv("OLLAMA_HOST", f.srv.URL)
		t.Setenv("AILANG_OLLAMA_HTTP_TIMEOUT_SEC", "0")

		if got := ollamaV1Timeout(); got != 0 {
			t.Fatalf("ollamaV1Timeout() = %v, want 0 (legacy 'disabled') — the legacy resolver must not change", got)
		}
		c := newTestClient(t, f.srv.URL)
		resp, err := c.Step(context.Background(), toolReq())
		if err != nil {
			t.Fatalf("legacy buffered path failed with the same value that the stream rejects: %v", err)
		}
		if resp.Text != "legacy" {
			t.Errorf("Text = %q, want %q", resp.Text, "legacy")
		}
	})
}

// --- AC-M2.4 — the outer 300s deadline must not survive (REFUTATION #1) ------

// readStreamMetrics returns the single stream_metrics record from the ollama
// debug log at path.
func readStreamMetrics(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read debug log: %v", err)
	}
	var found []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("debug log line is not JSON: %v (%s)", err, line)
		}
		if rec["kind"] == "stream_metrics" {
			found = append(found, rec)
		}
	}
	if len(found) != 1 {
		t.Fatalf("found %d stream_metrics records, want exactly 1 (log: %s)", len(found), raw)
	}
	return found[0]
}

// TestStreamBranch_EffectiveDeadlineIsNotTheOuter300s is the gate on
// REFUTATION #1. Step wraps the whole call in ollamaCallContext (300s default)
// BEFORE the /v1 branch is reached; a streaming branch derived from that
// context is capped at 300s no matter what the hard deadline says, and the
// feature ships inert with the rig reproducing today's ~4m59s signature.
//
// Why the deadline VALUE and not a wall clock: both clocks read the same env
// var, so whenever it is set they are equal and no elapsed-time test can tell
// them apart. They differ only in their DEFAULTS — 300s vs 3600s — so the env
// is left unset here and the discriminating observation is the effective
// deadline read back from streamCtx.Deadline().
func TestStreamBranch_EffectiveDeadlineIsNotTheOuter300s(t *testing.T) {
	f := newFakeV1(t, func(_ *fakeV1, w http.ResponseWriter, _ *http.Request) {
		sseHeaders(w)
		sseWrite(w, contentChunk("hi"))
		sseWrite(w, finishChunk("stop"))
		sseWrite(w, doneChunk)
	})
	streamEnv(t, f.srv.URL)
	t.Setenv("AILANG_OLLAMA_HTTP_TIMEOUT_SEC", "") // UNSET: the defaults diverge
	logPath := t.TempDir() + "/ollama.jsonl"
	t.Setenv("AILANG_OLLAMA_LOG_REQUESTS", logPath)

	c := newTestClient(t, f.srv.URL)
	if _, err := c.Step(context.Background(), toolReq()); err != nil {
		t.Fatalf("Step: %v", err)
	}

	rec := readStreamMetrics(t, logPath)
	eff, ok := rec["effective_deadline_sec"].(float64)
	if !ok {
		t.Fatalf("stream_metrics has no numeric effective_deadline_sec: %+v", rec)
	}
	if eff < 3500 {
		t.Fatalf("effective deadline = %.0fs, want >= 3500s. "+
			"~300s means the streaming branch inherited ollamaCallContext's wrap "+
			"instead of the pre-wrap context, and the feature is inert. (record: %+v)", eff, rec)
	}
	if hard, ok := rec["hard_deadline_sec"].(float64); !ok || hard != 3600 {
		t.Errorf("hard_deadline_sec = %v, want 3600 (the configured default)", rec["hard_deadline_sec"])
	}
	// The falsifier fields for Design Freeze #1/#2/#3 must all be present.
	for _, k := range []string{"ttft_ms", "max_gap_ms", "total_ms", "effective_deadline_sec"} {
		if _, ok := rec[k]; !ok {
			t.Errorf("stream_metrics is missing %q — AC-M4.2 reads this on the rig", k)
		}
	}
}

// --- AC-M2.5 — tool-call-only streams must not starve the clock (doc S1) -----

// TestStreamBranch_ToolCallOnlyStreamDoesNotStarve pins the property that makes
// read-level placement of the idle clock correct by construction: a stream of
// pure tool_calls fragments fires the onChunk callback ZERO times (streamstep.go
// has no tool-call callback site), so a callback-level clock would starve and
// kill a perfectly healthy turn.
//
// Fragments arrive every 300ms for ~3s under a 1s idle window and a 2s TTFT
// window; the call must COMPLETE with correctly assembled arguments.
func TestStreamBranch_ToolCallOnlyStreamDoesNotStarve(t *testing.T) {
	frags := []string{`{"pa`, `th":"`, `a.ail`, `"}`}
	f := newFakeV1(t, func(f *fakeV1, w http.ResponseWriter, r *http.Request) {
		sseHeaders(w)
		// 10 emissions at 300ms => ~3s of stream, all tool_calls, no content.
		send := func(s string) bool {
			select {
			case <-r.Context().Done():
				return false
			case <-f.stop:
				return false
			case <-time.After(300 * time.Millisecond):
				return sseWrite(w, s)
			}
		}
		if !send(toolFragChunk("write_file", "")) {
			return
		}
		for _, fr := range frags {
			if !send(toolFragChunk("", fr)) {
				return
			}
		}
		for i := 0; i < 5; i++ {
			if !send(toolFragChunk("", "")) { // keep-alive-shaped tool frags
				return
			}
		}
		if !send(finishChunk("tool_calls")) {
			return
		}
		sseWrite(w, doneChunk)
	})
	streamEnv(t, f.srv.URL)
	t.Setenv("AILANG_OLLAMA_IDLE_TIMEOUT_SEC", "1")
	t.Setenv("AILANG_OLLAMA_TTFT_TIMEOUT_SEC", "2")

	c := newTestClient(t, f.srv.URL)
	got := stepWithin(t, c, toolReq(), 8*time.Second)
	if got.err != nil {
		t.Fatalf("a progressing tool-call-only stream was killed: %v", got.err)
	}
	if len(got.resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls = %+v, want exactly 1", got.resp.ToolCalls)
	}
	tc := got.resp.ToolCalls[0]
	if tc.Name != "write_file" {
		t.Errorf("tool call name = %q, want write_file", tc.Name)
	}
	if want := strings.Join(frags, ""); tc.Arguments != want {
		t.Errorf("arguments = %q, want %q (fragments not concatenated)", tc.Arguments, want)
	}
	if got.resp.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want tool_calls", got.resp.FinishReason)
	}
}

// --- AC-M2.6 — idle window at branch level (doc S2) -------------------------

// TestStreamBranch_IdleTimeoutFiresMidStream is the failure this whole line of
// work exists to catch: bytes flow, then the model goes silent forever.
func TestStreamBranch_IdleTimeoutFiresMidStream(t *testing.T) {
	f := newFakeV1(t, func(f *fakeV1, w http.ResponseWriter, r *http.Request) {
		sseHeaders(w)
		for i := 0; i < 3; i++ {
			if !sseWrite(w, contentChunk(fmt.Sprintf("chunk%d ", i))) {
				return
			}
		}
		select { // then silence, forever
		case <-r.Context().Done():
		case <-f.stop:
		}
	})
	streamEnv(t, f.srv.URL)
	t.Setenv("AILANG_OLLAMA_IDLE_TIMEOUT_SEC", "1")

	c := newTestClient(t, f.srv.URL)
	got := stepWithin(t, c, toolReq(), 5*time.Second)
	if got.err == nil {
		t.Fatalf("expected ErrIdleTimeout from a stream that went silent, got resp=%+v", got.resp)
	}
	if !errors.Is(got.err, ErrIdleTimeout) {
		t.Fatalf("err = %v, want errors.Is(err, ErrIdleTimeout)", got.err)
	}
	if errors.Is(got.err, ErrTTFTTimeout) {
		t.Errorf("bytes had already flowed, so this is not a TTFT failure: %v", got.err)
	}
}

// --- AC-M2.7 — TTFT window at branch level (doc S3) -------------------------

// TestStreamBranch_TTFTWindowIsSeparateFromIdle pins that the two windows are
// genuinely two windows. A cold 35B load legitimately produces nothing for
// minutes; if the (short) idle window were armed from t=0, every cold start
// would be killed. The first subtest is red under collapsing the windows into
// one; the second is red if the TTFT window never fires at all.
func TestStreamBranch_TTFTWindowIsSeparateFromIdle(t *testing.T) {
	t.Run("slow_first_byte_within_ttft_completes", func(t *testing.T) {
		f := newFakeV1(t, func(f *fakeV1, w http.ResponseWriter, r *http.Request) {
			sseHeaders(w) // headers immediately; the DELAY is body-side
			select {
			case <-r.Context().Done():
				return
			case <-f.stop:
				return
			case <-time.After(1500 * time.Millisecond): // > idle 1s, < TTFT 3s
			}
			sseWrite(w, contentChunk("late"))
			sseWrite(w, finishChunk("stop"))
			sseWrite(w, doneChunk)
		})
		streamEnv(t, f.srv.URL)
		t.Setenv("AILANG_OLLAMA_IDLE_TIMEOUT_SEC", "1")
		t.Setenv("AILANG_OLLAMA_TTFT_TIMEOUT_SEC", "3")

		c := newTestClient(t, f.srv.URL)
		got := stepWithin(t, c, toolReq(), 6*time.Second)
		if got.err != nil {
			t.Fatalf("a 1.5s pre-first-byte wait under a 3s TTFT window was killed: %v", got.err)
		}
		if got.resp.Text != "late" {
			t.Errorf("Text = %q, want %q", got.resp.Text, "late")
		}
	})

	t.Run("first_byte_past_ttft_fires_the_ttft_sentinel", func(t *testing.T) {
		f := newFakeV1(t, func(f *fakeV1, w http.ResponseWriter, r *http.Request) {
			sseHeaders(w)
			select { // never send a body byte
			case <-r.Context().Done():
			case <-f.stop:
			}
		})
		streamEnv(t, f.srv.URL)
		t.Setenv("AILANG_OLLAMA_IDLE_TIMEOUT_SEC", "1")
		t.Setenv("AILANG_OLLAMA_TTFT_TIMEOUT_SEC", "2")

		c := newTestClient(t, f.srv.URL)
		got := stepWithin(t, c, toolReq(), 6*time.Second)
		if got.err == nil {
			t.Fatalf("expected ErrTTFTTimeout, got resp=%+v", got.resp)
		}
		if !errors.Is(got.err, ErrTTFTTimeout) {
			t.Fatalf("err = %v, want errors.Is(err, ErrTTFTTimeout)", got.err)
		}
		if errors.Is(got.err, ErrIdleTimeout) {
			t.Errorf("no byte ever arrived, so this is not an idle failure: %v", got.err)
		}
	})
}

// --- AC-M2.8 — long but progressing beats the old client timeout (doc S4) ----

// TestStreamBranch_LongProgressingStreamCompletes is the V12 trap in its pure
// form. A whole-call http.Client.Timeout cannot distinguish "still producing
// tokens" from "wedged": here the stream drips for 2.5s under a 4s budget and
// must COMPLETE. Any client-level timer shorter than the total stream duration
// interrupts the body read mid-stream and fails this.
func TestStreamBranch_LongProgressingStreamCompletes(t *testing.T) {
	const n = 12
	f := newFakeV1(t, func(f *fakeV1, w http.ResponseWriter, r *http.Request) {
		sseHeaders(w)
		for i := 0; i < n; i++ {
			select {
			case <-r.Context().Done():
				return
			case <-f.stop:
				return
			case <-time.After(200 * time.Millisecond):
			}
			if !sseWrite(w, contentChunk(fmt.Sprintf("t%d ", i))) {
				return
			}
		}
		sseWrite(w, finishChunk("stop"))
		sseWrite(w, doneChunk)
	})
	streamEnv(t, f.srv.URL)
	t.Setenv("AILANG_OLLAMA_HTTP_TIMEOUT_SEC", "4") // > the ~2.5s stream

	var want strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&want, "t%d ", i)
	}

	c := newTestClient(t, f.srv.URL)
	got := stepWithin(t, c, toolReq(), 8*time.Second)
	if got.err != nil {
		t.Fatalf("a 2.5s progressing stream under a 4s budget was killed: %v", got.err)
	}
	if got.resp.Text != want.String() {
		t.Errorf("Text = %q, want %q", got.resp.Text, want.String())
	}
}
