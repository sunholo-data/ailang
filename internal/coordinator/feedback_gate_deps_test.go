package coordinator

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/feedbackgate"
)

// These tests cover the M-FEEDBACK-GATE-CLOUD-ADAPTER wiring: the deps-
// attachment CALL-SITE (enableFeedbackGate, which initTaskProcessing calls),
// the nil-deps rules-only path, and the no-key fail-closed classifier. They use
// fakes only — no Firestore, no network (the sprint's no-credentials gate).

// fakeCooldown is an in-memory feedbackgate.CooldownStore. It always reports one
// attempt in-window so the gate would dispatch (we only assert it is CONSULTED,
// i.e. the dep reached Decide).
type fakeCooldown struct {
	calls int
	key   string
}

func (f *fakeCooldown) Increment(_ context.Context, key string, _ time.Time) (int, int, error) {
	f.calls++
	f.key = key
	return 1, 1, nil // under any limit -> dispatch continues
}

// fakeProvider is a minimal ai.Provider whose Generate returns a canned
// dispatch-worthy JSON. Used to prove a live-provider classifier reaches the
// network path (calls > 0) once attached.
type fakeProvider struct {
	calls int
	text  string
}

func (p *fakeProvider) Generate(_ context.Context, _ *ai.Request) (*ai.Response, error) {
	p.calls++
	return &ai.Response{Text: p.text}, nil
}
func (p *fakeProvider) Step(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	return p.Generate(ctx, req)
}
func (p *fakeProvider) Name() string { return "fake" }

func newDepsTestDaemon() *Daemon {
	return &Daemon{
		logger: log.New(&nopWriter{}, "", 0),
		ctx:    context.Background(),
	}
}

// flaggedGateInput passes the deterministic rules AND trips shouldClassify
// (auto: category from a non-agent sender), so it reaches the cooldown and
// classifier stages when those deps are attached.
func flaggedGateInput() feedbackgate.Input {
	return feedbackgate.Input{
		ID:       "m1",
		Category: "auto:bug",
		Body:     "a genuine bug report body",
		From:     "mcp-public",
		Inbox:    "pkg:a/b",
		Source:   "public",
	}
}

// TestEnableFeedbackGateAttachesDeps guards the CALL-SITE (M-ENV-FORWARD
// lesson): deps set on the daemon via SetFeedbackGateDeps must land on the
// config the decider reads, and a subsequent Decide must actually consult them.
func TestEnableFeedbackGateAttachesDeps(t *testing.T) {
	d := newDepsTestDaemon()

	cooldown := &fakeCooldown{}
	provider := &fakeProvider{text: `{"is_genuine_feedback":true,"is_prompt_injection":false,"best_category":"bug","estimated_dispatch_value":"high"}`}
	classifier := feedbackgate.NewClassifier(provider, feedbackgate.DefaultPrompt(), nil)

	// Simulate the CLI construction handing deps to the daemon.
	d.SetFeedbackGateDeps(cooldown, classifier)

	// Simulate the enabled-block call-site in initTaskProcessing.
	cfg := &FeedbackGateConfig{Enabled: true}
	d.enableFeedbackGate(cfg)

	if d.feedbackGateCfg.Cooldown == nil {
		t.Fatal("cooldown dep did not reach the gate config (call-site not wired)")
	}
	if d.feedbackGateCfg.Classifier == nil {
		t.Fatal("classifier dep did not reach the gate config (call-site not wired)")
	}

	// Prove the decider actually consults them: run a flagged input through the
	// real Decide with the daemon's config; both fakes must be exercised.
	v, err := feedbackgate.Decide(d.ctx, flaggedGateInput(), *d.feedbackGateCfg)
	if err != nil {
		t.Fatalf("Decide returned error: %v", err)
	}
	if cooldown.calls != 1 {
		t.Errorf("cooldown not consulted by Decide: calls=%d, want 1", cooldown.calls)
	}
	if provider.calls != 1 {
		t.Errorf("classifier provider not consulted by Decide: calls=%d, want 1", provider.calls)
	}
	if v.Action != feedbackgate.ActionDispatch {
		t.Errorf("verdict = %s, want dispatch (all fakes say pass)", v.Action)
	}
}

// TestEnableFeedbackGateNilDepsUnchanged: gate enabled but NO deps set => the
// config's Cooldown/Classifier stay nil, so the gate runs rules-only exactly as
// before this adapter (zero behavior change).
func TestEnableFeedbackGateNilDepsUnchanged(t *testing.T) {
	d := newDepsTestDaemon()
	// No SetFeedbackGateDeps call.

	cfg := &FeedbackGateConfig{Enabled: true}
	d.enableFeedbackGate(cfg)

	if d.feedbackGateCfg.Cooldown != nil {
		t.Error("nil-deps: Cooldown must remain nil (rules-only unchanged)")
	}
	if d.feedbackGateCfg.Classifier != nil {
		t.Error("nil-deps: Classifier must remain nil (rules-only unchanged)")
	}
	if d.feedbackGate == nil {
		t.Error("gate decider must still be installed when enabled")
	}
}

// TestNoKeyClassifierFailsClosed: the construction path with an empty
// ANTHROPIC_API_KEY yields a classifier whose nil-provider branch files a
// heuristic-flagged input. Assert on the BUILT classifier via Decide — no
// network. This mirrors what coordinator_lifecycle.go builds when the key is
// absent.
func TestNoKeyClassifierFailsClosed(t *testing.T) {
	// nil provider == no ANTHROPIC_API_KEY in the CLI construction block.
	var provider ai.Provider // nil
	classifier := feedbackgate.NewClassifier(provider, feedbackgate.DefaultPrompt(), nil)

	if classifier.HasProvider() {
		t.Fatal("classifier with nil provider must report HasProvider()==false")
	}

	cfg := feedbackgate.FeedbackGateConfig{
		Enabled:    true,
		Classifier: classifier,
	}
	v, err := feedbackgate.Decide(context.Background(), flaggedGateInput(), cfg)
	if err != nil {
		t.Fatalf("Decide returned error: %v", err)
	}
	if v.Action != feedbackgate.ActionFile {
		t.Errorf("no-key classifier must FILE a heuristic-flagged input (fail closed), got %s (reason=%s)", v.Action, v.Reason)
	}
}

// TestStageNamingHelpers verifies the startup-log stage names for the three
// configurations the runbook documents.
func TestStageNamingHelpers(t *testing.T) {
	// Fully wired: cooldown + live-provider classifier.
	full := &feedbackgate.FeedbackGateConfig{
		Cooldown:   &fakeCooldown{},
		Classifier: feedbackgate.NewClassifier(&fakeProvider{}, feedbackgate.DefaultPrompt(), nil),
	}
	if got := feedbackGateCooldownStage(full); got != "firestore" {
		t.Errorf("cooldown stage = %q, want firestore", got)
	}
	if got := feedbackGateClassifierStage(full); got != "anthropic" {
		t.Errorf("classifier stage = %q, want anthropic", got)
	}
	if got := feedbackGateBudgetStage(full); got != "firestore" {
		t.Errorf("budget stage = %q, want firestore", got)
	}

	// Key missing: cooldown attached, classifier attached but no provider.
	failClosed := &feedbackgate.FeedbackGateConfig{
		Cooldown:   &fakeCooldown{},
		Classifier: feedbackgate.NewClassifier(nil, feedbackgate.DefaultPrompt(), nil),
	}
	if got := feedbackGateClassifierStage(failClosed); got != "fail-closed" {
		t.Errorf("classifier stage (no key) = %q, want fail-closed", got)
	}

	// Rules-only: no deps at all.
	none := &feedbackgate.FeedbackGateConfig{}
	if got := feedbackGateCooldownStage(none); got != "none" {
		t.Errorf("cooldown stage (rules-only) = %q, want none", got)
	}
	if got := feedbackGateClassifierStage(none); got != "none" {
		t.Errorf("classifier stage (rules-only) = %q, want none", got)
	}
	if got := feedbackGateBudgetStage(none); got != "none" {
		t.Errorf("budget stage (rules-only) = %q, want none", got)
	}
}
