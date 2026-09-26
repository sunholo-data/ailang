package mission

import (
	"testing"
	"time"
)

func TestParseAnthropicCLIUsage_RealOutput(t *testing.T) {
	out := `You are currently using your subscription to power your Claude Code usage

Current session: 12% used · resets Sep 22 at 2:30pm (Europe/Copenhagen)
Current week (all models): 14% used · resets Sep 28 at 7am (Europe/Copenhagen)
Current week (Fable): 1% used · resets Sep 28 at 7am (Europe/Copenhagen)
`
	now := time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC)
	ws, err := parseAnthropicCLIUsage(out, now)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(ws) != 2 {
		t.Fatalf("want 2 windows, got %d", len(ws))
	}
	if ws[0].WindowMinutes != 300 || ws[0].UsedPercent != 12 {
		t.Errorf("session: got %d min / %.0f%%", ws[0].WindowMinutes, ws[0].UsedPercent)
	}
	if ws[1].WindowMinutes != 10080 || ws[1].UsedPercent != 14 {
		t.Errorf("week: got %d min / %.0f%%", ws[1].WindowMinutes, ws[1].UsedPercent)
	}
	for i, w := range ws {
		if !w.ResetsAt.After(now) {
			t.Errorf("window %d resets in the past: %s", i, w.ResetsAt)
		}
		t.Logf("window %d: %d min, %.0f%% used, resets %s", i, w.WindowMinutes, w.UsedPercent, w.ResetsAt.UTC().Format(time.RFC3339))
	}
}

func TestParseAnthropicCLIUsage_RefusesGarbage(t *testing.T) {
	for _, bad := range []string{"", "no usage here", "Current session: 12% used"} {
		if _, err := parseAnthropicCLIUsage(bad, time.Now()); err == nil {
			t.Errorf("accepted garbage %q — a silent zero reads as 'plenty left'", bad)
		}
	}
}
