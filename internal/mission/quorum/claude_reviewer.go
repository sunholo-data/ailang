package quorum

// The Anthropic quorum seat: a claude-sonnet-5 text review over `claude -p` on
// the subscription (keychain OAuth; ANTHROPIC_API_KEY is stripped so it can never
// fall through to metered billing), with every tool disabled and no settings,
// hooks or MCP servers loaded — same prompt and schema as the other seats.
//
// It is an ordinary seat, benched like any other when its vendor wrote the doc
// (seating.go). The mission's designer is Claude on the rotation's opus turn, so
// on that turn this seat sits out; on astra's or deepseek's turn it reviews.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ClaudeReviewerID is the roster id of the Anthropic seat. It is not a
// models.yml row: the suffix routes it to RunClaudeSubscriptionReviewer.
const ClaudeReviewerID = ClaudeReviewerModel + ClaudeReviewerSuffix

// ClaudeReviewerModel is the model the Anthropic seat runs — sonnet, not the
// opus that controls and (on its rotation turn) designs.
const ClaudeReviewerModel = "claude-sonnet-5"

// ClaudeReviewerSuffix marks a roster id that runs through `claude -p`.
const ClaudeReviewerSuffix = "@claude-p"

// ReasonQuota: the seat's provider bucket is over its mission ration.
const ReasonQuota = "quota"

// claudeReviewTimeout bounds one review. A text verdict takes well under a
// minute; a hung CLI must cost minutes, not the iteration.
const claudeReviewTimeout = 5 * time.Minute

// claudeReviewCall runs `claude -p` and returns its --output-format json envelope.
// A variable so tests never start a real claude process.
var claudeReviewCall = callClaudeReview

func callClaudeReview(ctx context.Context, model, sysPrompt, userPrompt string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, claudeReviewTimeout)
	defer cancel()
	out, err := claudeReviewCmd(ctx, model, sysPrompt, userPrompt).Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("claude -p: no reply within %s", claudeReviewTimeout)
		}
		return nil, fmt.Errorf("claude -p: %w", err)
	}
	return out, nil
}

// claudeReviewCmd builds the Anthropic seat's `claude -p` command. Split out so a test
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

// RunClaudeSubscriptionReviewer reviews the doc as the Anthropic seat.
// Like RunReviewer it never returns nil: a failure is a named absence. CostUSD
// stays 0 — the call is subscription-billed, and the CLI's total_cost_usd is a
// list-price notional, not a bill. Tokens are recorded from the CLI's own usage.
func RunClaudeSubscriptionReviewer(model, docPath, docBody string) *ReviewerOutcome {
	out := &ReviewerOutcome{Model: model + ClaudeReviewerSuffix}
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
