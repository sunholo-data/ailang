package quorum

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func fbAbsent(model string) *ReviewerOutcome {
	return &ReviewerOutcome{Model: model, AbsentReason: ReasonBudget}
}

func fbPresent(model string, v Verdict) *ReviewerOutcome {
	return &ReviewerOutcome{Model: model, Present: true, Result: &ReviewResult{Verdict: v, StrongestObjection: "x"}}
}

func TestAnthropicFallback_RunsOnlyWhenEveryExternalSeatIsAbsent(t *testing.T) {
	q := &QuorumResult{Reviewers: []*ReviewerOutcome{fbAbsent("a"), fbPresent("b", VerdictPass), fbAbsent("c")}}
	ran := ApplyAnthropicFallback(q, func() *ReviewerOutcome {
		t.Fatal("fallback ran although an off-Anthropic reviewer was present")
		return nil
	})
	if ran || len(q.Reviewers) != 3 {
		t.Fatalf("ran=%v reviewers=%d, want no fallback", ran, len(q.Reviewers))
	}
}

func TestAnthropicFallback_ControllerVoteDoesNotSuppressIt(t *testing.T) {
	// The controller is Anthropic too: its vote alone is the author's side.
	ctrl := &ControllerReview{Verdict: VerdictPass, Note: "fine"}
	q := &QuorumResult{Reviewers: []*ReviewerOutcome{fbAbsent("a"), fbAbsent("b")}, ControllerInSession: ctrl}
	q.Synthesis = synthesize(q.Reviewers, ctrl)
	ran := ApplyAnthropicFallback(q, func() *ReviewerOutcome { return fbPresent("claude-sonnet-5@claude-p", VerdictReject) })
	if !ran {
		t.Fatal("fallback did not run with every external seat absent")
	}
	last := q.Reviewers[len(q.Reviewers)-1]
	if last.Tier != TierAnthropicFallback {
		t.Errorf("fallback outcome tier = %q, want %q", last.Tier, TierAnthropicFallback)
	}
	if q.Synthesis.Verdict != SynthBlocked {
		t.Errorf("fallback reject must block, synthesis = %s", q.Synthesis.Verdict)
	}
	if !strings.Contains(MarkdownBlock(q), "Anthropic fallback") {
		t.Error("markdown does not label the fallback seat as same-vendor")
	}
}

func TestAnthropicFallback_AbsentFallbackStillBlocksOnZeroSignal(t *testing.T) {
	q := &QuorumResult{Reviewers: []*ReviewerOutcome{fbAbsent("a")}}
	ApplyAnthropicFallback(q, func() *ReviewerOutcome {
		return &ReviewerOutcome{Model: "claude-sonnet-5@claude-p", AbsentReason: ReasonQuota}
	})
	if q.Synthesis.Verdict != SynthBlocked || len(q.Synthesis.AbsentReviewers) != 2 {
		t.Fatalf("synthesis = %+v, want blocked with both seats named absent", q.Synthesis)
	}
}

func stubClaude(t *testing.T, out string, err error) {
	t.Helper()
	orig := claudeReviewCall
	claudeReviewCall = func(_ context.Context, model, sys, user string) ([]byte, error) {
		if model != AnthropicFallbackModel || sys != systemPrompt || !strings.Contains(user, `"verdict"`) {
			t.Errorf("claude call got model=%q, system prompt match=%v, schema in prompt=%v", model, sys == systemPrompt, strings.Contains(user, `"verdict"`))
		}
		return []byte(out), err
	}
	t.Cleanup(func() { claudeReviewCall = orig })
}

func TestRunClaudeSubscriptionReviewer_ParsesVerdictAndTokens(t *testing.T) {
	stubClaude(t, `{"is_error":false,"total_cost_usd":0.02,"result":"{\"verdict\":\"reject\",\"strongest_objection\":\"premise unverified\",\"catch\":\"c\",\"proposed_fix\":\"f\"}","usage":{"input_tokens":900,"cache_read_input_tokens":100,"cache_creation_input_tokens":0,"output_tokens":250}}`, nil)
	o := RunClaudeSubscriptionReviewer(AnthropicFallbackModel, "doc.md", "body")
	if !o.Present || o.Result.Verdict != VerdictReject {
		t.Fatalf("outcome = %+v", o)
	}
	if o.TokensIn != 1000 || o.TokensOut != 250 {
		t.Errorf("tokens = %d/%d, want 1000/250", o.TokensIn, o.TokensOut)
	}
	if o.CostUSD != 0 {
		t.Errorf("subscription call recorded as metered $%f; total_cost_usd is a list-price notional", o.CostUSD)
	}
}

func TestRunClaudeSubscriptionReviewer_FailuresAreNamedAbsences(t *testing.T) {
	cases := []struct {
		name, out string
		err       error
		want      string
	}{
		{"exec error", "", errors.New("exit status 1"), ReasonUnreachable},
		{"cli error", `{"is_error":true,"result":"usage limit reached"}`, nil, ReasonUnreachable},
		{"non-json envelope", `oops`, nil, ReasonInvalid},
		{"prose verdict", `{"is_error":false,"result":"looks good to me"}`, nil, ReasonInvalid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stubClaude(t, c.out, c.err)
			o := RunClaudeSubscriptionReviewer(AnthropicFallbackModel, "doc.md", "body")
			if o.Present || o.AbsentReason != c.want {
				t.Fatalf("present=%v reason=%q, want absent %q", o.Present, o.AbsentReason, c.want)
			}
		})
	}
}

func TestWithoutEnv_StripsTheAPIKeySoBillingStaysOnTheSubscription(t *testing.T) {
	env := withoutEnv([]string{"PATH=/bin", "ANTHROPIC_API_KEY=sk-x", "ANTHROPIC_API_KEY_OTHER=y"}, "ANTHROPIC_API_KEY")
	for _, kv := range env {
		if strings.HasPrefix(kv, "ANTHROPIC_API_KEY=") {
			t.Fatal("ANTHROPIC_API_KEY survived — the fallback could bill the API")
		}
	}
	if len(env) != 2 {
		t.Errorf("env = %v, want PATH and the unrelated var kept", env)
	}
}

func TestClaudeReviewCmd_SubscriptionOnlyAndToolless(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-must-not-reach-claude")
	cmd := claudeReviewCmd(context.Background(), AnthropicFallbackModel, "sys", "user")
	for _, kv := range cmd.Env {
		if strings.HasPrefix(kv, "ANTHROPIC_API_KEY=") {
			t.Fatal("the fallback's claude process inherits ANTHROPIC_API_KEY — it would bill the API, not the subscription")
		}
	}
	args := strings.Join(cmd.Args, "\x00")
	for _, want := range []string{"-p", "--tools\x00\x00", "--setting-sources\x00\x00", "--strict-mcp-config", "--model\x00" + AnthropicFallbackModel} {
		if !strings.Contains(args, want) {
			t.Errorf("claude args %q missing %q", cmd.Args, want)
		}
	}
}
