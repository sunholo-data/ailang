package trace

import (
	"encoding/json"
	"strings"
	"testing"
)

// M-TRACE-LABEL-AWARE: inside a label scope the collector records structure
// (names, arity, span tree) but no value, whichever site rendered it.
func TestLabelScopeRedactsEveryValueSurface(t *testing.T) {
	const canary = "CANARY_FAKE_NOT_A_REAL_SECRET"
	c := NewCollectorWithTier(TierDeep)

	c.RecordFunctionEnter("before", []string{"visible"})
	pop := c.PushLabelScope("secret")
	c.RecordFunctionEnter("exchange", []string{canary, "6"})
	c.RecordEffect("Net", "httpRequest", []string{"https://example.test", canary}, canary)
	c.RecordModedEffect("Rand", "rand_int", []string{canary}, canary, "seeded", "")
	c.RecordAIEffect("call", []string{canary}, canary, nil)
	c.RecordFunctionExit("exchange", canary)
	pop()
	c.RecordFunctionExit("before", "visible")

	raw, err := json.Marshal(c.Events())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), canary) {
		t.Fatalf("a value recorded inside a label scope reached the trace:\n%s", raw)
	}
	if !strings.Contains(string(raw), "visible") {
		t.Fatal("values outside the scope must still be recorded (control)")
	}
	evts := c.Events()
	enter := evts[1].Function
	if enter.Name != "exchange" || len(enter.Args) != 2 || enter.Args[0] != LabelRedactedMarker("secret") {
		t.Fatalf("enter inside scope = %+v; want name and arity kept, args %q", enter, LabelRedactedMarker("secret"))
	}
}

func TestLabelScopeNestsAndRestores(t *testing.T) {
	c := NewCollectorWithTier(TierDeep)
	popOuter := c.PushLabelScope("secret")
	popInner := c.PushLabelScope("pii")
	if got := c.LabelScope(); got != "pii" {
		t.Fatalf("innermost scope = %q, want pii", got)
	}
	popInner()
	if got := c.LabelScope(); got != "secret" {
		t.Fatalf("after inner pop = %q, want secret", got)
	}
	popOuter()
	if got := c.LabelScope(); got != "" {
		t.Fatalf("after outer pop = %q, want no scope", got)
	}
}

func TestLabelScopeNilAndDisabledCollector(t *testing.T) {
	var c *Collector
	if pop := c.PushLabelScope("secret"); pop == nil {
		t.Fatal("PushLabelScope on a nil collector must return a no-op, not nil")
	} else {
		pop()
	}
	if c.LabelScope() != "" {
		t.Fatal("nil collector has no scope")
	}
}
