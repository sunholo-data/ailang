package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// stderrGuard authenticates reserved line starts without buffering lines.
// Only the stderr copier mutates it until drainOutput joins that copier.
// Token holdback is bounded by len("policy-result:"); UTF-8 state is two bytes.
type stderrGuard struct {
	dst             *bufio.Writer
	pending         string
	atLineStart     bool
	terminator      byte
	endsWithNewline bool
}

func guardWorkerStderr(dst io.Writer) *stderrGuard {
	return &stderrGuard{dst: bufio.NewWriter(dst), atLineStart: true, endsWithNewline: true}
}

func (g *stderrGuard) Write(p []byte) (int, error) {
	for i, b := range p {
		if g.atLineStart {
			g.pending += string([]byte{b})
			switch {
			case g.pending == "policy:" || g.pending == "policy-result:":
				if _, err := g.dst.WriteString("worker: " + g.pending); err != nil {
					return i, err
				}
				g.pending = ""
				g.atLineStart = false
			case strings.HasPrefix("policy:", g.pending) || strings.HasPrefix("policy-result:", g.pending):
				continue
			default:
				if _, err := g.dst.WriteString(g.pending); err != nil {
					return i, err
				}
				g.pending = ""
				g.atLineStart = false
			}
		} else if err := g.dst.WriteByte(b); err != nil {
			return i, err
		}
		g.endsWithNewline = b == '\n'
		g.observeTerminator(b)
	}
	return len(p), g.dst.Flush()
}

// Terminators pass through immediately; this state handles split UTF-8 and
// reconsiders mismatched continuation bytes as possible new terminators.
func (g *stderrGuard) observeTerminator(b byte) {
	if (g.terminator == 0xc2 && b == 0x85) || (g.terminator == 0x80 && (b == 0xa8 || b == 0xa9)) {
		g.atLineStart = true
		g.terminator = 0
		return
	}
	if g.terminator == 0xe2 && b == 0x80 {
		g.terminator = 0x80
		return
	}
	g.terminator = 0
	switch b {
	case '\n', '\r', '\v', '\f', 0x1c, 0x1d, 0x1e:
		g.atLineStart = true
	case 0xc2, 0xe2:
		g.terminator = b
	}
}

// Flush is also called after cap cancellation or forced pipe closure so a
// strict token prefix already read from the worker is never lost.
func (g *stderrGuard) Flush() {
	if g.pending != "" {
		_, _ = g.dst.WriteString(g.pending)
		g.endsWithNewline = false
		g.pending = ""
	}
	_ = g.dst.Flush()
}

// supervisorLine is used only after relay drain. LF, specifically, separates
// contract lines even when the worker ended with CR or a Unicode separator.
func (g *stderrGuard) supervisorLine(format string, args ...any) {
	if !g.endsWithNewline {
		_ = g.dst.WriteByte('\n')
	}
	_, _ = fmt.Fprintf(g.dst, format, args...)
	_ = g.dst.Flush()
	g.endsWithNewline = strings.HasSuffix(format, "\n")
}
