package quorum

// The Anthropic reviewer of last resort.
//
// The quorum's external seats are deliberately off-Anthropic: the mission's author
// and controller are Claude, and "ideally no model provider marks its own work"
// (Mark). But a quorum where EVERY external seat is absent — over budget, over a
// provider ration, out of session quota — has only the controller's in-session
// vote, which is the author's side. Mark, attended 2026-09-25: Anthropic not
// reviewing Anthropic work is the rule, "but if all other reviewers are blocked
// it's ok to relax that a bit."
//
// So this seat runs ONLY when no off-Anthropic reviewer produced a verdict. It
// goes through `claude -p` on the subscription (keychain OAuth; ANTHROPIC_API_KEY
// is stripped so it can never fall through to metered billing), with every tool
// disabled and no settings, hooks or MCP servers loaded — a text review, same
// prompt and schema as the other seats. It is labelled in the artifact
// (tier "anthropic-fallback") so nobody reads it as an independent vendor.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// AnthropicFallbackModel is the Claude model the fallback seat runs. Sonnet, not
// the opus that designs and controls: same vendor, but at least not the author.
const AnthropicFallbackModel = "claude-sonnet-5"

// TierAnthropicFallback labels the fallback outcome in the artifact.
const TierAnthropicFallback = "anthropic-fallback"

// ReasonQuota: the seat's provider bucket is over its mission ration.
const ReasonQuota = "quota"

// anthropicFallbackTimeout bounds one review. A text verdict takes well under a
// minute; a hung CLI must cost minutes, not the iteration.
const anthropicFallbackTimeout = 5 * time.Minute

// NeedsAnthropicFallback is true iff no off-Anthropic reviewer produced a verdict.
// The controller's in-session vote does not count: it is Anthropic too.
func NeedsAnthropicFallback(q *QuorumResult) bool {
	if q == nil {
		return false
	}
	for _, o := range q.Reviewers {
		if o != nil && o.Present {
			return false
		}
	}
	return true
}

// ApplyAnthropicFallback runs the fallback seat when every external reviewer is
// absent, appends its outcome and re-synthesizes. It reports whether it ran.
// run is RunClaudeSubscriptionReviewer in production, a stub in tests.
func ApplyAnthropicFallback(q *QuorumResult, run func() *ReviewerOutcome) bool {
	if !NeedsAnthropicFallback(q) {
		return false
	}
	o := run()
	o.Tier = TierAnthropicFallback
	q.Reviewers = append(q.Reviewers, o)
	q.Synthesis = synthesize(q.Reviewers, q.ControllerInSession)
	return true
}

// claudeReviewCall runs `claude -p` and returns its --output-format json envelope.
// A variable so tests never start a real claude process.
var claudeReviewCall = callClaudeReview

func callClaudeReview(ctx context.Context, model, sysPrompt, userPrompt string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, anthropicFallbackTimeout)
	defer cancel()
	out, err := claudeReviewCmd(ctx, model, sysPrompt, userPrompt).Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("claude -p: no reply within %s", anthropicFallbackTimeout)
		}
		return nil, fmt.Errorf("claude -p: %w", err)
	}
	return out, nil
}

// claudeReviewCmd builds the fallback's `claude -p` command. Split out so a test
// can pin the call site itself: no API key (subscription only), no tools, no
// settings, hooks or MCP servers.
func claudeReviewCmd(ctx context.Context, model, sysPrompt, userPrompt string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "claude", "-p",
		"--model", model,
		"--output-format", "json",
		"--system-prompt", sysPrompt,
		"--tools", "",
		"--no-session-persistence",
		"--setting-sources", "",
		"--strict-mcp-config")
	cmd.Env = withoutEnv(os.Environ(), "ANTHROPIC_API_KEY")
	cmd.Stdin = strings.NewReader(userPrompt)
	cmd.WaitDelay = 5 * time.Second
	return cmd
}

func withoutEnv(env []string, name string) []string {
	out := env[:0:0]
	for _, kv := range env {
		if !strings.HasPrefix(kv, name+"=") {
			out = append(out, kv)
		}
	}
	return out
}

// RunClaudeSubscriptionReviewer reviews the doc as the Anthropic fallback seat.
// Like RunReviewer it never returns nil: a failure is a named absence. CostUSD
// stays 0 — the call is subscription-billed, and the CLI's total_cost_usd is a
// list-price notional, not a bill. Tokens are recorded from the CLI's own usage.
func RunClaudeSubscriptionReviewer(model, docPath, docBody string) *ReviewerOutcome {
	out := &ReviewerOutcome{Model: model + "@claude-p"}
	user := BuildPrompt(docPath, docBody) +
		"\n\nRespond with ONLY a JSON object matching this schema, no prose, no code fence:\n" + reviewSchema
	raw, err := claudeReviewCall(context.Background(), model, systemPrompt, user)
	if err != nil {
		out.AbsentReason = ReasonUnreachable
		out.Err = err.Error()
		return out
	}
	var env struct {
		Result  string `json:"result"`
		IsError bool   `json:"is_error"`
		Usage   struct {
			Input       int `json:"input_tokens"`
			CacheRead   int `json:"cache_read_input_tokens"`
			CacheCreate int `json:"cache_creation_input_tokens"`
			Output      int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		out.AbsentReason = ReasonInvalid
		out.Err = fmt.Sprintf("claude -p returned a non-JSON envelope: %v (raw: %.200q)", err, raw)
		return out
	}
	out.TokensIn = env.Usage.Input + env.Usage.CacheRead + env.Usage.CacheCreate
	out.TokensOut = env.Usage.Output
	if env.IsError {
		out.AbsentReason = ReasonUnreachable
		out.Err = fmt.Sprintf("claude -p reported an error: %.300s", env.Result)
		return out
	}
	result, perr := ParseReviewResult(env.Result)
	if perr != nil {
		out.AbsentReason = ReasonInvalid
		out.Err = perr.Error()
		return out
	}
	out.Present = true
	out.Result = result
	return out
}
