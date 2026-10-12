package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// stderrGuard reserves contract tokens without buffering entire worker lines.
// Only a strict token prefix (at most 13 bytes) is held across writes. The
// three-byte suffix recognises UTF-8 separators even across read boundaries.
// Owned by the stderr copier; supervisorLine is used only after relay drain.
type stderrGuard struct {
	dst     io.Writer
	out     *bufio.Writer
	pending string
	suffix  [3]byte
	atStart bool
	endsLF  bool
	empty   bool
}

func newStderrGuard(dst io.Writer) *stderrGuard {
	return &stderrGuard{dst: dst, out: bufio.NewWriter(dst), atStart: true, empty: true}
}

func (g *stderrGuard) emit(b byte) {
	_ = g.out.WriteByte(b)
	g.empty = false
	g.endsLF = b == '\n'
	g.suffix[0], g.suffix[1], g.suffix[2] = g.suffix[1], g.suffix[2], b
	g.atStart = b == '\n' || b == '\r' || b == '\v' || b == '\f' || b == 0x1c || b == 0x1d || b == 0x1e ||
		(g.suffix[1] == 0xc2 && b == 0x85) ||
		(g.suffix[0] == 0xe2 && g.suffix[1] == 0x80 && (b == 0xa8 || b == 0xa9))
}

func (g *stderrGuard) Write(p []byte) (int, error) {
	for _, b := range p {
		if g.atStart {
			g.pending += string([]byte{b})
			if g.pending == "policy:" || g.pending == "policy-result:" {
				_, _ = g.out.WriteString("worker: ")
			} else if strings.HasPrefix("policy:", g.pending) || strings.HasPrefix("policy-result:", g.pending) {
				continue
			}
			for i := 0; i < len(g.pending); i++ {
				g.emit(g.pending[i])
			}
			g.pending = ""
		} else {
			g.emit(b)
		}
	}
	return len(p), g.out.Flush()
}

func (g *stderrGuard) Flush() error {
	for i := 0; i < len(g.pending); i++ {
		g.emit(g.pending[i])
	}
	g.pending = ""
	return g.out.Flush()
}

// supervisorLine keeps contract and diagnostic lines LF-delimited, including
// when the final worker bytes were unterminated or a held token prefix.
func (g *stderrGuard) supervisorLine(line string) {
	if !g.empty && !g.endsLF {
		fmt.Fprint(g.dst, "\n")
	}
	fmt.Fprintln(g.dst, line)
	g.empty, g.endsLF = false, true
}
