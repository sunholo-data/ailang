package main

import (
	"bytes"
	"testing"
)

// TestLineBufferedWriter verifies the M-TERMINAL-IO line-buffering contract:
// partial lines stay buffered (until flush()), and a newline auto-flushes so
// per-line terminal renders appear in real time.
func TestLineBufferedWriter(t *testing.T) {
	var sink bytes.Buffer
	lw := newLineBufferedWriter(&sink)

	// Partial line (no newline) stays buffered — nothing on screen yet.
	if _, err := lw.Write([]byte("Loading 42%")); err != nil {
		t.Fatal(err)
	}
	if sink.Len() != 0 {
		t.Fatalf("partial line should be buffered, but sink has %q", sink.String())
	}

	// Explicit Flush() (the IO.flush() path) makes the partial line visible.
	if err := lw.Flush(); err != nil {
		t.Fatal(err)
	}
	if got := sink.String(); got != "Loading 42%" {
		t.Fatalf("after Flush expected %q, got %q", "Loading 42%", got)
	}

	// A write containing a newline auto-flushes the whole buffer (the println path).
	sink.Reset()
	if _, err := lw.Write([]byte("frame\npartial")); err != nil {
		t.Fatal(err)
	}
	if got := sink.String(); got != "frame\npartial" {
		t.Fatalf("newline should flush buffer immediately, got %q", got)
	}
}
