package eval

import "testing"

// M-V1-MEMORY-FOOTPRINT M2: the typed evaluator's per-call trace appended
// forever. It now keeps the newest DefaultMaxTypedTraceEntries and counts
// what it dropped.
func TestTypedTraceCollectorKeepsTheTail(t *testing.T) {
	c := &TraceCollector{Enabled: true}
	n := DefaultMaxTypedTraceEntries*2 + 7
	for i := 0; i < n; i++ {
		c.add(TraceEntry{CallSiteID: uint64(i)})
	}
	if len(c.Entries) > DefaultMaxTypedTraceEntries {
		t.Fatalf("retained %d entries, cap %d", len(c.Entries), DefaultMaxTypedTraceEntries)
	}
	if c.Dropped != n-len(c.Entries) {
		t.Fatalf("Dropped = %d, want %d", c.Dropped, n-len(c.Entries))
	}
	if last := c.Entries[len(c.Entries)-1].CallSiteID; last != uint64(n-1) {
		t.Fatalf("newest entry lost: last CallSiteID %d, want %d", last, n-1)
	}
	if first := c.Entries[0].CallSiteID; first != uint64(n-len(c.Entries)) {
		t.Fatalf("oldest retained %d, want %d (contiguous tail)", first, n-len(c.Entries))
	}
}
