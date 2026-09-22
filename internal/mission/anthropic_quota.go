package mission

// Anthropic subscription quota, read from the same endpoint the Claude Code UI reads.
//
// Until now `anthropic` was the fleet's loudest blind spot: the token ledger counted what
// WE spent (1.38M tokens on the long window, 2026-09-08) but had no capacity to pace it
// against, so every report said `capacity-unknown — UNRATIONED`. That is not a missing
// probe; it is a missing NUMBER, and the number exists — GET /api/oauth/usage returns the
// provider's own utilisation percentage and reset time for both windows.
//
// This reader is deliberately shaped like codex_quota.go rather than ollama_quota.go: the
// response carries `resets_at`, so the existing percentage pacing applies unchanged and no
// capacity has to be inferred, invented or configured. Ollama is the one provider that
// still cannot do this, because its gauge has no reset.
//
// ENFORCEMENT IS ON BY DEFAULT (Mark, attended 2026-09-08). AILANG_ANTHROPIC_RATION=0 is an
// attended operator's escape hatch for one process, not a fleet setting.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

// anthropicUsageURL is the endpoint behind the UI's usage display.
const anthropicUsageURL = "https://api.anthropic.com/api/oauth/usage"

// anthropicKeychainService is where Claude Code stores its OAuth credential on macOS.
const anthropicKeychainService = "Claude Code-credentials"

// AnthropicQuotaObservation is provider-reported subscription utilisation.
//
// It reuses CodexQuotaWindow so that one pacing rule, and one set of tests, govern every
// provider that reports a percentage against a window with a reset.
type AnthropicQuotaObservation struct {
	State      string             `json:"state"`
	Reason     string             `json:"reason"`
	ObservedAt time.Time          `json:"observed_at,omitempty"`
	Source     string             `json:"source,omitempty"`
	Enforced   bool               `json:"enforced"`
	Windows    []CodexQuotaWindow `json:"windows,omitempty"`
}

// Blocked reports whether new Anthropic routing should be refused.
//
// An observation that is over ration but NOT enforced is reported and admitted. The state
// still says "over" — the number must not be laundered into "ok" just because nothing acts
// on it yet, or the ruling would be taken on a false reading.
func (o AnthropicQuotaObservation) Blocked() bool {
	if !o.Enforced {
		return false
	}
	return o.State != "ok"
}

// AnthropicRationEnabled reports whether the Anthropic ration gates routing.
//
// Default ON (Mark, attended 2026-09-08: "really I just want it standard on everything").
// The ration paces UNATTENDED mission loops so ATTENDED sessions keep headroom, so the
// provider is the wrong place to make it opt-in — a bucket nobody paces is exactly the
// state that let ollama run at five times its ration for sixteen hours.
//
// It was briefly written opt-in on the fear that gating Anthropic would wedge the fleet,
// since that is where the controller lives. That fear was wrong twice over: the controller
// walks CONTROLLER_FALLBACK to a cheaper rung on an over-ration verdict, and when nothing
// is left the existing "NO usable controller" refusal is the pause D-4 asks for — announced
// once per episode, zero tokens beyond probes.
//
// AILANG_ANTHROPIC_RATION=0 opts out for one process. It is an escape hatch for an attended
// operator, not a fleet setting; unset means rationed.
func AnthropicRationEnabled() bool { return config.AnthropicRation() }

// anthropicOAuthToken returns the subscription OAuth token, or "".
//
// The env var wins so an operator (and the tests) can supply one explicitly. The keychain
// read is the fallback and is bounded: `security` prompts on a locked keychain, and an
// unbounded prompt inside a launchd fire would hang the whole iteration.
//
// NOTE the asymmetry that makes this work at all: launchd jobs have keychain access, plain
// shells often do not. That is why this resolves at RUN time in the mission fire rather
// than being captured into config by a human session.
//
// CLAUDE_CODE_OAUTH_TOKEN IS NOT A SUBSTITUTE FOR THE KEYCHAIN HERE, and the env
// var winning is a hazard rather than a convenience. Measured 2026-09-22: a token
// from `claude setup-token` authenticates but is FORBIDDEN from the usage
// endpoint — HTTP 403, not 401 — while the keychain credential read 12.0%/11.4%
// in the same minute. Because the env var is preferred, storing one in
// secrets.env does not sit inert: it OVERRIDES a working keychain read with an
// unreadable bucket, which `mission quota --over` then blocks by policy.
//
// The keychain item carries scopes user:file_upload, user:inference,
// user:mcp_servers, user:profile, user:sessions:claude_code; a setup-token token
// evidently carries a narrower set. So the env var remains the documented escape
// hatch for a token that CAN read usage, and is a footgun for one that cannot —
// verify against the endpoint before storing one, never assume.
func anthropicOAuthToken(ctx context.Context) string {
	// LookupEnv, not Getenv: an explicitly EMPTY CLAUDE_CODE_OAUTH_TOKEN means "no
	// credential" and must not fall through to the keychain. That is the same seam
	// OLLAMA_API_KEY="" gives the Ollama reader, and without it a test — or an operator
	// trying to observe the unauthenticated path — silently gets the login keychain and a
	// verdict that depends on the real account's live quota.
	if t, ok := config.ClaudeCodeOAuthToken(); ok {
		return t
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "security", "find-generic-password",
		"-s", anthropicKeychainService, "-w").Output()
	if err != nil {
		return ""
	}
	var cred struct {
		ClaudeAIOauth struct {
			AccessToken string `json:"accessToken"`
			ExpiresAt   int64  `json:"expiresAt"`
		} `json:"claudeAiOauth"`
		AccessToken string `json:"accessToken"`
	}
	if json.Unmarshal(out, &cred) != nil {
		return ""
	}
	if cred.ClaudeAIOauth.AccessToken != "" {
		// Record the stored token's expiry so a STALE credential can be named as
		// such. Claude Code refreshes its access token in memory and does not write
		// the fresh one back here, so this blob goes stale while the app keeps
		// working — which is exactly how the fleet ended up reading a week-old
		// token, getting HTTP 401, and reporting the bucket as unmeasurable.
		//
		// Measured 2026-09-22: expiresAt was 2026-09-15, seven days earlier, while
		// the neighbouring refreshToken was valid for another fourteen. Anthropic
		// sat at ~89% free and three World iterations were routed away from it.
		lastKeychainExpiry = cred.ClaudeAIOauth.ExpiresAt
		return cred.ClaudeAIOauth.AccessToken
	}
	return cred.AccessToken
}

// lastKeychainExpiry is the expiry (unix millis) of the access token most
// recently read from the keychain, or 0 when it was unset or the token came from
// the environment. It exists only to turn a bare HTTP 401 into a sentence an
// operator can act on; it is never used to decide whether to make the call,
// because a clock skew must not be able to suppress a reading that would work.
var lastKeychainExpiry int64

// keychainTokenStaleness returns a human clause when the stored token has
// expired, and "" otherwise.
func keychainTokenStaleness(now time.Time) string {
	if lastKeychainExpiry <= 0 {
		return ""
	}
	exp := time.UnixMilli(lastKeychainExpiry)
	if !exp.Before(now) {
		return ""
	}
	return fmt.Sprintf(" — the keychain access token EXPIRED %s (%s ago); "+
		"Claude Code refreshes in memory without writing back, so re-authenticate it "+
		"(the stored refreshToken is not used by this reader)",
		exp.UTC().Format("2006-01-02"), now.Sub(exp).Round(time.Hour))
}

// ObserveAnthropicQuota makes one bounded usage-only call. It never makes an inference call.
func ObserveAnthropicQuota(now time.Time) AnthropicQuotaObservation {
	client := &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	o := observeAnthropicQuota(anthropicOAuthToken(context.Background()), now, client)
	if o.State != "unknown" {
		return o
	}
	// FALL BACK TO THE CLI when the endpoint could not be read.
	//
	// Every credential route to the HTTP endpoint has failed in a different way
	// (see anthropic_usage_cli.go for the measurements), and the consequence was a
	// week of routing away from the fleet's largest allocation. `claude -p /usage`
	// carries its own live credential — the same one that makes `claude -p` work for
	// inference — so it answers when the endpoint cannot.
	//
	// Second, not first, only because it is slower: the endpoint returns in ~5s and
	// the CLI takes tens of seconds, and this runs on every mission fire. When the
	// endpoint works, nothing changes.
	windows, err := anthropicCLIUsage(context.Background(), now)
	if err != nil {
		o.Reason += fmt.Sprintf("; CLI fallback also failed (%v)", err)
		return o
	}
	o.Windows = windows
	o.Source = "claude -p /usage"
	o.State = "ok"
	o.Reason = "subscription usage read from `claude -p /usage` (the HTTP endpoint was unreadable)"
	evaluateAnthropicQuota(&o, now)
	return o
}

func observeAnthropicQuota(token string, now time.Time, client *http.Client) AnthropicQuotaObservation {
	o := AnthropicQuotaObservation{
		State:      "unknown",
		Reason:     "Anthropic subscription usage unavailable",
		ObservedAt: now,
		Source:     anthropicUsageURL,
		Enforced:   AnthropicRationEnabled(),
	}
	if token == "" {
		o.Reason = "no Anthropic OAuth credential (CLAUDE_CODE_OAUTH_TOKEN or login keychain); cannot read subscription usage"
		return o
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, anthropicUsageURL, nil)
	if err != nil {
		return o
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	resp, err := client.Do(req)
	if err != nil {
		o.Reason = "Anthropic usage request failed or timed out"
		return o
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		o.Reason = fmt.Sprintf("Anthropic usage endpoint returned HTTP %d%s", resp.StatusCode,
			keychainTokenStaleness(now))
		return o
	}
	const limit = 1 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || len(data) > limit {
		o.Reason = "Anthropic usage response unreadable or exceeds 1 MiB"
		return o
	}
	parsed := parseAnthropicUsage(data, now)
	parsed.Source = o.Source
	parsed.Enforced = o.Enforced
	if len(parsed.Windows) == 0 {
		return parsed
	}
	evaluateAnthropicQuota(&parsed, now)
	return parsed
}

// anthropicUsageBody is the subset of /api/oauth/usage this reader depends on.
//
// `limits` is the normalised array and is preferred; `five_hour`/`seven_day` are the older
// top-level objects and are the fallback. Both carry the same two numbers we need. Fields
// we do not use (spend, extra_usage, the scoped/experimental buckets) are ignored rather
// than rejected, so a new bucket appearing upstream does not turn into a fleet outage.
type anthropicUsageBody struct {
	FiveHour *anthropicWindowBody `json:"five_hour"`
	SevenDay *anthropicWindowBody `json:"seven_day"`
	Limits   []struct {
		Kind     string     `json:"kind"`
		Group    string     `json:"group"`
		Percent  *float64   `json:"percent"`
		Severity string     `json:"severity"`
		ResetsAt *time.Time `json:"resets_at"`
		Scope    *struct {
			Model *struct {
				DisplayName string `json:"display_name"`
			} `json:"model"`
		} `json:"scope"`
	} `json:"limits"`
}

type anthropicWindowBody struct {
	Utilization  *float64   `json:"utilization"`
	ResetsAt     *time.Time `json:"resets_at"`
	LockedReason *string    `json:"locked_reason"`
}

func parseAnthropicUsage(data []byte, now time.Time) AnthropicQuotaObservation {
	o := AnthropicQuotaObservation{
		State:      "unknown",
		Reason:     "unsupported or malformed Anthropic usage response",
		ObservedAt: now,
	}
	var body anthropicUsageBody
	if json.Unmarshal(data, &body) != nil {
		return o
	}

	// Prefer the normalised array. Only unscoped windows ration the fleet: a
	// `weekly_scoped` entry caps ONE model (Fable, 25% on 2026-09-08) and treating it as
	// the account limit would block every lane for a ceiling that binds one of them.
	for _, l := range body.Limits {
		if l.Percent == nil || l.ResetsAt == nil || l.Scope != nil {
			continue
		}
		var minutes int64
		switch l.Kind {
		case "session":
			minutes = 300
		case "weekly_all":
			minutes = 7 * 24 * 60
		default:
			continue
		}
		o.Windows = append(o.Windows, CodexQuotaWindow{
			UsedPercent: *l.Percent, WindowMinutes: minutes, ResetsAt: l.ResetsAt.UTC(),
		})
	}
	if len(o.Windows) == 0 {
		for _, w := range []struct {
			body    *anthropicWindowBody
			minutes int64
		}{{body.FiveHour, 300}, {body.SevenDay, 7 * 24 * 60}} {
			if w.body == nil || w.body.Utilization == nil || w.body.ResetsAt == nil {
				continue
			}
			o.Windows = append(o.Windows, CodexQuotaWindow{
				UsedPercent: *w.body.Utilization, WindowMinutes: w.minutes, ResetsAt: w.body.ResetsAt.UTC(),
			})
		}
	}
	if len(o.Windows) == 0 {
		o.Reason = "Anthropic usage response carried no usable window; new Anthropic routing not paced"
		return o
	}

	// A locked window is the provider saying no, whatever the percentage reads.
	for _, w := range []*anthropicWindowBody{body.FiveHour, body.SevenDay} {
		if w != nil && w.LockedReason != nil && *w.LockedReason != "" {
			o.State = "over"
			o.Reason = "Anthropic subscription window is locked: " + *w.LockedReason
			return o
		}
	}
	o.Reason = "provider-reported Anthropic utilisation recorded"
	return o
}

// evaluateAnthropicQuota applies the fleet ration to provider percentages.
//
// It delegates to the Codex evaluator so there is ONE pacing rule. Only the public reason
// is rewritten, because a report naming the wrong provider is how an operator ends up
// investigating the wrong bucket.
func evaluateAnthropicQuota(o *AnthropicQuotaObservation, now time.Time) {
	if o.State == "over" {
		return // a locked window is already decided
	}
	verdict := CodexQuotaObservation{ObservedAt: now, Windows: o.Windows}
	verdict.evaluate(now)
	o.State = verdict.State
	o.Windows = verdict.Windows
	switch o.State {
	case "ok":
		o.Reason = "provider-reported Anthropic utilisation is within ration"
	case "expired":
		o.Reason = "Anthropic provider window expired; refresh required before routing"
	default:
		o.Reason = "provider-reported Anthropic utilisation exceeds ration or exhausts a window"
	}
	if !o.Enforced && o.State != "ok" {
		o.Reason += " (NOT ENFORCED — AILANG_ANTHROPIC_RATION=0 is set for this process)"
	}
}
