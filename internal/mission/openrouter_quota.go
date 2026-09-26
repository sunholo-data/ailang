package mission

// OpenRouter quota — the fourth provider, routable through pi in every role chain, and until
// now the ONLY one with no row in the quota report at all.
//
// Not "unrationed and loud", which is the documented safe state: absent. `ailang mission quota`
// printed codex, ollama and anthropic and simply said nothing about openrouter, while
// pi:openrouter/* sat in the planner, executor and designer fallback chains as the metered rung
// behind ollama. Measured 2026-09-08, with codex stale, ollama at 86.8% weekly and anthropic
// over ration, openrouter was at 7.4% of its monthly cap with $92.63 left — the one lane with
// real headroom, and the one nothing was watching.
//
// This reader is the EASIEST of the three, because the provider supplies everything the other
// two withhold. GET /api/v1/key returns a hard dollar `limit`, the `limit_reset` period, and
// usage already bucketed by day, week and month. Compare:
//
//	codex       percentages + reset  -> window-position pacing
//	anthropic   percentages + reset  -> same pacing, reused unchanged
//	ollama      fractions, NO reset  -> had to ration on observed RATE, banking readings
//	openrouter  dollars + daily      -> rate pacing with NO banking; the provider counts for us
//
// So there is no observations file here and no reset arithmetic: `usage_daily` IS the trailing
// day, straight from the provider.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// openRouterKeyURL reports usage and limit for the calling key. Inference-only keys can read
// it; /api/v1/activity 403s on this account, which is why this endpoint and not that one.
const openRouterKeyURL = "https://openrouter.ai/api/v1/key"

// openRouterMonthDays is the pacing denominator for a monthly cap.
//
// 30, not 28 or 31: a fixed divisor keeps the daily allowance stable across months, and the
// error either way is smaller than the 30% reserve below absorbs.
const openRouterMonthDays = 30.0

// OpenRouterQuotaObservation is provider-reported spend against a stated cap.
type OpenRouterQuotaObservation struct {
	State        string    `json:"state"`
	Reason       string    `json:"reason"`
	ObservedAt   time.Time `json:"observed_at,omitempty"`
	Source       string    `json:"source,omitempty"`
	LimitUSD     *float64  `json:"limit_usd,omitempty"`
	RemainingUSD *float64  `json:"remaining_usd,omitempty"`
	UsedDayUSD   *float64  `json:"used_day_usd,omitempty"`
	UsedMonthUSD *float64  `json:"used_month_usd,omitempty"`
	AllowanceUSD *float64  `json:"allowance_usd,omitempty"`
	LimitReset   string    `json:"limit_reset,omitempty"`
}

// Blocked reports whether new OpenRouter routing should be refused.
func (o OpenRouterQuotaObservation) Blocked() bool { return o.State != "ok" }

// ObserveOpenRouterQuota makes one bounded usage-only call; it never makes an inference call.
func ObserveOpenRouterQuota(key string, now time.Time) OpenRouterQuotaObservation {
	client := &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return observeOpenRouterQuota(key, now, client)
}

func observeOpenRouterQuota(key string, now time.Time, client *http.Client) OpenRouterQuotaObservation {
	o := OpenRouterQuotaObservation{
		State: "unknown", Reason: "OpenRouter quota unavailable; new OpenRouter routing blocked",
		ObservedAt: now, Source: openRouterKeyURL,
	}
	if key == "" {
		o.Reason = "OPENROUTER_API_KEY missing; cannot read OpenRouter quota"
		return o
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openRouterKeyURL, nil)
	if err != nil {
		return o
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := client.Do(req)
	if err != nil {
		o.Reason = "OpenRouter key request failed or timed out"
		return o
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		o.Reason = fmt.Sprintf("OpenRouter key endpoint returned HTTP %d", resp.StatusCode)
		return o
	}
	const limit = 1 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || len(data) > limit {
		o.Reason = "OpenRouter key response unreadable or exceeds 1 MiB"
		return o
	}
	return evaluateOpenRouterQuota(parseOpenRouterKey(data, now), now)
}

type openRouterKeyBody struct {
	Data struct {
		Limit          *float64 `json:"limit"`
		LimitRemaining *float64 `json:"limit_remaining"`
		LimitReset     string   `json:"limit_reset"`
		UsageDaily     *float64 `json:"usage_daily"`
		UsageMonthly   *float64 `json:"usage_monthly"`
	} `json:"data"`
}

func parseOpenRouterKey(data []byte, now time.Time) OpenRouterQuotaObservation {
	o := OpenRouterQuotaObservation{
		State: "unknown", Reason: "unsupported or malformed OpenRouter key response", ObservedAt: now,
	}
	var body openRouterKeyBody
	if json.Unmarshal(data, &body) != nil {
		return o
	}
	d := body.Data
	// An UNCAPPED key reports limit null. That is not an error and not "ok" either: there is
	// no denominator, so nothing can be paced. Say so rather than inventing a cap or waving
	// the bucket through — the same call the ollama reader makes about a missing capacity.
	if d.Limit == nil || *d.Limit <= 0 {
		o.Reason = "OpenRouter key reports no spend limit; nothing to pace against"
		return o
	}
	if d.UsageDaily == nil {
		o.Reason = "OpenRouter key response omits usage_daily; cannot pace"
		return o
	}
	o.LimitUSD, o.RemainingUSD, o.UsedDayUSD, o.UsedMonthUSD = d.Limit, d.LimitRemaining, d.UsageDaily, d.UsageMonthly
	o.LimitReset = d.LimitReset
	o.Reason = "provider-reported OpenRouter spend recorded"
	return o
}

// evaluateOpenRouterQuota paces daily spend against a share of the monthly cap.
//
// The allowance derives from DailyRationFraction rather than a new constant. That fraction is
// 10%/day of a SEVEN-day bucket, i.e. it spends 70% over the window and holds 30% back — the
// reserve is the point, and the code comment on D-1 says so. Generalised to a monthly cap the
// same reserve gives 70% over ~30 days: $100 * 0.7 / 30 = $2.33/day.
//
// Rate, not window position, because `limit_reset: "monthly"` names the PERIOD and not the
// date. The provider counts usage_daily for us, so unlike the ollama reader this needs no
// banked observations to compute one.
func evaluateOpenRouterQuota(o OpenRouterQuotaObservation, _ time.Time) OpenRouterQuotaObservation {
	if o.LimitUSD == nil || o.UsedDayUSD == nil {
		return o
	}
	reserveKept := DailyRationFraction * 7 // 0.70 — the weekly rule's spend share
	allowance := *o.LimitUSD * reserveKept / openRouterMonthDays
	o.AllowanceUSD = &allowance

	// A spent cap blocks regardless of today's rate: remaining is the harder bound.
	if o.RemainingUSD != nil && *o.RemainingUSD <= 0 {
		o.State = "over"
		o.Reason = "OpenRouter monthly limit is exhausted; new OpenRouter routing blocked"
		return o
	}
	if *o.UsedDayUSD > allowance {
		o.State = "over"
		o.Reason = fmt.Sprintf("OpenRouter spent $%.2f today, over the $%.2f/day ration (%.0f%% of a $%.2f monthly cap); new OpenRouter routing blocked",
			*o.UsedDayUSD, allowance, reserveKept*100, *o.LimitUSD)
		return o
	}
	o.State = "ok"
	o.Reason = fmt.Sprintf("OpenRouter within ration: $%.2f of $%.2f today, $%.2f of $%.2f left this month",
		*o.UsedDayUSD, allowance, orZero(o.RemainingUSD), *o.LimitUSD)
	return o
}

func orZero(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
