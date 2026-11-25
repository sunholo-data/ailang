package quorum

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

func stubCodex(t *testing.T, out string, err error) {
	t.Helper()
	orig := codexReviewCall
	codexReviewCall = func(_ context.Context, model, prompt string) ([]byte, error) {
		if model != CodexReviewerModel || !strings.HasPrefix(prompt, systemPrompt) || !strings.Contains(prompt, `"verdict"`) {
			t.Errorf("codex call got model=%q, system prompt first=%v, schema in prompt=%v", model, strings.HasPrefix(prompt, systemPrompt), strings.Contains(prompt, `"verdict"`))
		}
		return []byte(out), err
	}
	t.Cleanup(func() { codexReviewCall = orig })
}

const codexVerdictStream = `{"type":"thread.started","thread_id":"t"}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"i0","type":"reasoning","text":"thinking"}}
{"type":"item.completed","item":{"id":"i1","type":"agent_message","text":"{\"verdict\":\"reject\",\"strongest_objection\":\"premise unverified\",\"catch\":\"c\",\"proposed_fix\":\"f\"}"}}
{"type":"turn.completed","usage":{"input_tokens":1200,"cached_input_tokens":200,"output_tokens":300}}
`

func TestRunCodexSubscriptionReviewer_ParsesVerdictAndTokens(t *testing.T) {
	stubCodex(t, codexVerdictStream, nil)
	o := RunCodexSubscriptionReviewer(CodexReviewerModel, "doc.md", "body")
	if !o.Present || o.Result.Verdict != VerdictReject || o.Model != CodexReviewerID {
		t.Fatalf("outcome = %+v", o)
	}
	// cached_input_tokens is a subset of input_tokens: not added again.
	if o.TokensIn != 1200 || o.TokensOut != 300 {
		t.Errorf("tokens = %d/%d, want 1200/300", o.TokensIn, o.TokensOut)
	}
	if o.CostUSD != 0 {
		t.Errorf("subscription call recorded as metered $%f", o.CostUSD)
	}
}

func TestRunCodexSubscriptionReviewer_FailuresAreNamedAbsences(t *testing.T) {
	cases := []struct {
		name, out string
		err       error
		want      string
	}{
		{"process error", "", errors.New("codex exec: exit status 1"), ReasonUnreachable},
		{"turn failed", `{"type":"turn.failed","error":{"message":"You've hit your usage limit"}}`, nil, ReasonUnreachable},
		{"stream error", `{"type":"error","message":"stream disconnected"}`, nil, ReasonUnreachable},
		{"no agent message", `{"type":"turn.completed","usage":{"input_tokens":5,"output_tokens":0}}`, nil, ReasonInvalid},
		{"prose verdict", `{"type":"item.completed","item":{"type":"agent_message","text":"looks fine to me"}}`, nil, ReasonInvalid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stubCodex(t, c.out, c.err)
			o := RunCodexSubscriptionReviewer(CodexReviewerModel, "doc.md", "body")
			if o == nil || o.Present || o.AbsentReason != c.want || o.Err == "" {
				t.Fatalf("outcome = %+v, want absent %q with an error", o, c.want)
			}
		})
	}
}

// The call site itself: subscription only, read-only, nothing persisted, prompt on
// stdin, outside the shared tree.
func TestCodexReviewCmd_SubscriptionReadOnlyStdin(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-must-not-reach-codex")
	cmd := codexReviewCmd(context.Background(), CodexReviewerModel, "/tmp/review-dir", "THE PROMPT")
	args := cmd.Args[1:]
	for _, want := range [][]string{
		{"exec", "--json"}, {"--model", CodexReviewerModel}, {"--sandbox", "read-only"},
		{"--ephemeral"}, {"-c", SpawnOverrideKey + "=false"},
	} {
		if !containsRun(args, want) {
			t.Errorf("argv %q missing %q", args, want)
		}
	}
	if args[len(args)-1] != "-" || slices.Contains(args, "THE PROMPT") {
		t.Errorf("prompt must come from stdin, argv = %q", args)
	}
	if slices.Contains(args, "--dangerously-bypass-approvals-and-sandbox") {
		t.Error("a reviewer must not bypass the sandbox")
	}
	if cmd.Dir != "/tmp/review-dir" {
		t.Errorf("cmd.Dir = %q, want the private review dir", cmd.Dir)
	}
	for _, kv := range cmd.Env {
		if strings.HasPrefix(kv, "OPENAI_API_KEY=") {
			t.Fatal("OPENAI_API_KEY reached codex: the seat must bill the subscription, never the API")
		}
	}
	opts := codexDaemonReviewOptions(CodexReviewerModel, "/tmp/review-dir", "THE PROMPT", []string{"OPENAI_API_KEY=x", "CODEX_HOME=/h"})
	if opts.Sandbox != "read-only" || !opts.Ephemeral || opts.Config[SpawnOverrideKey] != false || slices.Contains(opts.Env, "OPENAI_API_KEY=x") {
		t.Errorf("daemon options = %+v", opts)
	}
}

func containsRun(args, run []string) bool {
	for i := 0; i+len(run) <= len(args); i++ {
		if slices.Equal(args[i:i+len(run)], run) {
			return true
		}
	}
	return false
}

// The seat id is the mission's own lane spelling, so the allowlist, the vendor map
// and author benching already understand it.
func TestCodexReviewerID_IsAnAllowlistedOpenAISeat(t *testing.T) {
	if !QuorumModelAllowed(CodexReviewerID) {
		t.Fatalf("%s refused by the quorum allowlist", CodexReviewerID)
	}
	if v := VendorOf(CodexReviewerID); v != "openai" {
		t.Fatalf("VendorOf(%s) = %q, want openai", CodexReviewerID, v)
	}
	sel := SelectReviewers([]string{CodexReviewerID, "gemini-4-argon", "oc-glm-5-3", "oc-kimi-k3"}, "codex:gpt-6.1-sol", "d.md", 3)
	if !slices.Equal(sel.Benched, []string{CodexReviewerID}) {
		t.Fatalf("an OpenAI-authored doc must bench the OpenAI seat, got %+v", sel)
	}
}
