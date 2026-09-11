package trace

import (
	"strings"
	"testing"
)

// M-TRACE-TIER-NOT-ENFORCED M2. `deep` is meant to have everything, so the
// bounds must cost as little of that as possible:
//   - per-VALUE truncation keeps every event, span and parent link, and only
//     elides the tail of pathologically large values. This is what makes an
//     accumulator recursion O(n) instead of O(n^2).
//   - the total-bytes cap evicts the OLDEST events, because the tail is what a
//     debugger needs, and announces the discontinuity in the stream itself.

func TestValueTruncationKeepsEveryEvent(t *testing.T) {
	c := NewCollectorWithTier(TierDeep)
	c.SetLimits(64, 0) // bound values, no retention cap

	huge := strings.Repeat("x", 5000)
	for i := 0; i < 50; i++ {
		c.RecordFunctionEnter("f", []string{huge})
		c.RecordFunctionExit("f", huge)
	}

	if got := len(c.Events()); got != 100 {
		t.Fatalf("kept %d events, want all 100 — value truncation must not drop events", got)
	}
	for _, e := range c.Events() {
		if e.Function == nil {
			continue
		}
		for _, a := range e.Function.Args {
			if len(a) > 200 {
				t.Errorf("argument not bounded: %d bytes", len(a))
			}
			if a != "" && !strings.Contains(a, "elided") {
				t.Errorf("truncated value carries no elision marker: %.40q", a)
			}
		}
	}
	if c.DroppedEvents() != 0 {
		t.Errorf("dropped %d events with no retention cap set", c.DroppedEvents())
	}
}

// TestRetentionCapKeepsTheTail is the policy call: on overflow, the newest
// events survive.
func TestRetentionCapKeepsTheTail(t *testing.T) {
	c := NewCollectorWithTier(TierDeep)
	c.SetLimits(0, 8000)

	c.RecordModuleStart("m", []string{"IO"})
	for i := 0; i < 400; i++ {
		c.RecordFunctionEnter(markerName(i), []string{strings.Repeat("y", 200)})
	}

	if c.DroppedEvents() == 0 {
		t.Fatal("nothing was evicted; the cap is not biting and the assertions below prove nothing")
	}

	evs := c.Events()
	var names []string
	sawModuleStart, sawTruncation := false, false
	for _, e := range evs {
		switch {
		case e.Event == EventModuleStart:
			sawModuleStart = true
		case e.Event == EventTraceTruncated:
			sawTruncation = true
			if e.Truncation == nil || e.Truncation.DroppedEvents == 0 {
				t.Error("truncation marker does not say how much was dropped")
			}
		case e.Function != nil:
			names = append(names, e.Function.Name)
		}
	}

	// The tail must be present: the LAST event recorded is the one a debugger
	// wants most.
	if len(names) == 0 || names[len(names)-1] != markerName(399) {
		t.Errorf("last retained function event is %v, want the final call — the tail was evicted", names[len(names)-1:])
	}
	// The head must be gone, or nothing was actually bounded.
	if len(names) > 0 && names[0] == markerName(0) {
		t.Error("the oldest event survived; eviction is not dropping from the head")
	}
	// module_start is pinned: it carries the capability set, and is the first
	// thing a naive oldest-first ring would discard.
	if !sawModuleStart {
		t.Error("module_start was evicted — the capability set is the most useful line in the trace")
	}
	// The artifact must declare its own incompleteness.
	if !sawTruncation {
		t.Error("no trace_truncated marker: a truncated trace is indistinguishable from a complete one that starts late")
	}
}

// TestObserversSeeTheCompleteStream is what lets `deep` genuinely keep
// everything: retention is an in-MEMORY bound, and a streaming consumer is
// notified before it applies.
func TestObserversSeeTheCompleteStream(t *testing.T) {
	c := NewCollectorWithTier(TierDeep)
	c.SetLimits(0, 4000)

	streamed := 0
	c.OnEvent = func(TraceEvent) { streamed++ }

	const n = 300
	for i := 0; i < n; i++ {
		c.RecordFunctionEnter("f", []string{strings.Repeat("z", 200)})
	}

	if c.DroppedEvents() == 0 {
		t.Fatal("cap did not bite; this test proves nothing")
	}
	if streamed != n {
		t.Errorf("observer saw %d of %d events; a streaming exporter must receive the full stream "+
			"even while the in-memory cap evicts", streamed, n)
	}
	if len(c.Events()) >= n {
		t.Error("retention did not bound anything, so the comparison above is vacuous")
	}
}

func markerName(i int) string {
	return "fn_" + string(rune('a'+i%26)) + "_" + itoa(i)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

// ValuesRedacted is the setting a confidentiality-bound workload runs: the
// COMPLETE call tree with no payloads. Orthogonal to tier on purpose — the
// useful combination is `deep` + redacted, which folding it into the tier ladder
// would have made unexpressible.
func TestValuesRedactedKeepsStructureAndDropsContent(t *testing.T) {
	const secret = "ya29.CANARY_LIVE_BEARER_TOKEN"

	full := NewCollectorWithTier(TierDeep)
	redacted := NewCollectorWithTier(TierDeep)
	redacted.SetValueMode(ValuesRedacted)

	for _, c := range []*Collector{full, redacted} {
		c.RecordModuleStart("m", []string{"IO", "Net"})
		c.RecordFunctionEnter("exchangeRefresh", []string{secret})
		c.RecordEffect("Net", "httpRequest", []string{secret}, secret)
		c.RecordFunctionExit("exchangeRefresh", secret)
		c.RecordModuleEnd("m", 1)
	}

	// Control: without redaction the canary IS present, so a green result below
	// cannot come from the fixture simply not recording anything.
	if !containsSecret(full.Events(), secret) {
		t.Fatal("control failed: the unredacted collector did not retain the canary, so this test proves nothing")
	}
	if containsSecret(redacted.Events(), secret) {
		t.Error("redacted collector retained the secret verbatim")
	}

	// The whole point: structure is untouched.
	if len(redacted.Events()) != len(full.Events()) {
		t.Errorf("redaction changed the event count: %d vs %d — it must drop CONTENT, not events",
			len(redacted.Events()), len(full.Events()))
	}
	var sawNames, sawEffect, sawSize bool
	for _, e := range redacted.Events() {
		if e.Function != nil && e.Function.Name == "exchangeRefresh" {
			sawNames = true
			for _, a := range e.Function.Args {
				if strings.Contains(a, "bytes") {
					sawSize = true
				}
			}
		}
		if e.Effect != nil && e.Effect.EffectName == "Net" && e.Effect.OpName == "httpRequest" {
			sawEffect = true
		}
	}
	if !sawNames {
		t.Error("function names were lost; the call tree is the audit value")
	}
	if !sawEffect {
		t.Error("effect name/op were lost; 'which effects ran' is the core provenance question")
	}
	if !sawSize {
		t.Error("no size descriptor: a byte count leaks no content but keeps real debugging signal")
	}
}

// TestResolveValueModeFailsClosed: a typo must not silently record everything.
func TestResolveValueModeFailsClosed(t *testing.T) {
	for _, s := range []string{"on", "full", ""} {
		if m, err := ResolveValueMode(s); err != nil || m != ValuesFull {
			t.Errorf("ResolveValueMode(%q) = %v, %v; want ValuesFull, nil", s, m, err)
		}
	}
	for _, s := range []string{"off", "redacted", "none", "OFF"} {
		if m, err := ResolveValueMode(s); err != nil || m != ValuesRedacted {
			t.Errorf("ResolveValueMode(%q) = %v, %v; want ValuesRedacted, nil", s, m, err)
		}
	}
	m, err := ResolveValueMode("of") // the typo that matters
	if err == nil {
		t.Error("a misspelled setting was accepted; a typo must not become a data leak")
	}
	if m != ValuesRedacted {
		t.Error("on error the mode must fall to ValuesRedacted — fail closed, not open")
	}
}

func containsSecret(evs []TraceEvent, secret string) bool {
	for _, e := range evs {
		if e.Function != nil {
			if strings.Contains(e.Function.Result, secret) {
				return true
			}
			for _, a := range e.Function.Args {
				if strings.Contains(a, secret) {
					return true
				}
			}
		}
		if e.Effect != nil {
			if strings.Contains(e.Effect.Result, secret) {
				return true
			}
			for _, a := range e.Effect.Args {
				if strings.Contains(a, secret) {
					return true
				}
			}
		}
	}
	return false
}
