package firestore

import (
	"testing"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/sunholo-data/ailang/internal/feedbackgate"
)

// readAttempts / readCount MUST return an error alongside their value so the
// transaction can abort on a transient Firestore read failure instead of
// silently committing a reset window/counter (round-2 fix, evaluator issue #1:
// fail closed, no silent fallback — CLAUDE.md Critical Principle 2). The
// no-emulator convention means we can't drive a real transient read here, so
// this is a signature-level guard: if either helper ever drops its error
// return, this file stops compiling and the swallow-regression is caught.
var (
	_ func(*firestore.Transaction, *firestore.DocumentRef) ([]time.Time, error) = readAttempts
	_ func(*firestore.Transaction, *firestore.DocumentRef) (int, error)         = readCount
)

// Compile-time assertions that the adapters satisfy the feedbackgate interfaces.
// If these break, the interface drifted and the wiring would silently fail.
var (
	_ feedbackgate.CooldownStore = (*FeedbackGateCooldownStore)(nil)
	_ feedbackgate.BudgetStore   = (*FeedbackGateBudgetStore)(nil)
)

// TestTrimAndCount exercises the pure window math that backs the cooldown
// adapter. No Firestore/emulator is touched — trimAndCount is extracted so ALL
// window logic is testable without credentials (the firestore-package
// no-emulator convention). Boundaries are documented and asserted:
//   - the trailing 24h window is HALF-OPEN: an attempt strictly older than
//     now-24h is dropped; an attempt exactly at now-24h is KEPT.
//   - the hour/day counts include an attempt exactly at the window edge
//     (now-1h / now-24h are counted).
func TestTrimAndCount(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		attempts []time.Time
		wantKept int
		wantHour int
		wantDay  int
	}{
		{
			name:     "empty attempts",
			attempts: nil,
			wantKept: 0,
			wantHour: 0,
			wantDay:  0,
		},
		{
			name: "single recent attempt within the hour",
			attempts: []time.Time{
				now.Add(-30 * time.Minute),
			},
			wantKept: 1,
			wantHour: 1,
			wantDay:  1,
		},
		{
			name: "attempt exactly at the 1h boundary is counted in the hour",
			attempts: []time.Time{
				now.Add(-1 * time.Hour),
			},
			wantKept: 1,
			wantHour: 1,
			wantDay:  1,
		},
		{
			name: "attempt just past the 1h boundary counts to day not hour",
			attempts: []time.Time{
				now.Add(-1*time.Hour - time.Second),
			},
			wantKept: 1,
			wantHour: 0,
			wantDay:  1,
		},
		{
			name: "attempt exactly at the 24h boundary is KEPT and counted in the day",
			attempts: []time.Time{
				now.Add(-24 * time.Hour),
			},
			wantKept: 1,
			wantHour: 0,
			wantDay:  1,
		},
		{
			name: "attempt strictly older than 24h is trimmed",
			attempts: []time.Time{
				now.Add(-24*time.Hour - time.Second),
			},
			wantKept: 0,
			wantHour: 0,
			wantDay:  0,
		},
		{
			name: "mixed: one stale (trimmed), one in-day, one in-hour",
			attempts: []time.Time{
				now.Add(-48 * time.Hour),   // trimmed
				now.Add(-5 * time.Hour),    // in day, not hour
				now.Add(-10 * time.Minute), // in hour
			},
			wantKept: 2,
			wantHour: 1,
			wantDay:  2,
		},
		{
			name: "cross-day span: attempts on both sides of a UTC midnight, all within 24h",
			attempts: []time.Time{
				// now is 2026-07-10 12:00Z; these straddle 2026-07-10 00:00Z
				now.Add(-13 * time.Hour), // 2026-07-09 23:00Z, in 24h window
				now.Add(-11 * time.Hour), // 2026-07-10 01:00Z, in 24h window
			},
			wantKept: 2,
			wantHour: 0,
			wantDay:  2,
		},
		{
			name: "out-of-order attempts are handled (no sort assumption)",
			attempts: []time.Time{
				now.Add(-10 * time.Minute),
				now.Add(-48 * time.Hour),
				now.Add(-2 * time.Hour),
			},
			wantKept: 2,
			wantHour: 1,
			wantDay:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kept, hour, day := trimAndCount(tt.attempts, now)
			if len(kept) != tt.wantKept {
				t.Errorf("kept: got %d, want %d", len(kept), tt.wantKept)
			}
			if hour != tt.wantHour {
				t.Errorf("hour: got %d, want %d", hour, tt.wantHour)
			}
			if day != tt.wantDay {
				t.Errorf("day: got %d, want %d", day, tt.wantDay)
			}
			// Every kept attempt must be within the 24h window (not trimmed).
			cutoff := now.Add(-cooldownWindow)
			for _, a := range kept {
				if a.Before(cutoff) {
					t.Errorf("kept a trimmed attempt %v (cutoff %v)", a, cutoff)
				}
			}
		})
	}
}

// TestTrimAndCountIncludesJustRecorded records that trimAndCount + append is
// what the transaction wrapper does: after appending now, the counts include
// the new attempt.
func TestTrimAndCountIncludesJustRecorded(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	// Two prior attempts within the hour, plus we append now.
	attempts := []time.Time{
		now.Add(-20 * time.Minute),
		now.Add(-5 * time.Minute),
	}
	kept, _, _ := trimAndCount(attempts, now)
	kept = append(kept, now)
	_, hour, day := trimAndCount(kept, now)
	if hour != 3 {
		t.Errorf("hour after append: got %d, want 3", hour)
	}
	if day != 3 {
		t.Errorf("day after append: got %d, want 3", day)
	}
}

// TestSaturationCap asserts that once the kept window is at/over the cap, a new
// attempt is NOT appended and the returned window saturates at the cap. This
// bounds Firestore doc size under a flood; precision above the cap is
// meaningless because applyCooldown only compares > MaxDispatchPerHour/Day.
func TestSaturationCap(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)

	// Fill exactly cooldownSaturationCap attempts, all within the hour.
	attempts := make([]time.Time, cooldownSaturationCap)
	for i := range attempts {
		attempts[i] = now.Add(-time.Duration(i) * time.Second)
	}

	kept, appended := appendCapped(attempts, now)
	if appended {
		t.Errorf("expected no append when at saturation cap, but appended")
	}
	if len(kept) != cooldownSaturationCap {
		t.Errorf("kept len: got %d, want %d (saturated)", len(kept), cooldownSaturationCap)
	}

	// Below the cap, the attempt IS appended.
	below := attempts[:cooldownSaturationCap-1]
	kept2, appended2 := appendCapped(below, now)
	if !appended2 {
		t.Errorf("expected append when below saturation cap")
	}
	if len(kept2) != cooldownSaturationCap {
		t.Errorf("kept len after append: got %d, want %d", len(kept2), cooldownSaturationCap)
	}
}

// TestSaturationCapTrimsBeforeChecking ensures the cap is applied to the
// TRIMMED window: stale attempts don't count against the cap.
func TestSaturationCapTrimsBeforeChecking(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)

	// cap stale attempts (older than 24h) + 1 fresh. After trimming, only the
	// fresh one remains, so we are far below the cap and MUST append.
	attempts := make([]time.Time, cooldownSaturationCap)
	for i := range attempts {
		attempts[i] = now.Add(-48*time.Hour - time.Duration(i)*time.Second)
	}
	attempts = append(attempts, now.Add(-1*time.Minute))

	kept, appended := appendCapped(attempts, now)
	if !appended {
		t.Errorf("expected append after stale attempts trimmed below cap")
	}
	if len(kept) != 2 {
		t.Errorf("kept len: got %d, want 2 (1 fresh + just-recorded)", len(kept))
	}
}
