package runner

import (
	"bufio"
	"bytes"
	"io"
	"os"

	"golang.org/x/term"
)

// IsStdoutTTY reports whether stdout is connected to a terminal. Used to decide
// whether to line-buffer stdout (terminal: yes, pipe: no — see the buffer
// construction site in Run for context).
func IsStdoutTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// LineBufferedWriter buffers stdout but flushes whenever a newline is written
// (line buffering). This gives real-time terminal rendering for the common
// per-line render — a println appears immediately — while still coalescing the
// many small Fprint calls that compose a single line. Partial-line output
// (progress bars / game frames using "\r") is flushed via the IO.flush() effect.
// Replaces the prior unconditional 64KB full-buffer on a TTY (M-TERMINAL-IO),
// which only updated the screen once the buffer filled.
type LineBufferedWriter struct {
	w *bufio.Writer
}

// NewLineBufferedWriter wraps w in a 64KB line-flushing buffer.
func NewLineBufferedWriter(w io.Writer) *LineBufferedWriter {
	return &LineBufferedWriter{w: bufio.NewWriterSize(w, 64*1024)}
}

func (lw *LineBufferedWriter) Write(p []byte) (int, error) {
	n, err := lw.w.Write(p)
	if err != nil {
		return n, err
	}
	if bytes.IndexByte(p, '\n') >= 0 {
		if ferr := lw.w.Flush(); ferr != nil {
			return n, ferr
		}
	}
	return n, nil
}

// Flush writes any buffered bytes to the underlying writer. Satisfies the
// interface used by EffContext.FlushIO() and the exit-time flush.
func (lw *LineBufferedWriter) Flush() error { return lw.w.Flush() }
