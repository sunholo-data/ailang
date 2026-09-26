package mission

// Codex account usage read directly from the provider, via the Codex CLI's app-server.
//
// Until 2026-09-24 the only Codex quota instrument was the session scan in codex_quota.go:
// the `rate_limits` record Codex writes into a rollout file during a real run. That scan is
// self-sealing. With no Codex run in 24h the bucket reads "unknown", "unknown" blocks new
// Codex routing, a blocked bucket runs nothing, and nothing ever writes the next rollout —
// so Codex stays dark until a human happens to use it. Measured 2026-09-24: the last
// rollout was 24h old and the gate said "no recent Codex provider observation", while the
// account was really at 59% of its week (over ration — the unknown was HIDING an over).
//
// `codex app-server` answers `account/rateLimits/read` with the same numbers the rollout
// records carry, on demand, with no model call and nothing spent (probe, 2026-09-24:
// usedPercent 59, windowDurationMins 10080, resetsAt 1790500432 — identical to the last
// rollout record). It is read FIRST; the session scan stays as the fallback, so a Codex
// upgrade that changes the protocol degrades to the old behaviour, never to "ok".

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// codexAppServerTimeout bounds the whole exchange. The app-server answers in ~1-2s; this
// runs on every mission fire, and a hung read must cost seconds, not the slot.
const codexAppServerTimeout = 20 * time.Second

// codexAppServerSource names the instrument in reports.
const codexAppServerSource = "codex app-server account/rateLimits/read"

// codexAppServerCall sends one JSON-RPC request to `codex app-server` and returns its
// raw `result`. It is a variable so tests never start a real codex process.
var codexAppServerCall = callCodexAppServer

// callCodexAppServer speaks the app-server's JSON-RPC over stdio: initialize, initialized,
// then METHOD as request id 2. stdin is held open until the answer arrives — closing it
// early lets the server exit before it replies. params is a JSON object or "".
func callCodexAppServer(ctx context.Context, codexHome, method, params string) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, codexAppServerTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "codex", "app-server")
	cmd.Env = append(os.Environ(), "CODEX_HOME="+codexHome)
	cmd.WaitDelay = 2 * time.Second
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("codex app-server: %w", err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	methodJSON, _ := json.Marshal(method)
	call := `{"jsonrpc":"2.0","id":2,"method":` + string(methodJSON)
	if params != "" {
		call += `,"params":` + params
	}
	requests := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"clientInfo":{"name":"ailang-mission-quota","version":"1"}}}
{"jsonrpc":"2.0","method":"initialized"}
` + call + "}\n"
	if _, err := io.WriteString(stdin, requests); err != nil {
		return nil, fmt.Errorf("codex app-server: write request: %w", err)
	}
	return scanCodexAppServerReply(ctx, stdout, method)
}

// scanCodexAppServerReply reads JSON-RPC lines until the reply to id 2.
func scanCodexAppServerReply(ctx context.Context, r io.Reader, method string) (json.RawMessage, error) {
	type reply struct {
		ID     *int            `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 4<<20)
	for sc.Scan() {
		var m reply
		if json.Unmarshal(sc.Bytes(), &m) != nil || m.ID == nil || *m.ID != 2 {
			continue // notifications, the initialize reply, non-JSON noise
		}
		if m.Error != nil {
			return nil, fmt.Errorf("codex app-server: %s: %s", method, m.Error.Message)
		}
		if len(m.Result) == 0 {
			return nil, fmt.Errorf("codex app-server: %s returned no result", method)
		}
		return m.Result, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("codex app-server: no %s reply within %s: %w", method, codexAppServerTimeout, err)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("codex app-server: read reply: %w", err)
	}
	return nil, fmt.Errorf("codex app-server exited without answering %s", method)
}

// parseCodexAppServerRateLimits turns an account/rateLimits/read result into an
// observation. Any shape it cannot fully validate is "unknown", never "ok".
func parseCodexAppServerRateLimits(result []byte, now time.Time) CodexQuotaObservation {
	type window struct {
		Used    *float64 `json:"usedPercent"`
		Minutes int64    `json:"windowDurationMins"`
		Reset   int64    `json:"resetsAt"`
	}
	type limits struct {
		LimitID              string  `json:"limitId"`
		Primary              *window `json:"primary"`
		Secondary            *window `json:"secondary"`
		SpendControlReached  *bool   `json:"spendControlReached"`
		RateLimitReachedType *string `json:"rateLimitReachedType"`
	}
	type credit struct {
		ID        string  `json:"id"`
		Title     *string `json:"title"`
		Status    string  `json:"status"`
		ResetType string  `json:"resetType"`
		ExpiresAt *int64  `json:"expiresAt"`
	}
	var body struct {
		OrdinaryUsageAllowed *bool             `json:"ordinaryUsageAllowed"`
		RateLimits           *limits           `json:"rateLimits"`
		ByLimitID            map[string]limits `json:"rateLimitsByLimitId"`
		ResetCredits         *struct {
			Available int64    `json:"availableCount"`
			Credits   []credit `json:"credits"`
		} `json:"rateLimitResetCredits"`
	}
	o := CodexQuotaObservation{State: "unknown", ObservedAt: now, Source: codexAppServerSource}
	if err := json.Unmarshal(result, &body); err != nil {
		o.Reason = "malformed Codex app-server rate-limit reply"
		return o
	}
	// Credits are recorded whatever the windows say: what is held in reserve is exactly
	// what an operator needs to know when the bucket is spent.
	if rc := body.ResetCredits; rc != nil {
		o.ResetCredits = &CodexResetCredits{Available: rc.Available}
		for _, c := range rc.Credits {
			if c.Status != "available" {
				continue
			}
			rec := CodexResetCredit{ID: c.ID, Status: c.Status}
			if c.Title != nil {
				rec.Title = *c.Title
			}
			if c.ExpiresAt != nil {
				rec.ExpiresAt = time.Unix(*c.ExpiresAt, 0).UTC()
			}
			o.ResetCredits.Credits = append(o.ResetCredits.Credits, rec)
		}
	}
	var l *limits
	if c, ok := body.ByLimitID["codex"]; ok {
		l = &c
	} else if body.RateLimits != nil && body.RateLimits.LimitID == "codex" {
		l = body.RateLimits
	}
	if l == nil {
		o.Reason = "Codex app-server reply carries no `codex` rate limit"
		return o
	}
	var raw []*codexRateWindow
	for _, w := range []*window{l.Primary, l.Secondary} {
		if w != nil {
			raw = append(raw, &codexRateWindow{Used: w.Used, Minutes: w.Minutes, Reset: w.Reset})
		}
	}
	windows, ok := codexValidWindows(raw, now)
	if !ok || len(windows) == 0 {
		o.Reason = "Codex app-server rate-limit windows failed validation"
		return o
	}
	o.Windows = windows
	o.evaluate(now)
	// The provider's own verdict outranks our arithmetic when it says stop.
	reached := l.RateLimitReachedType != nil ||
		(l.SpendControlReached != nil && *l.SpendControlReached) ||
		(body.OrdinaryUsageAllowed != nil && !*body.OrdinaryUsageAllowed)
	if reached && o.State == "ok" {
		o.State = "over"
		o.Reason = "Codex reports the account's rate limit or spend control reached"
	}
	if o.State != "ok" {
		o.Reason += o.ResetCredits.hint()
	}
	return o
}

// hint is appended to a blocked Codex reason so every surface that prints the reason — the
// quota report, `--over`, the driver's LANE DEGRADED notice on GitHub — also says what is
// held in reserve and how an ATTENDED operator spends it. Empty when nothing is held.
func (c *CodexResetCredits) hint() string {
	if c == nil || c.Available <= 0 {
		return ""
	}
	s := fmt.Sprintf("; %d Codex reset credit(s) in reserve", c.Available)
	if len(c.Credits) > 0 && !c.Credits[0].ExpiresAt.IsZero() {
		s += fmt.Sprintf(" (next expires %s)", c.Credits[0].ExpiresAt.Format("2006-01-02"))
	}
	return s + " — attended only: `ailang mission quota --codex-reset --yes`"
}

// ConsumeCodexResetCredit spends one Codex reset credit and returns the provider's
// outcome: "reset", "nothingToReset", "noCredit" or "alreadyRedeemed".
//
// OPERATOR ACTION ONLY (Mark, attended 2026-09-24: "leave that as a choice for attended
// sessions"). Nothing in the loop calls this; the CLI refuses inside a mission iteration.
// creditID "" lets the provider pick. The idempotency key makes a retried call safe: the
// same key never spends twice.
func ConsumeCodexResetCredit(ctx context.Context, codexHome, creditID, idempotencyKey string) (string, error) {
	if idempotencyKey == "" {
		return "", fmt.Errorf("consume Codex reset credit: idempotency key required")
	}
	params := map[string]any{"idempotencyKey": idempotencyKey}
	if creditID != "" {
		params["creditId"] = creditID
	}
	raw, _ := json.Marshal(params)
	result, err := codexAppServerCall(ctx, codexHome, "account/rateLimitResetCredit/consume", string(raw))
	if err != nil {
		return "", err
	}
	var r struct {
		Outcome string `json:"outcome"`
	}
	if err := json.Unmarshal(result, &r); err != nil || r.Outcome == "" {
		return "", fmt.Errorf("consume Codex reset credit: unreadable reply %s", string(result))
	}
	return r.Outcome, nil
}

// observeCodexAppServer reads Codex usage from the provider. ok=false means "not
// observed" — the caller falls back — and reason says why.
func observeCodexAppServer(codexHome string, now time.Time) (CodexQuotaObservation, bool, string) {
	// No credential, nothing to ask with. Checked before spawning so a missing login costs
	// a stat, not a process start on every fire.
	if _, err := os.Stat(filepath.Join(codexHome, "auth.json")); err != nil {
		return CodexQuotaObservation{}, false, "no Codex credential at " + filepath.Join(codexHome, "auth.json")
	}
	result, err := codexAppServerCall(context.Background(), codexHome, "account/rateLimits/read", "")
	if err != nil {
		return CodexQuotaObservation{}, false, err.Error()
	}
	o := parseCodexAppServerRateLimits(result, now)
	if o.State == "unknown" {
		return CodexQuotaObservation{}, false, o.Reason
	}
	return o, true, ""
}
