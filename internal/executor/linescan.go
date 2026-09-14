package executor

// Reading an agent's line-oriented output without a length limit that can kill
// the run.
//
// Every executor reads its subprocess as JSONL with a bufio.Scanner, and a
// Scanner has a hard maximum token size: exceed it and Scan() stops, Err()
// returns "token too long", and the caller treats that as a failed task. The
// caps were four different arbitrary numbers — pi 8MB, claude/codex/opencode
// 1MB, motoko and the SSE reader on the 64KB default — so the same output would
// kill a task on one harness and not another.
//
// Measured 2026-09-14: four sprint-planner tasks died with
//
//	executor failed: codex task failed: stdout scanner error: bufio.Scanner: token too long
//
// A single over-long line is one event we cannot parse. Losing the WHOLE RUN
// over it is the wrong trade by a wide margin: the agent had already done the
// work, and the line is usually a large tool result or a file dump, not the
// final answer.
//
// So: one reader, one cap, and exceeding the cap degrades to a truncated line
// rather than an error. Truncation is REPORTED, never silent — a caller that
// parses JSON will fail on the fragment and must be able to say why.

import (
	"bufio"
	"io"
	"strings"
)

// MaxLineBytes bounds a single line held in memory.
//
// 16MB: comfortably above the largest legitimate JSONL event observed (a full
// file read as a tool result) and far below anything that threatens a container
// whose whole budget is measured in GB. It is a MEMORY bound, not a
// correctness one — nothing fails when a line exceeds it.
const MaxLineBytes = 16 << 20

// LineReader yields lines of arbitrary length, truncating past MaxLineBytes.
type LineReader struct {
	r         *bufio.Reader
	line      string
	truncated bool
	// Truncations counts lines that exceeded the cap, so a caller can report
	// "output was truncated" instead of reporting a parse error with no cause.
	Truncations int
	err         error
}

// NewLineReader wraps r. The bufio buffer starts small and grows only as needed.
func NewLineReader(r io.Reader) *LineReader {
	return &LineReader{r: bufio.NewReaderSize(r, 64*1024)}
}

// Scan advances to the next line, reporting false at EOF or on a read error.
//
// Deliberately Scanner-shaped so call sites change by one line. The difference
// is the one that matters: a long line is never a reason to stop.
func (l *LineReader) Scan() bool {
	var b strings.Builder
	l.truncated = false
	for {
		chunk, err := l.r.ReadString('\n')
		if b.Len() < MaxLineBytes {
			room := MaxLineBytes - b.Len()
			if len(chunk) > room {
				b.WriteString(chunk[:room])
				l.truncated = true
			} else {
				b.WriteString(chunk)
			}
		} else if len(chunk) > 0 {
			l.truncated = true
		}
		if err == nil {
			// Got a full line.
			l.line = strings.TrimRight(b.String(), "\n")
			if l.truncated {
				l.Truncations++
			}
			return true
		}
		if err == bufio.ErrBufferFull {
			continue // more of the same line
		}
		// EOF or a real error. A trailing fragment with no newline is still a
		// line; dropping it would lose a final result on an unterminated stream.
		if err == io.EOF {
			if b.Len() > 0 {
				l.line = strings.TrimRight(b.String(), "\n")
				if l.truncated {
					l.Truncations++
				}
				return true
			}
			return false
		}
		l.err = err
		return false
	}
}

// Text returns the current line, truncated to MaxLineBytes if it was longer.
func (l *LineReader) Text() string { return l.line }

// Bytes mirrors bufio.Scanner.Bytes so call sites port unchanged.
//
// Unlike Scanner.Bytes this is a FRESH slice, not a view into a reusable
// buffer — the Scanner contract's "invalidated by the next Scan" footgun does
// not apply, and a caller that retains it is safe.
func (l *LineReader) Bytes() []byte { return []byte(l.line) }

// Truncated reports whether the CURRENT line was cut.
func (l *LineReader) Truncated() bool { return l.truncated }

// Err returns a genuine read error — never a length error, which is the whole
// point. A nil here means the stream was read to the end.
func (l *LineReader) Err() error { return l.err }
