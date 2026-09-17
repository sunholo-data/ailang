package effects

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// M-V1-MEMORY-FOOTPRINT M2 (D-E): with a sink attached, Debug.log lines are
// written when they arrive and never accumulate; --log-level is applied on
// arrival. Without a sink the collect-then-flush host contract is unchanged.

func attachedSink(minLevel int) (*DebugContext, *bytes.Buffer) {
	var buf bytes.Buffer
	d := NewDebugContext()
	DebugSink{W: &buf, MinLevel: minLevel}.Attach(d)
	return d, &buf
}

func TestDebugLogStreamsThroughSinkAndDoesNotAccumulate(t *testing.T) {
	d, buf := attachedSink(0)
	for i := 0; i < 1000; i++ {
		d.Log(fmt.Sprintf("line %d", i), "t.ail:1")
	}
	if got := len(d.Collect().Logs); got != 0 {
		t.Fatalf("sink attached but %d lines were retained", got)
	}
	if n := strings.Count(buf.String(), "\n"); n != 1000 {
		t.Fatalf("sink received %d lines, want 1000", n)
	}
	if !strings.Contains(buf.String(), "line 999") {
		t.Fatalf("last line missing from sink output")
	}
}

func TestDebugLogFiltersByLevelOnArrival(t *testing.T) {
	d, buf := attachedSink(3) // ERROR
	d.Log(`{"severity":"DEBUG","message":"noise"}`, "t.ail:1")
	d.Log(`{"severity":"ERROR","message":"boom"}`, "t.ail:2")
	d.Log("plain text always passes", "t.ail:3")
	d.Log(`{"message":"no severity passes"}`, "t.ail:4")
	out := buf.String()
	if strings.Contains(out, "noise") {
		t.Fatalf("DEBUG line passed an ERROR threshold: %s", out)
	}
	for _, want := range []string{`"boom"`, "plain text always passes", "no severity passes"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	if got := len(d.Collect().Logs); got != 0 {
		t.Fatalf("filtered lines were retained: %d", got)
	}
}

func TestDebugCheckFailureStreamsAndPassesAreNotRetained(t *testing.T) {
	d, buf := attachedSink(0)
	d.Check(true, "fine", "t.ail:1")
	d.Check(false, "broken", "t.ail:2")
	if !strings.Contains(buf.String(), "[ASSERT FAIL] broken at t.ail:2") {
		t.Fatalf("failed check not streamed: %q", buf.String())
	}
	if strings.Contains(buf.String(), "fine") {
		t.Fatalf("passed check was written: %q", buf.String())
	}
	if got := len(d.Collect().Assertions); got != 0 {
		t.Fatalf("sink attached but %d assertions were retained", got)
	}
	// Flush after streaming is a no-op: nothing is written twice.
	before := buf.Len()
	DebugSink{W: buf}.Flush(d)
	if buf.Len() != before {
		t.Fatalf("Flush re-emitted streamed output")
	}
}

func TestDebugContextWithoutSinkStillCollects(t *testing.T) {
	d := NewDebugContext()
	d.Log("kept", "t.ail:1")
	d.Check(false, "kept too", "t.ail:2")
	out := d.Collect()
	if len(out.Logs) != 1 || len(out.Assertions) != 1 {
		t.Fatalf("host contract changed: %+v", out)
	}
}

// A request clone must not share the log buffer of the server's context —
// serve-api runs requests concurrently on shallow clones. It must share the
// sink, so streamed lines still reach stderr.
func TestCloneGivesEachRequestItsOwnDebugContext(t *testing.T) {
	ctx := NewEffContext(nil)
	d, buf := attachedSink(0)
	ctx.Debug = d
	clone := ctx.Clone().(*EffContext)
	if clone.Debug == nil || clone.Debug == ctx.Debug {
		t.Fatalf("clone shares the Debug context (%p == %p)", clone.Debug, ctx.Debug)
	}
	clone.Debug.Log("from clone", "t.ail:1")
	if !strings.Contains(buf.String(), "from clone") {
		t.Fatalf("clone's sink not inherited: %q", buf.String())
	}
	// Without a sink, the clone still gets its own accumulator.
	plain := NewEffContext(nil)
	plain.Debug = NewDebugContext()
	pc := plain.Clone().(*EffContext)
	pc.Debug.Log("only in clone", "t.ail:1")
	if len(plain.Debug.Collect().Logs) != 0 || len(pc.Debug.Collect().Logs) != 1 {
		t.Fatalf("clone accumulator not isolated")
	}
}
