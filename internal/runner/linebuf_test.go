package runner

import (
	"bytes"
	"os"
	"sync"
	"testing"
)

func TestLineBufferedWriterPreservesEndpoint(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "endpoint")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got := NewLineBufferedWriter(f).Fd(); got != f.Fd() {
		t.Fatalf("endpoint fd = %d, want %d", got, f.Fd())
	}
	if got := NewLineBufferedWriter(&bytes.Buffer{}).Fd(); got != ^uintptr(0) {
		t.Fatalf("non-file writer must have invalid fd, got %d", got)
	}
}

// TestLineBufferedWriter verifies the M-TERMINAL-IO line-buffering contract:
// partial lines stay buffered (until flush()), and a newline auto-flushes so
// per-line terminal renders appear in real time.
func TestLineBufferedWriter(t *testing.T) {
	var sink bytes.Buffer
	lw := NewLineBufferedWriter(&sink)

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

func TestLineBufferedWriterConcurrentFlush(t *testing.T) {
	var sink bytes.Buffer
	writer := NewLineBufferedWriter(&sink)
	var workers sync.WaitGroup
	for range 4 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range 100 {
				if _, err := writer.Write([]byte("frame\n")); err != nil {
					t.Error(err)
					return
				}
				if err := writer.Flush(); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	workers.Wait()
	if got := bytes.Count(sink.Bytes(), []byte("frame\n")); got != 400 {
		t.Fatalf("concurrent frames = %d, want 400", got)
	}
}
