package mission

// The daily ration for Ollama Cloud, which is the one provider that cannot be paced the
// normal way.
//
// Codex and Anthropic both report a percentage AND a reset, so their allowance is a
// function of where we are inside the window. Ollama's hidden gauge reports only the
// fraction consumed — GET /api/usage returns session/weekly `usage` and nothing else, no
// capacity and no `resets_at` (verified against the live response, 2026-09-08). Without a
// reset there is no window position, so the codex-style allowance cannot be computed at all.
//
// That is why ollama has been the fleet's ONLY unrationed bucket, and it showed: the weekly
// gauge went 36.1% -> 43.1% -> 69.4% between 2026-09-07 17:02 and 2026-09-08 09:06, about
// 2.1 percentage points per hour, against a 10%/day ration of 0.42pp/h. Five times the
// intended rate with nothing to stop it before the 95% cutoff.
//
// The fix is to ration the RATE instead of the position. D-1 says "spend at most 10% of a
// bucket per day"; with a fraction gauge that is directly measurable as the percentage
// points consumed in the trailing 24 hours. A rate needs no capacity (the gauge is already
// a fraction) and no reset (it does not care where the window starts) — the two things the
// provider will not tell us.
//
// PURPOSE, which decides the shape: the ration paces UNATTENDED mission loops so attended
// sessions keep headroom. The gauge is account-wide, so attended spend legitimately
// consumes the loops' allowance and the loops yield first. That is the intended direction.

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ollamaObservationsRetention bounds the history file. 14 days is two weekly windows, so a
// full window is always reconstructible, and the file stays a few tens of KiB.
const ollamaObservationsRetention = 14 * 24 * time.Hour

// ollamaRateWindow is the trailing period the ration is measured over.
const ollamaRateWindow = 24 * time.Hour

// ollamaMinRateSpan is the shortest history that may produce a verdict.
//
// Below this a single burst reads as an enormous hourly rate and would block the fleet on
// one observation. An unpaced bucket that says so is safer than a paced one that is wrong.
const ollamaMinRateSpan = time.Hour

// ollamaObservation is one banked gauge reading.
type ollamaObservation struct {
	At      time.Time `json:"at"`
	Session float64   `json:"session"`
	Weekly  float64   `json:"weekly"`
}

// OllamaObservationsPath is where gauge readings are banked.
func OllamaObservationsPath(p Paths) string {
	return filepath.Join(p.Home, ".ailang", "state", "ollama-quota-observations.jsonl")
}

// recordOllamaObservation appends a reading and prunes expired ones.
//
// Failure to bank is deliberately NOT an error for the caller: a missing history makes the
// bucket unpaced and loud, which is the documented safe state, whereas failing the whole
// quota read would take routing down over a log file.
func recordOllamaObservation(p Paths, o OllamaQuotaObservation, now time.Time) []ollamaObservation {
	if o.SessionUsage == nil || o.WeeklyUsage == nil {
		return nil
	}
	path := OllamaObservationsPath(p)
	history := loadOllamaObservations(path, now)
	history = append(history, ollamaObservation{
		At: now.UTC(), Session: *o.SessionUsage, Weekly: *o.WeeklyUsage,
	})

	var buf strings.Builder
	for _, h := range history {
		line, err := json.Marshal(h)
		if err != nil {
			continue
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err == nil {
		_ = writeAtomic(path, []byte(buf.String()), 0o600)
	}
	return history
}

// loadOllamaObservations reads the banked history, dropping expired and malformed rows.
func loadOllamaObservations(path string, now time.Time) []ollamaObservation {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	cutoff := now.Add(-ollamaObservationsRetention)
	var out []ollamaObservation
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		var o ollamaObservation
		if json.Unmarshal(scanner.Bytes(), &o) != nil {
			continue // a malformed row must not uncover or invent a reading
		}
		if o.At.Before(cutoff) || o.At.After(now.Add(time.Minute)) {
			continue
		}
		out = append(out, o)
	}
	return out
}

// ollamaConsumedLast24h returns percentage points consumed in the trailing window, the
// span those observations actually cover, and whether a verdict is supported.
//
// Consumption is the sum of POSITIVE deltas between consecutive readings. Summing deltas
// rather than subtracting endpoints is what makes this reset-safe: when a window rolls the
// gauge drops, and a naive endpoint subtraction would report negative consumption and
// silently license a fresh burst. A drop contributes zero and the readings after it count
// normally.
func ollamaConsumedLast24h(history []ollamaObservation, now time.Time) (pp float64, span time.Duration, ok bool) {
	cutoff := now.Add(-ollamaRateWindow)
	var recent []ollamaObservation
	for _, h := range history {
		if !h.At.Before(cutoff) {
			recent = append(recent, h)
		}
	}
	if len(recent) < 2 {
		return 0, 0, false
	}
	span = recent[len(recent)-1].At.Sub(recent[0].At)
	if span < ollamaMinRateSpan {
		return 0, span, false
	}
	for i := 1; i < len(recent); i++ {
		if d := recent[i].Weekly - recent[i-1].Weekly; d > 0 {
			pp += d * 100
		}
	}
	return pp, span, true
}

// applyOllamaRateRation paces the weekly gauge on observed consumption.
//
// It only ever makes the verdict STRICTER. The 95% gauge cutoff already decided its own
// cases before this runs, and a ration must not launder a critical gauge into "ok".
//
// The weekly gauge alone is paced, matching rationApplies: the 5-hour window is a refill
// rate the provider enforces itself, and pacing it at 10%/day would cap each of the ~4.8
// daily windows at ~2% and idle the fleet for a limit nobody imposed.
func applyOllamaRateRation(o OllamaQuotaObservation, history []ollamaObservation, now time.Time) OllamaQuotaObservation {
	if o.State != "ok" {
		return o
	}
	pp, span, ok := ollamaConsumedLast24h(history, now)
	if !ok {
		o.Reason += "; daily ration UNPACED — needs 2+ gauge readings spanning 1h+"
		return o
	}
	allowance := 100 * DailyRationFraction
	if pp > allowance {
		o.State = "over"
		o.GaugeStatus = "RATION"
		o.Reason = "Ollama Cloud weekly gauge consumed " +
			trimFloat(pp) + "pp in the last " + span.Round(time.Minute).String() +
			", over the " + trimFloat(allowance) + "pp/day ration; new cloud routing blocked"
		return o
	}
	o.Reason += "; daily ration ok (" + trimFloat(pp) + "pp of " +
		trimFloat(allowance) + "pp per day over " + span.Round(time.Minute).String() + ")"
	return o
}

// trimFloat renders a percentage without trailing noise.
//
// The zero-trim applies ONLY to a fractional part. Trimming unconditionally turned the
// 10pp/day allowance into "1pp" in the operator-facing ration line — the comparison used the
// float and stayed correct, so the guard behaved while the number it reported was wrong by
// 10x. A report that misstates the budget is how the next reader mis-tunes it.
func trimFloat(f float64) string {
	s := formatFloat(f)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

func formatFloat(f float64) string {
	b, err := json.Marshal(roundTo(f, 1))
	if err != nil {
		return "0"
	}
	return string(b)
}

func roundTo(f float64, places int) float64 {
	mult := 1.0
	for i := 0; i < places; i++ {
		mult *= 10
	}
	return float64(int64(f*mult+0.5)) / mult
}
