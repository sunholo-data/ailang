package pi

import (
	"encoding/json"
	"testing"
)

func TestPiEvidencePreservesPresenceAndDecimal(t *testing.T) {
	line := []byte(`{"type":"message_end","message":{"role":"assistant","provider":"p","model":"m","requestId":"req1","usage":{"input":0,"output":9,"cacheRead":0,"cacheWrite":0,"cost":{"total":0.000000123456789}}}}`)
	ev, err := parsePiEvent(line)
	if err != nil {
		t.Fatal(err)
	}
	c := newPiUsageCapture()
	c.observe(line, ev)
	x := c.snapshot()
	if len(x.Observations) != 1 {
		t.Fatalf("observations: %+v", x)
	}
	u := x.Observations[0]
	if u.Input == nil || *u.Input != 0 || u.Reasoning != nil || u.CacheRead == nil || *u.CacheRead != 0 {
		t.Fatalf("presence lost: %+v", u)
	}
	if u.CostAmount == nil || *u.CostAmount != "0.000000123456789" || u.RequestID == nil || *u.RequestID != "req1" {
		t.Fatalf("source lost: %+v", u)
	}
	if x.CoverageComplete {
		t.Fatal("pi does not attest exhaustive billable request coverage")
	}
	b, _ := json.Marshal(x)
	if !json.Valid(b) {
		t.Fatal("not JSON")
	}
}

func TestPiEvidenceCountsOnlySettledLeavesAndRetainsGaps(t *testing.T) {
	c := newPiUsageCapture()
	lines := []string{
		`{"type":"message_update","message":{"role":"assistant","usage":{"input":99}}}`,
		`{"type":"message_end","message":{"role":"assistant","usage":{"input":3,"output":7,"reasoning":2}}}`,
		`{"type":"turn_end","message":{"role":"assistant","usage":{"input":3,"output":7,"reasoning":2}}}`,
		`{"type":"auto_retry_start","attempt":1}`,
		`{"type":"auto_compaction_start"}`,
		`{"type":"agent_settled"}`,
	}
	for _, line := range lines {
		ev, err := parsePiEvent([]byte(line))
		if err != nil {
			t.Fatal(err)
		}
		c.observe([]byte(line), ev)
	}
	c.observe([]byte("broken stream"), nil)
	x := c.snapshot()
	if len(x.Observations) != 1 || *x.Observations[0].Output != 7 || *x.Observations[0].Reasoning != 2 {
		t.Fatalf("double-count or lost split: %+v", x)
	}
	if x.RetryEvents != 1 || x.CompactionEvents != 1 || !x.Settled || len(x.UnavailableReasons) < 3 {
		t.Fatalf("gaps lost: %+v", x)
	}
}
