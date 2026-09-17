package trace

import (
	"fmt"
	"strings"
	"testing"
	"unsafe"
)

// A render site that already bounded a value (eval.ShowBounded) hands the
// collector a string of exactly maxValueBytes plus the elision marker. The
// collector's own bound must recognise it and leave it alone; otherwise every
// pre-bounded value would be truncated a second time and carry two markers.
func TestTruncateValueIsIdempotentOnPreBoundedStrings(t *testing.T) {
	prefix := strings.Repeat("a", 64)
	bounded := fmt.Sprintf("%s…(+%d bytes elided)", prefix, 9000)
	if got := truncateValue(bounded, 64); got != bounded {
		t.Fatalf("pre-bounded value was re-truncated:\n got %q\nwant %q", got, bounded)
	}
	// A natural string that merely exceeds the budget is still truncated.
	if got := truncateValue(strings.Repeat("b", 100), 64); !strings.HasSuffix(got, "…(+36 bytes elided)") {
		t.Fatalf("natural overflow not truncated: %q", got)
	}
	// The marker is position-anchored: the same text at a different offset is
	// ordinary content and is truncated like anything else.
	shifted := "x" + bounded
	if got := truncateValue(shifted, 64); got == shifted {
		t.Fatalf("marker at the wrong offset was treated as pre-bounded")
	}
}

func TestRedactValueIsIdempotent(t *testing.T) {
	d := RedactedDescriptor(4096)
	if got := redactValue(d); got != d {
		t.Fatalf("descriptor re-redacted: %q -> %q", d, got)
	}
	if got := redactValue("payload"); got != "<redacted:7 bytes>" {
		t.Fatalf("plain value not redacted: %q", got)
	}
}

// Through the real record() path: a value the site bounded at the collector's
// budget must come out of Events() unchanged.
func TestRecordKeepsPreBoundedValuesUnchanged(t *testing.T) {
	c := NewCollectorWithTier(TierDeep)
	budget, redacted := c.ValueBudget()
	if redacted || budget != DefaultMaxValueBytes {
		t.Fatalf("ValueBudget = (%d, %v), want (%d, false)", budget, redacted, DefaultMaxValueBytes)
	}
	pre := fmt.Sprintf("%s…(+%d bytes elided)", strings.Repeat("v", budget), 123456)
	c.RecordEffect("FS", "readFile", []string{pre}, pre)
	evts := c.Events()
	e := evts[len(evts)-1].Effect
	if e.Args[0] != pre || e.Result != pre {
		t.Fatalf("pre-bounded value altered by record(): %.60q / %.60q", e.Args[0], e.Result)
	}

	c.SetValueMode(ValuesRedacted)
	if _, redacted := c.ValueBudget(); !redacted {
		t.Fatal("ValueBudget must report redacted mode")
	}
	d := RedactedDescriptor(999)
	c.RecordEffect("FS", "readFile", []string{d}, d)
	evts = c.Events()
	if e := evts[len(evts)-1].Effect; e.Args[0] != d || e.Result != d {
		t.Fatalf("descriptor altered by record(): %q / %q", e.Args[0], e.Result)
	}
}

// eventSize used to charge 128 bytes plus string contents, ignoring the event
// struct, the payload struct, and every string header — so a 256 MB "cap"
// admitted well over a gigabyte of real heap. The floor is now the structs.
func TestEventSizeChargesStructsAndHeaders(t *testing.T) {
	floor := int(unsafe.Sizeof(TraceEvent{}))
	empty := TraceEvent{}
	if got := eventSize(empty); got < floor {
		t.Fatalf("eventSize(empty) = %d < sizeof(TraceEvent) %d", got, floor)
	}
	args := []string{"abc", "defgh"}
	evt := TraceEvent{Effect: &EffectEvent{EffectName: "FS", OpName: "readFile", Args: args, Result: "r"}}
	want := floor + 4*stringHeaderBytes + // Version, TraceID, SpanID, ParentSpanID headers (empty here)
		int(unsafe.Sizeof(EffectEvent{})) +
		len("FS") + len("readFile") + len("r") + 3*stringHeaderBytes +
		len(args)*stringHeaderBytes + len("abc") + len("defgh")
	if got := eventSize(evt); got != want {
		t.Fatalf("eventSize = %d, want %d", got, want)
	}
	if DefaultMaxRetainedBytes != 32<<20 {
		t.Fatalf("DefaultMaxRetainedBytes = %d, want 32 MB (D-B, ratified 2026-09-16)", DefaultMaxRetainedBytes)
	}
}
