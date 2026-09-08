package mission

import (
	"os"
	"strings"
	"testing"
	"time"
)

func ratNow() time.Time { return time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC) }

// obs builds a history from (hoursAgo, weeklyFraction) pairs, oldest first.
func obs(now time.Time, pairs ...[2]float64) []ollamaObservation {
	out := make([]ollamaObservation, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, ollamaObservation{
			At:     now.Add(-time.Duration(p[0] * float64(time.Hour))),
			Weekly: p[1],
		})
	}
	return out
}

func gaugeOK(weekly float64) OllamaQuotaObservation {
	sess := 0.10
	return OllamaQuotaObservation{
		State: "ok", GaugeStatus: "OK", Reason: "gauge ok",
		SessionUsage: &sess, WeeklyUsage: &weekly,
	}
}

func TestOllamaRateRation_InsufficientHistoryIsUnpacedNotBlocked(t *testing.T) {
	now := ratNow()
	for name, history := range map[string][]ollamaObservation{
		"empty":       nil,
		"one reading": obs(now, [2]float64{2, 0.50}),
		"under 1h":    obs(now, [2]float64{0.5, 0.40}, [2]float64{0, 0.90}),
	} {
		got := applyOllamaRateRation(gaugeOK(0.5), history, now)
		if got.State != "ok" {
			t.Errorf("%s: state = %q, want ok — thin history must not block the fleet", name, got.State)
		}
		if !strings.Contains(got.Reason, "UNPACED") {
			t.Errorf("%s: reason = %q, want it to say UNPACED", name, got.Reason)
		}
	}
}

func TestOllamaRateRation_OverDailyRationBlocks(t *testing.T) {
	now := ratNow()
	// The real incident: 36.1% -> 43.1% -> 69.4% across ~16h = 33.3pp, vs a 10pp/day ration.
	history := obs(now, [2]float64{16, 0.361}, [2]float64{9, 0.431}, [2]float64{0, 0.694})
	got := applyOllamaRateRation(gaugeOK(0.694), history, now)
	if got.State != "over" {
		t.Fatalf("state = %q (%s), want over", got.State, got.Reason)
	}
	if got.GaugeStatus != "RATION" {
		t.Errorf("gauge status = %q, want RATION so the cause is legible", got.GaugeStatus)
	}
	if !strings.Contains(got.Reason, "33.3pp") {
		t.Errorf("reason = %q, want the measured 33.3pp", got.Reason)
	}
}

func TestOllamaRateRation_WithinRationStaysOK(t *testing.T) {
	now := ratNow()
	// 4pp over 20h — inside the 10pp/day allowance.
	history := obs(now, [2]float64{20, 0.30}, [2]float64{0, 0.34})
	got := applyOllamaRateRation(gaugeOK(0.34), history, now)
	if got.State != "ok" {
		t.Fatalf("state = %q (%s), want ok", got.State, got.Reason)
	}
	if !strings.Contains(got.Reason, "daily ration ok") {
		t.Errorf("reason = %q, want it to report the ration", got.Reason)
	}
}

func TestOllamaRateRation_WindowResetDoesNotLicenseABurst(t *testing.T) {
	now := ratNow()
	// Gauge rolls over mid-history: 0.90 -> 0.02, then climbs 20pp. Endpoint subtraction
	// would read (0.22 - 0.90) = -68pp and wave it through; summing positive deltas sees
	// the 20pp that was actually spent after the reset.
	history := obs(now, [2]float64{20, 0.90}, [2]float64{10, 0.02}, [2]float64{0, 0.22})
	got := applyOllamaRateRation(gaugeOK(0.22), history, now)
	if got.State != "over" {
		t.Fatalf("state = %q (%s), want over — 20pp was spent after the reset", got.State, got.Reason)
	}
	pp, _, ok := ollamaConsumedLast24h(history, now)
	if !ok || pp < 19.9 || pp > 20.1 {
		t.Errorf("consumed = %.2fpp (ok=%v), want ~20pp", pp, ok)
	}
}

func TestOllamaRateRation_NeverLaundersACriticalGauge(t *testing.T) {
	now := ratNow()
	crit := gaugeOK(0.97)
	crit.State, crit.GaugeStatus = "over", "CRITICAL"
	crit.Reason = "Ollama Cloud session or weekly gauge reached 95%; new cloud routing blocked"

	// A quiet trailing day would satisfy the rate ration on its own.
	got := applyOllamaRateRation(crit, obs(now, [2]float64{20, 0.96}, [2]float64{0, 0.97}), now)
	if got.State != "over" {
		t.Fatalf("state = %q, want over — the 95%% cutoff must survive the ration", got.State)
	}
	if got.GaugeStatus != "CRITICAL" {
		t.Errorf("gauge status = %q, want CRITICAL preserved", got.GaugeStatus)
	}
}

func TestOllamaObservations_RoundTripAndRetention(t *testing.T) {
	now := ratNow()
	paths := Paths{Home: t.TempDir()}
	sess, weekly := 0.44, 0.69

	got := recordOllamaObservation(paths, OllamaQuotaObservation{
		SessionUsage: &sess, WeeklyUsage: &weekly,
	}, now)
	if len(got) != 1 {
		t.Fatalf("history = %d, want 1", len(got))
	}
	// A second reading a day later keeps both; one 15 days back is pruned.
	got = recordOllamaObservation(paths, OllamaQuotaObservation{
		SessionUsage: &sess, WeeklyUsage: &weekly,
	}, now.Add(24*time.Hour))
	if len(got) != 2 {
		t.Fatalf("history = %d, want 2", len(got))
	}
	if reread := loadOllamaObservations(OllamaObservationsPath(paths), now.Add(24*time.Hour)); len(reread) != 2 {
		t.Errorf("reread = %d, want 2", len(reread))
	}
	if pruned := loadOllamaObservations(OllamaObservationsPath(paths), now.Add(20*24*time.Hour)); len(pruned) != 0 {
		t.Errorf("after 20 days: %d rows, want 0 (retention is 14 days)", len(pruned))
	}
}

func TestOllamaObservations_MalformedRowsAreSkippedNotFatal(t *testing.T) {
	now := ratNow()
	paths := Paths{Home: t.TempDir()}
	sess, weekly := 0.44, 0.69
	recordOllamaObservation(paths, OllamaQuotaObservation{SessionUsage: &sess, WeeklyUsage: &weekly}, now)

	path := OllamaObservationsPath(paths)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(path, append([]byte("{ not json\n"), body...), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := loadOllamaObservations(path, now); len(got) != 1 {
		t.Errorf("rows = %d, want 1 good row kept and the malformed one skipped", len(got))
	}
}
