package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/mission"
)

// D-FLEET-28: the quorum's Anthropic seat refuses only on a MEASURED refusal and runs
// on an unreadable ("unknown") quota, matching the driver's lane-probe rule. Blocked()
// stays true for "unknown" (admission and `mission quota --over` depend on it), so the
// seat must not gate on Blocked() alone.
func TestAnthropicSeatRefused_UnreadableIsNotExhausted(t *testing.T) {
	cases := []struct {
		state    string
		enforced bool
		refuse   bool
	}{
		{"unknown", true, false}, // the D-FLEET-28 case: usage endpoint 429, CLI fallback dead
		{"over", true, true},
		{"expired", true, true},
		{"ok", true, false},
		{"over", false, false}, // AILANG_ANTHROPIC_RATION=0: reported, not enforced
		{"unknown", false, false},
	}
	for _, c := range cases {
		a := mission.AnthropicQuotaObservation{State: c.state, Enforced: c.enforced}
		if got := anthropicSeatRefused(a); got != c.refuse {
			t.Errorf("state=%q enforced=%v: anthropicSeatRefused = %v, want %v", c.state, c.enforced, got, c.refuse)
		}
	}
	// The premise of the fix: Blocked() alone would refuse the unknown case.
	if !(mission.AnthropicQuotaObservation{State: "unknown", Enforced: true}).Blocked() {
		t.Fatal("premise changed: Blocked() no longer refuses an unknown reading — revisit anthropicSeatRefused")
	}
}
