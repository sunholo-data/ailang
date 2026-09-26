package mission

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Reading Anthropic subscription usage by ASKING THE CLI, not the HTTP endpoint.
//
// The endpoint path needs a credential this fleet does not control, and every way
// of supplying one has failed in a different way — measured 2026-09-22:
//
//	keychain item     ACL names Claude Code, so the `security` shell-out is denied
//	                  (rc=36); when readable, its access token expires every ~8h and
//	                  Claude Code refreshes it IN MEMORY without writing back, so the
//	                  stored copy was a week stale and returned HTTP 401.
//	setup-token       authenticates but is FORBIDDEN from the usage endpoint —
//	                  HTTP 403 — and because CLAUDE_CODE_OAUTH_TOKEN is PREFERRED,
//	                  storing one overrode a working keychain read with a broken one.
//
// The cost of that was a week of the fleet routing away from its largest allocation
// while the subscription sat at ~12% consumed, and three World iterations that
// descended to a hung pi because Anthropic and codex both read as blocked.
//
// `claude -p /usage` has worked throughout. The CLI holds its own live credential —
// the same one that makes `claude -p` work for inference — so asking it sidesteps
// the credential problem entirely rather than solving it. It prints exactly the two
// windows the ration needs:
//
//	Current session: 12% used · resets Sep 22 at 2:30pm (Europe/Copenhagen)
//	Current week (all models): 14% used · resets Sep 28 at 7am (Europe/Copenhagen)
//
// The trade is parsing human text instead of JSON, so every parse failure is loud
// and returns "not observed" rather than a zero that would read as "plenty left".

var (
	anthropicCLISessionRe = regexp.MustCompile(`Current session:\s*(\d+)%\s*used\s*·\s*resets\s*([^(]+?)\s*\(([^)]+)\)`)
	anthropicCLIWeekRe    = regexp.MustCompile(`Current week \(all models\):\s*(\d+)%\s*used\s*·\s*resets\s*([^(]+?)\s*\(([^)]+)\)`)
)

// anthropicCLIUsage runs `claude -p /usage` and returns the two windows.
//
// stdin is /dev/null: the CLI waits on stdin otherwise, which is the same defect
// that hung the pi controller for three iterations.
func anthropicCLIUsage(ctx context.Context, now time.Time) ([]CodexQuotaWindow, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "claude", "-p", "/usage")
	cmd.Stdin = nil // exec gives a nil Stdin /dev/null, which EOFs immediately
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("claude -p /usage: %w", err)
	}
	return parseAnthropicCLIUsage(string(out), now)
}

func parseAnthropicCLIUsage(text string, now time.Time) ([]CodexQuotaWindow, error) {
	var windows []CodexQuotaWindow
	add := func(re *regexp.Regexp, minutes int, label string) error {
		m := re.FindStringSubmatch(text)
		if m == nil {
			return fmt.Errorf("no %q line in `claude -p /usage` output", label)
		}
		pct, err := strconv.Atoi(m[1])
		if err != nil {
			return fmt.Errorf("%s: unparsable percent %q", label, m[1])
		}
		reset, err := parseAnthropicCLIReset(m[2], m[3], now)
		if err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		windows = append(windows, CodexQuotaWindow{
			WindowMinutes: int64(minutes),
			UsedPercent:   float64(pct),
			ResetsAt:      reset,
		})
		return nil
	}
	if err := add(anthropicCLISessionRe, 300, "Current session"); err != nil {
		return nil, err
	}
	if err := add(anthropicCLIWeekRe, 10080, "Current week (all models)"); err != nil {
		return nil, err
	}
	return windows, nil
}

// parseAnthropicCLIReset turns "Sep 28 at 7am" + "Europe/Copenhagen" into a time.
//
// The CLI prints no YEAR, so it is inferred: the reset is always in the future, and
// a parsed date more than a day behind `now` means the year rolled. Inferring
// forward rather than assuming the current year is what keeps this correct across
// 31 December.
func parseAnthropicCLIReset(when, zone string, now time.Time) (time.Time, error) {
	loc, err := time.LoadLocation(strings.TrimSpace(zone))
	if err != nil {
		return time.Time{}, fmt.Errorf("unknown timezone %q: %w", zone, err)
	}
	clean := strings.Join(strings.Fields(strings.ReplaceAll(when, " at ", " ")), " ")
	var parsed time.Time
	for _, layout := range []string{"Jan 2 3:04pm", "Jan 2 3pm", "Jan 2 15:04"} {
		if t, perr := time.ParseInLocation(layout, clean, loc); perr == nil {
			parsed = t
			break
		}
	}
	if parsed.IsZero() {
		return time.Time{}, fmt.Errorf("unparsable reset time %q", when)
	}
	y := now.In(loc).Year()
	reset := time.Date(y, parsed.Month(), parsed.Day(), parsed.Hour(), parsed.Minute(), 0, 0, loc)
	if reset.Before(now.Add(-24 * time.Hour)) {
		reset = reset.AddDate(1, 0, 0)
	}
	return reset, nil
}
