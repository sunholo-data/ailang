package executor

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"
)

// The production failure, reproduced against a plain Scanner and then against
// the reader that replaces it.
//
// 2026-09-14: four sprint-planner tasks died with
// "stdout scanner error: bufio.Scanner: token too long". The agent had already
// done the work; one over-long line destroyed the run.
func TestLineReader_SurvivesTheLineThatKilledAScanner(t *testing.T) {
	huge := strings.Repeat("x", 2*1024*1024) // 2MB, over codex's old 1MB cap
	stream := "first\n" + huge + "\nlast\n"

	// Control: a Scanner configured exactly as codex was DOES die here. Without
	// this the test would not be evidence of anything.
	sc := bufio.NewScanner(strings.NewReader(stream))
	sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	var scanned int
	for sc.Scan() {
		scanned++
	}
	if sc.Err() == nil {
		t.Fatal("control failed: a 1MB-capped Scanner must reject a 2MB line, or this test proves nothing")
	}
	if scanned != 1 {
		t.Fatalf("control: Scanner read %d lines before dying, want 1", scanned)
	}

	// The replacement reads every line, including the one after the huge one —
	// which is what the old code never reached.
	lr := NewLineReader(strings.NewReader(stream))
	var got []string
	for lr.Scan() {
		got = append(got, lr.Text())
	}
	if lr.Err() != nil {
		t.Fatalf("a long line must never be an error: %v", lr.Err())
	}
	if len(got) != 3 {
		t.Fatalf("read %d lines, want 3 — the line AFTER the long one is the one that was being lost", len(got))
	}
	if got[0] != "first" || got[2] != "last" {
		t.Errorf("surrounding lines corrupted: %q / %q", got[0], got[2])
	}
	if len(got[1]) != 2*1024*1024 {
		t.Errorf("a 2MB line under the 16MB cap must arrive whole, got %d bytes", len(got[1]))
	}
	if lr.Truncations != 0 {
		t.Errorf("nothing was over the cap; Truncations = %d", lr.Truncations)
	}
}

// Past the cap the line is CUT, not dropped, and the cut is reported — a
// caller that then fails to parse JSON must be able to say why.
func TestLineReader_TruncatesPastTheCapAndSaysSo(t *testing.T) {
	lr := NewLineReader(strings.NewReader(strings.Repeat("y", MaxLineBytes+4096) + "\nafter\n"))

	if !lr.Scan() {
		t.Fatal("an over-cap line must still yield a line")
	}
	if !lr.Truncated() {
		t.Error("truncation must be visible on the line it happened to")
	}
	if len(lr.Text()) != MaxLineBytes {
		t.Errorf("truncated to %d bytes, want the cap %d", len(lr.Text()), MaxLineBytes)
	}
	if !lr.Scan() || lr.Text() != "after" {
		t.Error("the stream must resync at the next newline, not lose the rest of the output")
	}
	if lr.Truncations != 1 {
		t.Errorf("Truncations = %d, want 1", lr.Truncations)
	}
	if lr.Err() != nil {
		t.Errorf("truncation is not an error: %v", lr.Err())
	}
}

// A final line with no trailing newline is still a line. Dropping it would lose
// a result on an unterminated stream — the most likely place for one.
func TestLineReader_KeepsAnUnterminatedFinalLine(t *testing.T) {
	lr := NewLineReader(strings.NewReader("one\ntwo-no-newline"))
	var got []string
	for lr.Scan() {
		got = append(got, lr.Text())
	}
	if len(got) != 2 || got[1] != "two-no-newline" {
		t.Errorf("unterminated final line lost: %v", got)
	}
}

func TestLineReader_EmptyStream(t *testing.T) {
	lr := NewLineReader(strings.NewReader(""))
	if lr.Scan() {
		t.Error("an empty stream yields no lines")
	}
	if lr.Err() != nil {
		t.Errorf("EOF is not an error: %v", lr.Err())
	}
}

type errReader struct{ n int }

func (e *errReader) Read(p []byte) (int, error) {
	if e.n > 0 {
		e.n--
		p[0] = 'a'
		return 1, nil
	}
	return 0, errors.New("pipe broke")
}

// A GENUINE read error must still surface — the point is to stop conflating
// "this line is long" with "the stream is broken", not to swallow both.
func TestLineReader_RealReadErrorsSurface(t *testing.T) {
	lr := NewLineReader(&errReader{n: 3})
	for lr.Scan() {
	}
	if lr.Err() == nil {
		t.Error("a broken pipe must be reported")
	}
	if errors.Is(lr.Err(), io.EOF) {
		t.Error("a read error must not be reported as EOF")
	}
}
