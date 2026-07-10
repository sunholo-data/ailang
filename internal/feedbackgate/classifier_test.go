package feedbackgate

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// countingProvider is a fake ai.Provider that returns a canned response and
// counts how many times Generate was called (to assert the agent-* bypass and
// heuristic gating never hit the network).
type countingProvider struct {
	text  string
	err   error
	calls int
}

func (p *countingProvider) Generate(_ context.Context, _ *ai.Request) (*ai.Response, error) {
	p.calls++
	if p.err != nil {
		return nil, p.err
	}
	return &ai.Response{Text: p.text}, nil
}

func (p *countingProvider) Step(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	return p.Generate(ctx, req)
}
func (p *countingProvider) Name() string { return "counting-fake" }

// flaggedInput returns an input that trips shouldClassify (auto: from a
// non-agent sender) and passes the deterministic rules.
func flaggedInput() Input {
	in := baseInput()
	in.From = "mcp-public" // non-agent -> auto: message is flagged
	return in
}

func TestClassifierMatrix(t *testing.T) {
	cases := []struct {
		name       string
		json       string
		wantAction string
		wantReason string
	}{
		{
			name:       "genuine matching category dispatches",
			json:       `{"is_genuine_feedback":true,"is_prompt_injection":false,"best_category":"bug","estimated_dispatch_value":"high","reasoning":"real"}`,
			wantAction: ActionDispatch,
			wantReason: ReasonPassed,
		},
		{
			name:       "prompt injection rejects",
			json:       `{"is_genuine_feedback":false,"is_prompt_injection":true,"best_category":"spam","estimated_dispatch_value":"none","reasoning":"inj"}`,
			wantAction: ActionReject,
			wantReason: ReasonClassifierInjection,
		},
		{
			name:       "no dispatch value files",
			json:       `{"is_genuine_feedback":true,"is_prompt_injection":false,"best_category":"bug","estimated_dispatch_value":"none","reasoning":"junk"}`,
			wantAction: ActionFile,
			wantReason: ReasonClassifierNoValue,
		},
		{
			name:       "not genuine files",
			json:       `{"is_genuine_feedback":false,"is_prompt_injection":false,"best_category":"bug","estimated_dispatch_value":"low","reasoning":"nope"}`,
			wantAction: ActionFile,
			wantReason: ReasonClassifierNotGenuine,
		},
		{
			name:       "category mismatch files",
			json:       `{"is_genuine_feedback":true,"is_prompt_injection":false,"best_category":"feature","estimated_dispatch_value":"high","reasoning":"misrouted"}`,
			wantAction: ActionFile,
			wantReason: ReasonClassifierMismatch,
		},
		{
			name:       "malformed json fails closed to file",
			json:       `{"is_genuine_feedback": tru`,
			wantAction: ActionFile,
			wantReason: ReasonClassifierParseFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prov := &countingProvider{text: tc.json}
			cfg := FeedbackGateConfig{}.normalized()
			cfg.Classifier = NewClassifier(prov, DefaultPrompt(), nil)

			v, err := applyClassifier(context.Background(), flaggedInput(), cfg)
			if err != nil {
				t.Fatalf("applyClassifier error: %v", err)
			}
			if v.Action != tc.wantAction || v.Reason != tc.wantReason {
				t.Fatalf("got %q/%q, want %q/%q", v.Action, v.Reason, tc.wantAction, tc.wantReason)
			}
			if prov.calls != 1 {
				t.Fatalf("provider called %d times, want 1", prov.calls)
			}
		})
	}
}

// TestClassifierProviderErrorFailsClosed: a provider error must file, not
// dispatch, and must NOT surface as a gate-level error (a flaky classifier
// can't open the gate).
func TestClassifierProviderErrorFailsClosed(t *testing.T) {
	prov := &countingProvider{err: errors.New("boom")}
	cfg := FeedbackGateConfig{}.normalized()
	cfg.Classifier = NewClassifier(prov, DefaultPrompt(), nil)

	v, err := applyClassifier(context.Background(), flaggedInput(), cfg)
	if err != nil {
		t.Fatalf("provider error must not propagate as gate error, got %v", err)
	}
	if v.Action != ActionFile || v.Reason != ReasonClassifierError {
		t.Fatalf("got %q/%q, want file/classifier_error", v.Action, v.Reason)
	}
}

// TestClassifierBypassesAgentSenders: agent-* senders never reach the provider.
func TestClassifierBypassesAgentSenders(t *testing.T) {
	prov := &countingProvider{text: `{"is_prompt_injection":true}`} // would reject if called
	cfg := FeedbackGateConfig{}.normalized()
	cfg.Classifier = NewClassifier(prov, DefaultPrompt(), nil)

	in := baseInput()
	in.From = "agent-eval-suite"
	v, err := applyClassifier(context.Background(), in, cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if v.Action != ActionDispatch {
		t.Fatalf("agent sender: Action = %q, want dispatch", v.Action)
	}
	if prov.calls != 0 {
		t.Fatalf("provider called %d times for agent sender, want 0", prov.calls)
	}
}

// TestShouldClassifyHeuristics locks in the last-resort heuristic: only long
// code blocks, big snippet-less bugs, or auto:+non-agent messages are flagged.
// A small message from an agent sender is NOT flagged.
func TestShouldClassifyHeuristics(t *testing.T) {
	// auto: + non-agent -> flagged (the risky case the gate exists for).
	if !shouldClassify(flaggedInput()) {
		t.Error("auto: from non-agent should be flagged")
	}
	// Short body from an agent sender -> not flagged by any heuristic.
	agentShort := baseInput()
	agentShort.From = "agent-x"
	if shouldClassify(agentShort) {
		t.Error("short agent-sent message should not be flagged")
	}
	// Very long body (>200 newlines) -> flagged.
	longLines := baseInput()
	longLines.From = "agent-x"
	longLines.Body = strings.Repeat("line\n", 250)
	if !shouldClassify(longLines) {
		t.Error("long code block should be flagged")
	}
	// Big snippet-less bug -> flagged.
	bigBug := baseInput()
	bigBug.From = "agent-x"
	bigBug.Body = strings.Repeat("x", 5000)
	if !shouldClassify(bigBug) {
		t.Error("big snippet-less bug should be flagged")
	}
}

// TestClassifierFileOnlyModeDisablesLLM: Mode=file-only skips the provider.
func TestClassifierFileOnlyModeDisablesLLM(t *testing.T) {
	prov := &countingProvider{text: `{"is_prompt_injection":true}`}
	cfg := FeedbackGateConfig{Mode: ModeFileOnly}.normalized()
	cfg.Classifier = NewClassifier(prov, DefaultPrompt(), nil)

	v, err := applyClassifier(context.Background(), flaggedInput(), cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if v.Action != ActionDispatch {
		t.Fatalf("file-only: Action = %q, want dispatch (classifier disabled)", v.Action)
	}
	if prov.calls != 0 {
		t.Fatalf("provider called %d times in file-only mode, want 0", prov.calls)
	}
}

// TestPromptHasVersion: the embedded prompt carries a version marker for replay.
func TestPromptHasVersion(t *testing.T) {
	p := DefaultPrompt()
	if !strings.Contains(p, "version:") {
		t.Fatal("classifier prompt missing a version: marker")
	}
	// PromptHash is stable and non-empty.
	c := NewClassifier(nil, p, nil)
	if h := c.PromptHash(); len(h) == 0 {
		t.Fatal("PromptHash returned empty")
	}
}

// TestNilProviderFailsClosed: a classifier with a nil provider files a flagged
// message (fail closed), never dispatch.
func TestNilProviderFailsClosed(t *testing.T) {
	cfg := FeedbackGateConfig{}.normalized()
	cfg.Classifier = NewClassifier(nil, DefaultPrompt(), nil)
	v, err := applyClassifier(context.Background(), flaggedInput(), cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if v.Action != ActionFile || v.Reason != ReasonClassifierError {
		t.Fatalf("nil provider: got %q/%q, want file/classifier_error", v.Action, v.Reason)
	}
}
