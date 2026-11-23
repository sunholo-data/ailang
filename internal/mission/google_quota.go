package mission

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

// The google bucket: the Google AI subscription the agy executor runs on
// (internal/executor/agy, design_docs/planned/m-agy-executor.md).
//
// The instrument is agy itself. `agy -p /quota` under the executor's subscription
// HOME prints the plan's own usage with NO model call (measured 2026-11-23, agy
// 1.5.15, Google AI Ultra):
//
//	Gemini Models	Weekly Limit Remaining	97%	2026-11-30T14:09:17Z
//	Gemini Models	Five Hour Limit Remaining	95%	2026-11-23T19:09:17Z
//	Claude and GPT models	Weekly Limit Remaining	100%	2026-11-30T14:21:28Z
//	Claude and GPT models	Five Hour Limit Remaining	100%	2026-11-23T19:21:28Z
//
// The columns are percent REMAINING; used = 100 - remaining. Only the "Gemini Models"
// rows ration the bucket (that is what agy: lanes spend); the Claude/GPT rows are
// reported, never paced.
//
// PACE (Mark, attended 2026-11-23: "15% a day similar to the other agent subs"): the
// weekly window's pace line is AILANG_GOOGLE_DAILY_PACE (default 15) percent per
// CALENDAR day since the window started, capped at 100 — so a week spends at most
// ~100% by day 7 with no weekday/weekend split. The five-hour window is the
// provider's own refill rate and is not paced; it blocks only when exhausted, as
// for Codex. Usage is ACCOUNT usage (attended agy on the same account counts).
//
// Fails CLOSED like Codex and Ollama: a quota that cannot be read blocks new google
// routing, because the alternative is spending a subscription we cannot see.

// googleBucket is the ration bucket name, matching _mc_rung_bucket's agy:* arm.
const googleBucket = "google"

// GoogleQuotaObservation is the subscription's own reading, paced.
type GoogleQuotaObservation struct {
	State      string             `json:"state"`
	Reason     string             `json:"reason"`
	ObservedAt time.Time          `json:"observed_at,omitempty"`
	Home       string             `json:"home,omitempty"`
	PacePerDay float64            `json:"pace_percent_per_day,omitempty"`
	Windows    []CodexQuotaWindow `json:"windows,omitempty"`
	// Other carries rows for model families the google bucket does not pace.
	Other         []GoogleQuotaRow `json:"other,omitempty"`
	HeadroomShort bool             `json:"headroom_short,omitempty"`
}

// GoogleQuotaRow is one line of `agy -p /quota`.
type GoogleQuotaRow struct {
	Family           string    `json:"family"`
	Window           string    `json:"window"`
	RemainingPercent float64   `json:"remaining_percent"`
	ResetsAt         time.Time `json:"resets_at"`
}

// Blocked reports whether new google routing must wait.
func (o GoogleQuotaObservation) Blocked() bool { return o.State != "ok" }

// agyQuotaRunner runs `agy -p /quota` and returns its combined output. A variable so
// tests replay fixtures instead of calling the network.
var agyQuotaRunner = runAgyQuota

const agyQuotaTimeout = 45 * time.Second

func runAgyQuota(agyPath, home string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), agyQuotaTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, agyPath, "--print-timeout", "30s", "-p", "/quota")
	cmd.Dir = os.TempDir()
	// Minimal env: the subscription HOME, no API key (a GEMINI_API_KEY would switch agy
	// to the key route, whose /quota exits 2), and never the self-updater.
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + home, "AGY_CLI_DISABLE_AUTO_UPDATE=true"}
	cmd.Stdin = nil
	cmd.WaitDelay = 2 * time.Second
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return out, fmt.Errorf("agy /quota timed out after %v", agyQuotaTimeout)
	}
	return out, err
}

// GoogleSubscriptionHome is where the agy executor's subscription route keeps its
// sign-in, for an operator home.
func GoogleSubscriptionHome(home string) string {
	return filepath.Join(config.AgyHomeRootFor(home), "subscription")
}

func findAgy(home string) string {
	if p, err := exec.LookPath("agy"); err == nil {
		return p
	}
	return filepath.Join(home, ".local", "bin", "agy")
}

// ObserveGoogleQuota reads and paces the Google AI subscription's quota.
func ObserveGoogleQuota(paths Paths, now time.Time) GoogleQuotaObservation {
	sub := GoogleSubscriptionHome(paths.Home)
	o := GoogleQuotaObservation{State: "unknown", Home: sub}
	pace, err := config.GoogleDailyPace()
	if err != nil {
		o.Reason = err.Error() + "; new google routing blocked"
		return o
	}
	o.PacePerDay = pace
	agy := findAgy(paths.Home)
	if _, err := os.Stat(agy); err != nil {
		o.Reason = "agy CLI not installed (make install-agy); google quota unreadable, new google routing blocked"
		return o
	}
	if _, err := os.Stat(sub); err != nil {
		o.Reason = "no agy subscription HOME at " + sub + " (make agy-signin); new google routing blocked"
		return o
	}
	out, runErr := agyQuotaRunner(agy, sub)
	return evaluateGoogleQuota(out, runErr, sub, pace, now)
}

// evaluateGoogleQuota parses one `agy -p /quota` reading and paces it.
func evaluateGoogleQuota(out []byte, runErr error, home string, pace float64, now time.Time) GoogleQuotaObservation {
	o := GoogleQuotaObservation{State: "unknown", Home: home, PacePerDay: pace, ObservedAt: now}
	lower := strings.ToLower(string(out))
	if strings.Contains(lower, "sign in") || strings.Contains(lower, "authentication") {
		o.Reason = "agy is not signed in for the subscription route (make agy-signin); new google routing blocked"
		return o
	}
	rows, perr := parseAgyQuota(out)
	if runErr != nil && len(rows) == 0 {
		o.Reason = fmt.Sprintf("agy /quota failed: %v: %s; new google routing blocked", runErr, strings.TrimSpace(googleTail(string(out), 200)))
		return o
	}
	if perr != nil {
		o.Reason = "agy /quota output unparseable (" + perr.Error() + "); new google routing blocked"
		return o
	}
	var weekly, fiveHour *GoogleQuotaRow
	for i := range rows {
		r := &rows[i]
		if r.Family != "Gemini Models" {
			o.Other = append(o.Other, *r)
			continue
		}
		switch {
		case strings.HasPrefix(r.Window, "Weekly"):
			weekly = r
		case strings.HasPrefix(r.Window, "Five Hour"):
			fiveHour = r
		}
	}
	if weekly == nil {
		o.Reason = "agy /quota has no Gemini weekly row; new google routing blocked"
		return o
	}
	margin := StartMargin(googleBucket)
	addWindow := func(r *GoogleQuotaRow, minutes int64, paced bool) CodexQuotaWindow {
		w := CodexQuotaWindow{UsedPercent: 100 - r.RemainingPercent, WindowMinutes: minutes, ResetsAt: r.ResetsAt, AllowancePercent: 100}
		if paced {
			start := r.ResetsAt.Add(-time.Duration(minutes) * time.Minute)
			days := now.Sub(start).Hours() / 24
			if days < 0 {
				days = 0
			}
			w.AllowancePercent = pace * days
			if w.AllowancePercent > 100 {
				w.AllowancePercent = 100
			}
		}
		o.Windows = append(o.Windows, w)
		return w
	}
	wk := addWindow(weekly, 7*24*60, true)
	var fh *CodexQuotaWindow
	if fiveHour != nil {
		w := addWindow(fiveHour, 5*60, false)
		fh = &w
	}
	for _, w := range o.Windows {
		if !w.ResetsAt.After(now) {
			o.State = "expired"
			o.Reason = "agy reported a google window whose reset is already past; re-read before routing"
			return o
		}
	}
	switch {
	case wk.UsedPercent >= 100 || (fh != nil && fh.UsedPercent >= 100):
		o.State = "over"
		o.Reason = "Google AI subscription window exhausted"
	case wk.UsedPercent > wk.AllowancePercent:
		o.State = "over"
		o.Reason = fmt.Sprintf("Google weekly usage %.1f%% exceeds the %.0f%%/day pace line %.1f%%", wk.UsedPercent, pace, wk.AllowancePercent)
	case wk.UsedPercent > 0 && wk.AllowancePercent-wk.UsedPercent < margin:
		// Same rule as Codex: an UNUSED window takes no margin, or nothing could start it.
		o.State = "over"
		o.HeadroomShort = true
		o.Reason = headroomShortReason("Google", "pp", wk.AllowancePercent-wk.UsedPercent, margin)
	default:
		o.State = "ok"
		o.Reason = fmt.Sprintf("Google AI subscription usage %.1f%% within the %.0f%%/day pace line %.1f%%", wk.UsedPercent, pace, wk.AllowancePercent)
	}
	return o
}

// parseAgyQuota reads the TSV lines of `agy -p /quota`; other lines are ignored. A
// line with the right shape but a bad field is an error: a half-read could hide the
// window that is over.
func parseAgyQuota(out []byte) ([]GoogleQuotaRow, error) {
	var rows []GoogleQuotaRow
	for _, line := range bytes.Split(out, []byte{'\n'}) {
		f := strings.Split(strings.TrimRight(string(line), "\r"), "\t")
		if len(f) != 4 || !strings.HasSuffix(strings.TrimSpace(f[2]), "%") {
			continue
		}
		pct, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(f[2]), "%"), 64)
		if err != nil || pct < 0 || pct > 100 {
			return nil, fmt.Errorf("bad percent %q", f[2])
		}
		reset, err := time.Parse(time.RFC3339, strings.TrimSpace(f[3]))
		if err != nil {
			return nil, fmt.Errorf("bad reset %q", f[3])
		}
		rows = append(rows, GoogleQuotaRow{Family: strings.TrimSpace(f[0]), Window: strings.TrimSpace(f[1]), RemainingPercent: pct, ResetsAt: reset.UTC()})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no quota rows")
	}
	return rows, nil
}

func googleTail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
