package mission

import (
	"os"
	"testing"
	"time"
)

// Pace verdicts depend on the calendar the weekday rule is judged in; pin it to UTC so the
// suite means the same thing on the rig (CEST) and on CI.
func TestMain(m *testing.M) {
	paceLocation = time.UTC
	os.Exit(m.Run())
}

// Mark, attended 2026-09-30: "20% a day is fine assuming 5 day weeks ... having unused
// slack used up on weekends is good."
func TestWeekdayPacePercent(t *testing.T) {
	mon := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC) // a Monday
	for _, tc := range []struct {
		name       string
		start, now time.Time
		want       float64
	}{
		{"one weekday is 20%", mon, mon.Add(24 * time.Hour), 20},
		{"five weekdays are 100%", mon, mon.Add(5 * 24 * time.Hour), 100},
		{"the weekend adds nothing", mon.Add(5 * 24 * time.Hour), mon.Add(7 * 24 * time.Hour), 0},
		{"a full week clamps at 100", mon, mon.Add(7 * 24 * time.Hour), 100},
		{"a Saturday-start window is flat until Monday", mon.Add(5*24*time.Hour + 17*time.Hour), mon.Add(7*24*time.Hour + 12*time.Hour), 10},
		{"half a Wednesday", mon.Add(2 * 24 * time.Hour), mon.Add(2*24*time.Hour + 12*time.Hour), 10},
		{"now before start is 0", mon, mon.Add(-time.Hour), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := WeekdayPacePercent(tc.start, tc.now, time.UTC); got < tc.want-0.01 || got > tc.want+0.01 {
				t.Fatalf("pace = %.2f%%, want %.2f%%", got, tc.want)
			}
		})
	}
	// The calendar matters: 23:00 UTC Friday is already Saturday in UTC+2.
	cest := time.FixedZone("CEST", 2*3600)
	fri := time.Date(2026, 10, 2, 21, 0, 0, 0, time.UTC)
	if a, b := WeekdayPacePercent(fri, fri.Add(2*time.Hour), time.UTC), WeekdayPacePercent(fri, fri.Add(2*time.Hour), cest); a <= b {
		t.Fatalf("UTC pace %.2f should exceed CEST pace %.2f across the local midnight", a, b)
	}
}
