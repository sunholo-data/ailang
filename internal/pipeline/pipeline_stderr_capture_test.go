package pipeline

// Regression tests for the capturePipelineStderr teardown ordering. These
// inject a channel-gated reader so the fix is pinned structurally (M1 of the
// sprint), not by timing: the helper's drain-wait hook (opts.release) is what
// lets the injected reader perform its first underlying Read, and that hook
// runs only AFTER the write end is closed. Under the old ordering (MUT-1: the
// read end closed before the hook) the reader's first Read lands on a closed
// fd, io.Copy returns os.ErrClosed, and the helper panics with its explicit
// "copier failed" diagnostic — deterministically red regardless of goroutine
// scheduling.
//
// This file is test-only and does not exercise the compiler pipeline at all:
// it drives the helper directly. The tests are sequential (they swap the
// global os.Stderr) and must never be parallelized.

import (
	"errors"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// The four fixed V9 cold-drain lines, in order, exactly as the pipeline
// writes them. This is the payload every iteration of GatedReaderLosesNothing
// asserts against.
const coldPayload = "[CACHE] std/option: MISS\n[CACHE] std/result: MISS\n[CACHE] answer: MISS\n[CACHE] Summary: 0 hits, 3 misses (3 modules cached)\n"

// gatedReader wraps an io.Reader and parks every Read on a gate channel before
// its first underlying Read, so nothing can be consumed until the gate is
// closed (the helper's drain-wait hook does the closing in the tests). reads
// counts how many times Read has been invoked — the seam that kills MUT-2a.
// The underlying reader must be the pipe's read end `r` itself: the wrap seam
// installs it, so that under MUT-1 (r.Close() hoisted above the release hook)
// the first underlying Read lands on an already-closed *os.File and io.Copy
// returns os.ErrClosed deterministically, rather than racing the drain.
type gatedReader struct {
	rd    io.Reader // set by the wrap seam to the pipe's read end
	gate  chan struct{}
	reads atomic.Int64
}

func newGatedReader() *gatedReader {
	return &gatedReader{gate: make(chan struct{})}
}

// setUnderlying installs the reader this gated reader delegates to (the pipe's
// read end). Called exactly once, inside the wrap seam, before any Read.
func (g *gatedReader) setUnderlying(r io.Reader) { g.rd = r }

func (g *gatedReader) Read(p []byte) (int, error) {
	<-g.gate
	g.reads.Add(1)
	return g.rd.Read(p)
}

// release opens the gate. It is safe to call more than once; subsequent calls
// are no-ops.
func (g *gatedReader) release() {
	select {
	case <-g.gate:
	default:
		close(g.gate)
	}
}

// TestCapturePipelineStderr_GatedReaderLosesNothing runs N iterations, each
// with a fresh gated reader released by the helper's drain-wait hook. Under
// the fixed ordering the capture must equal the payload exactly (writer →
// wait → reader), and the wrapper's Read must actually have been invoked
// (reads ≥ 1 — kills MUT-2a). The test sets drainTimeout=2s so that MUT-2b
// (release hook ignored / gate never opened) fails in seconds rather than
// waiting out the full 10s default.
func TestCapturePipelineStderr_GatedReaderLosesNothing(t *testing.T) {
	const n = 50
	payload := coldPayload
	if len(payload) >= 4096 {
		t.Fatal("payload exceeds the pipe-capacity bound; keep it well below 4 KiB so writes never block while the reader is parked")
	}
	for i := 0; i < n; i++ {
		gr := newGatedReader()
		got := capturePipelineStderrWith(&captureOpts{
			wrap: func(r io.Reader) io.Reader {
				gr.setUnderlying(r) // delegate to the pipe's read end itself
				return gr
			},
			release:      gr.release, // opens the gate; runs after w.Close()
			drainTimeout: 2 * time.Second,
		}, func() {
			_, _ = io.WriteString(os.Stderr, payload)
		})
		if got != payload {
			t.Fatalf("iteration %d: reads=%d got=%q want %q", i, gr.reads.Load(), got, payload)
		}
		if gr.reads.Load() < 1 {
			t.Fatalf("iteration %d: reader wrapper never invoked (reads=%d); the seam is broken", i, gr.reads.Load())
		}
	}
	if got := os.Stderr; got == nil {
		t.Fatal("os.Stderr was not restored after the loop")
	}
}

// TestCapturePipelineStderr_PanicRestoresStderr drives a f() that panics and
// asserts the deferred teardown still runs: os.Stderr is restored and the
// write end is closed (kills MUT-3a, where the teardown is inline and skipped
// by the panic).
func TestCapturePipelineStderr_PanicRestoresStderr(t *testing.T) {
	original := os.Stderr
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected f() to panic")
		}
		if os.Stderr != original {
			t.Fatal("os.Stderr was not restored after a panicking f()")
		}
	}()
	capturePipelineStderr(func() {
		panic("boom in f")
	})
	// If we get here without a recover above, os.Stderr is restored and the panic propagated.
}

// TestCapturePipelineStderr_DrainTimeoutIsLoud injects a reader whose gate is
// never released, so the copier can never finish; the helper must time out,
// panic with a diagnostic naming the drain timeout, and still have restored
// os.Stderr (kills MUT-3b, where the bare receive parks forever and only the
// go-test timeout catches it).
func TestCapturePipelineStderr_DrainTimeoutIsLoud(t *testing.T) {
	original := os.Stderr
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected the helper to panic on drain timeout")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "did not drain") {
			t.Fatalf("panic did not name the drain timeout: %v", r)
		}
		if os.Stderr != original {
			t.Fatal("os.Stderr was not restored after the drain-timeout panic")
		}
	}()
	gr := newGatedReader() // gate stays closed forever; the wrapper parks on it and never reads
	capturePipelineStderrWith(&captureOpts{
		wrap: func(r io.Reader) io.Reader {
			gr.setUnderlying(r)
			return gr
		},
		drainTimeout: 50 * time.Millisecond,
	}, func() {
		_, _ = io.WriteString(os.Stderr, coldPayload)
	})
}

// TestCapturePipelineStderr_CopyErrorIsLoud injects a reader that yields a
// prefix of the payload and then a sentinel error. The helper must panic with
// a diagnostic containing that sentinel (propagating the io.Copy error), never
// return a silently-truncated capture (kills MUT-4).
func TestCapturePipelineStderr_CopyErrorIsLoud(t *testing.T) {
	original := os.Stderr
	sentinel := errors.New("sentinel read failure")
	// A reader that yields a prefix of the payload and then the sentinel error:
	// io.Copy stops at the sentinel, so the drain is incomplete and must panic.
	prefixReader := io.MultiReader(strings.NewReader(coldPayload[:len(coldPayload)-10]), failingReader{err: sentinel})
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected the helper to panic on a non-nil copy error")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, sentinel.Error()) {
			t.Fatalf("panic did not propagate the sentinel copy error: %v", r)
		}
		if os.Stderr != original {
			t.Fatal("os.Stderr was not restored after the copy-error panic")
		}
	}()
	capturePipelineStderrWith(&captureOpts{
		wrap: func(r io.Reader) io.Reader { return prefixReader },
	}, func() {
		_, _ = io.WriteString(os.Stderr, coldPayload) // f may write; the injected reader still determines the stream
	})
}

// failingReader returns its stored error on the first Read, after io.MultiReader
// has exhausted the prefix.
type failingReader struct {
	err error
}

func (f failingReader) Read([]byte) (int, error) { return 0, f.err }
