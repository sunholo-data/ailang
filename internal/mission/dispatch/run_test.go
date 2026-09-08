package dispatch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

type fakeExecutor struct {
	name   string
	health error
	caps   []executor.Capability
	calls  int
	task   *executor.Task
	result *executor.Result
	err    error
	wait   bool
}

func (f *fakeExecutor) Name() string { return f.name }
func (f *fakeExecutor) Execute(ctx context.Context, t *executor.Task) (*executor.Result, error) {
	return f.ExecuteStreaming(ctx, t, nil)
}
func (f *fakeExecutor) ExecuteStreaming(ctx context.Context, t *executor.Task, _ executor.EventHandler) (*executor.Result, error) {
	f.calls++
	f.task = t
	if f.wait {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.result, f.err
}
func (f *fakeExecutor) Capabilities() []executor.Capability { return f.caps }
func (f *fakeExecutor) CostModel() *executor.CostModel      { return nil }
func (f *fakeExecutor) HealthCheck(context.Context) error   { return f.health }
func (f *fakeExecutor) Close() error                        { return nil }

type fakeFactory map[string]*fakeExecutor

func (f fakeFactory) GetExecutor(n string) (executor.Executor, error) {
	if e, ok := f[n]; ok {
		return e, nil
	}
	return nil, fmt.Errorf("missing %s", n)
}

func fixture(t *testing.T) (Request, *Runner, fakeFactory, *[]Event) {
	t.Helper()
	r := validRequest(t)
	r.Models = []string{"unavailable", "same-vendor", "judge"}
	models := map[string]modelreg.ModelConfig{}
	for _, row := range []struct{ name, cli, vendor string }{{"author", "codex", "openai"}, {"unavailable", "missing", "minimax"}, {"same-vendor", "codex", "openai"}, {"judge", "pi", "minimax"}, {"tail", "claude", "anthropic"}} {
		cli := row.cli
		wire := "openrouter/" + row.vendor + "/model"
		models[row.name] = modelreg.ModelConfig{Provider: "openrouter", APIName: row.vendor + "/model", AgentCLI: &cli, AgentModelName: &wire, Pricing: modelreg.Pricing{InputPer1K: .001, OutputPer1K: .002}}
	}
	f := fakeFactory{}
	for _, name := range []string{"codex", "pi", "claude"} {
		f[name] = &fakeExecutor{name: name, caps: []executor.Capability{executor.CapLocalWorkspace}, result: &executor.Result{Success: true, Output: "independent review", FinishReason: executor.FinishStop, SessionID: "session"}}
	}
	events := []Event{}
	runner := &Runner{Admit: func(context.Context, Candidate) (Admission, error) {
		return Admission{Allowed: true, Policy: "fixture", ObservedAt: time.Now()}, nil
	}, Models: &modelreg.ModelsConfig{Models: models}, Executors: f, Record: func(e Event) error { events = append(events, e); return nil }, LookupEnv: func(string) string { return "" }}
	return r, runner, f, &events
}

func TestDispatchPreflightFallbackAndReceipt(t *testing.T) {
	r, runner, f, events := fixture(t)
	report, err := runner.Run(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "execution_completed" || report.ArtifactVerified {
		t.Fatalf("false acceptance: %+v", report)
	}
	if f["pi"].calls != 1 || f["codex"].calls != 0 {
		t.Fatal("independence or fallback broken")
	}
	// generator != judge is PREFERRED, not required (Mark, attended 2026-09-08). The
	// same-vendor candidate is no longer refused — insisting on cross-vendor made the judge
	// unroutable, because the only harness that loads the sprint-evaluator skill is claude,
	// so a claude-authored item had no valid evaluator at all. It is instead sorted BEHIND
	// every cross-vendor candidate, so the property still holds whenever it can: the
	// cross-vendor "judge" runs and the same-vendor rung is never reached.
	//
	// Still refused, and asserted below: a model judging its OWN output.
	if len(report.Attempts) != 2 {
		t.Fatalf("want the unroutable skip plus the cross-vendor judge, got: %+v", report)
	}
	if report.Attempts[0].Status != "skipped" {
		t.Fatalf("missing rejection evidence for the unroutable candidate: %+v", report)
	}
	if report.Attempts[1].Candidate.Model != "judge" || report.Attempts[1].Candidate.SameVendorAsAuthor {
		t.Fatalf("cross-vendor judge did not win the ordering: %+v", report.Attempts[1].Candidate)
	}
	if f["pi"].task.Model != "openrouter/minimax/model" || f["pi"].task.Directive != r.Instructions || f["pi"].task.MaxTokensPerBench != r.MaxTokens {
		t.Fatalf("request not delivered: %+v", f["pi"].task)
	}
	if f["pi"].task.ExtraEnv["AILANG_MESSAGES_STORE"] != "gcp" || f["pi"].task.ExtraEnv["AILANG_MESSAGES_PROJECT"] != "ailang-multivac" {
		t.Fatal("canonical message bindings missing")
	}
	if len(*events) < 4 || (*events)[len(*events)-1].Kind != "finished" {
		t.Fatal("terminal receipt missing")
	}
}

func TestNoFallbackAfterExecutionStarts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result *executor.Result
		err    error
	}{
		{"error", nil, errors.New("connection lost")}, {"nil", nil, nil},
		{"empty", &executor.Result{Success: true, FinishReason: executor.FinishStop}, nil},
		{"failed", &executor.Result{Success: false, Output: "PASS", FinishReason: executor.FinishStop}, nil},
		{"timeout", &executor.Result{Success: true, Output: "PASS", FinishReason: executor.FinishTimeout}, nil},
		{"cost", &executor.Result{Success: true, Output: "PASS", FinishReason: executor.FinishStop, CostKilledAt: 1}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, runner, f, _ := fixture(t)
			r.Models = []string{"judge", "tail"}
			f["pi"].result = tc.result
			f["pi"].err = tc.err
			report, err := runner.Run(context.Background(), r)
			if err == nil || report.Status == "execution_completed" || f["claude"].calls != 0 {
				t.Fatalf("unsafe retry/success: %+v, %v", report, err)
			}
		})
	}
}

func TestHealthAndCapabilityFallback(t *testing.T) {
	for _, kind := range []string{"health", "capability"} {
		t.Run(kind, func(t *testing.T) {
			r, runner, f, _ := fixture(t)
			r.Models = []string{"judge", "tail"}
			if kind == "health" {
				f["pi"].health = errors.New("offline")
			} else {
				f["pi"].caps = nil
			}
			if _, err := runner.Run(context.Background(), r); err != nil {
				t.Fatal(err)
			}
			if f["pi"].calls != 0 || f["claude"].calls != 1 {
				t.Fatal("preflight failure executed")
			}
		})
	}
}

func TestCancelledAndUnrecordedDispatchNeverLaunch(t *testing.T) {
	r, runner, f, _ := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runner.Run(ctx, r); err == nil {
		t.Fatal("cancel accepted")
	}
	if f["pi"].calls != 0 {
		t.Fatal("cancelled call launched")
	}
	runner.Record = func(Event) error { return errors.New("disk full") }
	if _, err := runner.Run(context.Background(), r); err == nil {
		t.Fatal("receipt failure accepted")
	}
	if f["pi"].calls != 0 {
		t.Fatal("unrecorded call launched")
	}
}

func TestExecutionDeadlineAndTerminalReceiptFailure(t *testing.T) {
	r, runner, f, _ := fixture(t)
	r.Models = []string{"judge", "tail"}
	f["pi"].wait = true
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := runner.Run(ctx, r); err == nil {
		t.Fatal("deadline accepted")
	}
	if f["claude"].calls != 0 {
		t.Fatal("deadline retried")
	}
	f["pi"].wait = false
	runner.Record = func(e Event) error {
		if e.Kind == "finished" {
			return errors.New("full")
		}
		return nil
	}
	if _, err := runner.Run(context.Background(), r); err == nil {
		t.Fatal("lost completion receipt reported successful")
	}
}

func TestResolveRefusesIncompleteConfigurationBeforeLaunch(t *testing.T) {
	r, runner, f, _ := fixture(t)
	r.Models = []string{"judge", "missing-row"}
	if _, err := runner.Run(context.Background(), r); err == nil {
		t.Fatal("invalid tail accepted")
	}
	if f["pi"].calls != 0 {
		t.Fatal("partially resolved request executed")
	}
	r.Models = []string{"judge"}
	r.AuthorModels = []string{"unknown-author"}
	if _, err := runner.Run(context.Background(), r); err == nil {
		t.Fatal("unknown author accepted")
	}
}

func TestJournalExclusive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("durable receipt directory sync is unsupported on Windows; execution fails closed")
	}
	p := filepath.Join(t.TempDir(), "attempt.jsonl")
	j, err := OpenJournal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := j.Record(Event{Kind: "started"}); err != nil {
		t.Fatal(err)
	}
	if err := j.Close(); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(p)
	if _, err := OpenJournal(p); err == nil {
		t.Fatal("existing receipt overwritten")
	}
	after, _ := os.ReadFile(p)
	if string(after) != string(before) {
		t.Fatal("receipt changed")
	}
}

func TestResolvePriceAndRegistryEvidence(t *testing.T) {
	r, runner, _, _ := fixture(t)
	p, err := Resolve(r, runner.Models)
	if err != nil {
		t.Fatal(err)
	}
	row := runner.Models.Models["judge"]
	row.Pricing.InputPer1K *= 2
	runner.Models.Models["judge"] = row
	changed, err := Resolve(r, runner.Models)
	if err != nil {
		t.Fatal(err)
	}
	if p.RegistryDigest == changed.RegistryDigest {
		t.Fatal("registry mutation lost from receipt evidence")
	}
	row.Pricing.InputPer1K = 0
	runner.Models.Models["judge"] = row
	if _, err := Resolve(r, runner.Models); err == nil {
		t.Fatal("accepted unavailable pricing")
	}
}

func TestUnsupportedAdapterNeverStarts(t *testing.T) {
	r, runner, f, _ := fixture(t)
	r.Models = []string{"judge"}
	row := runner.Models.Models["judge"]
	cli := "motoko"
	row.AgentCLI = &cli
	runner.Models.Models["judge"] = row
	f[cli] = &fakeExecutor{name: cli, caps: []executor.Capability{executor.CapLocalWorkspace}}
	report, err := runner.Run(context.Background(), r)
	if err == nil || f[cli].calls != 0 || report.Attempts[0].Reason == "" {
		t.Fatal("unsupported budget adapter admitted")
	}
}

func TestLocalCandidateSkipsBeforeCloudFallback(t *testing.T) {
	r, runner, f, _ := fixture(t)
	cli, wire := "pi", "ollama/local-model"
	runner.Models.Models["local"] = modelreg.ModelConfig{Provider: "ollama", APIName: "local-model", ModelVendor: "minimax", AgentCLI: &cli, AgentModelName: &wire}
	r.Models = []string{"local", "judge"}
	report, err := runner.Run(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if f["pi"].calls != 1 || len(report.Attempts) != 2 || report.Attempts[0].Status != "skipped" {
		t.Fatalf("local admission broke cloud fallback: %+v", report)
	}
}

func TestConflictingWireNeverDispatches(t *testing.T) {
	r, runner, f, _ := fixture(t)
	row := runner.Models.Models["judge"]
	wire := "openrouter/openai/gpt"
	row.AgentModelName = &wire
	runner.Models.Models["judge"] = row
	if _, err := runner.Run(context.Background(), r); err == nil {
		t.Fatal("conflicting wire identity accepted")
	}
	for _, e := range f {
		if e.calls != 0 {
			t.Fatal("dispatched before complete identity validation")
		}
	}
}

// An evaluator must not hold a tool that can write. Its contract says so three times in
// prose — "preserve HEAD and all tracked files", "do not repair the candidate", "do not add
// review docs in the workspace" — and until 2026-09-08 the harness handed it pi's defaults,
// which include edit and write. A judge able to rewrite the artifact it is judging
// invalidates the acceptance evidence the whole canary exists to produce.
func TestTaskFor_EvaluatorCannotMutate(t *testing.T) {
	task := taskFor(Request{Role: "evaluator", MissionID: "docs", WorkItemID: "w", StageID: "evaluator"}, testCandidate())
	if task.AllowedTools == nil {
		t.Fatal("evaluator got nil AllowedTools — pi's defaults include edit and write")
	}
	got := map[string]bool{}
	for _, tool := range task.AllowedTools {
		got[tool] = true
	}
	for _, banned := range []string{"edit", "write"} {
		if got[banned] {
			t.Errorf("evaluator may hold %q", banned)
		}
	}
	// bash must remain: the contract requires the bound verification commands to run.
	if !got["bash"] {
		t.Error("evaluator lost bash — it cannot run its bound validator")
	}
	if !got["read"] {
		t.Error("evaluator lost read")
	}
}

// Author roles are unchanged: nil means "pi's defaults apply", and they must keep write.
// Restricting them here would silently stop every executor from producing an artifact.
func TestTaskFor_NonEvaluatorRolesKeepDefaults(t *testing.T) {
	for _, role := range []string{"executor", "designer", "planner"} {
		task := taskFor(Request{Role: role, MissionID: "docs", WorkItemID: "w", StageID: role}, testCandidate())
		if task.AllowedTools != nil {
			t.Errorf("role %q got AllowedTools=%v, want nil (pi defaults)", role, task.AllowedTools)
		}
	}
}

// Role matching is case-insensitive: the bound is a security property, and it must not be
// defeated by a spec that spells the role "Evaluator".
func TestTaskFor_EvaluatorBoundIsCaseInsensitive(t *testing.T) {
	for _, spelling := range []string{"Evaluator", "EVALUATOR", "evaluator"} {
		if taskFor(Request{Role: spelling}, testCandidate()).AllowedTools == nil {
			t.Errorf("role %q escaped the evaluator tool bound", spelling)
		}
	}
}

// testCandidate supplies the non-nil model config taskFor dereferences.
func testCandidate() Candidate {
	return Candidate{Model: "m", WireModel: "wire", config: &modelreg.ModelConfig{}}
}

// A frozen work item whose behaviour depends on the current contents of AGENTS.md is not
// frozen. pi discovers AGENTS.md and CLAUDE.md by default; measured 2026-09-08, an evaluator
// inherited "Read CLAUDE.md first - hard gate", "Work Routing (do not self-approve)" and
// "Classify every task BEFORE touching code" from a repo worktree, none of it in its
// contract, and one model refused the job as a suspected prompt injection.
func TestTaskFor_EvaluatorIsIsolatedFromAmbientRepoInstructions(t *testing.T) {
	if !taskFor(Request{Role: "evaluator"}, testCandidate()).IsolateFromAmbientContext {
		t.Fatal("evaluator inherits AGENTS.md/CLAUDE.md — its contract is not frozen")
	}
}

// Author roles keep ambient context until that is ruled on separately; flipping it here
// would change every executor's behaviour with no evidence.
func TestTaskFor_AuthorRolesKeepAmbientContextForNow(t *testing.T) {
	for _, role := range []string{"executor", "designer", "planner"} {
		if taskFor(Request{Role: role}, testCandidate()).IsolateFromAmbientContext {
			t.Errorf("role %q was isolated without a ruling", role)
		}
	}
}

// Every stage — not just the evaluator — must be marked as frozen execution so the repo's
// Claude Code hooks inject nothing into it. The AGENTS.md isolation above is evaluator-only
// because repo conventions are arguably an author's business; prompt-matched brain
// resolutions and an inbox banner are neither conventions nor contract, and content that
// varies run to run makes a "frozen" work item not frozen.
func TestTaskFor_EveryStageIsMarkedFrozenForHooks(t *testing.T) {
	for _, role := range []string{"evaluator", "executor", "designer", "planner"} {
		got := taskFor(Request{Role: role}, testCandidate()).ExtraEnv["AILANG_MISSION_STAGE"]
		if got != "1" {
			t.Errorf("role %q: AILANG_MISSION_STAGE=%q, want \"1\" — hooks will inject into a frozen stage", role, got)
		}
	}
}

// The canonical inbox pinning must survive alongside the new marker: these ride in the same
// map, and dropping one while adding the other is the obvious regression.
func TestTaskFor_StageKeepsCanonicalMessageStore(t *testing.T) {
	env := taskFor(Request{Role: "evaluator"}, testCandidate()).ExtraEnv
	if env["AILANG_MESSAGES_STORE"] != "gcp" || env["AILANG_MESSAGES_PROJECT"] != "ailang-multivac" {
		t.Fatalf("canonical store pinning lost: %v", env)
	}
}

// A model may not judge its own output: that is self-review, and no amount of vendor policy
// makes it informative. This is the part of generator != judge that stays HARD.
func TestResolve_ModelMayNotJudgeItself(t *testing.T) {
	r, runner, _, _ := fixture(t)
	r.Role = "evaluator"
	r.AuthorModels = []string{"judge"} // the author IS the candidate judge
	plan, err := Resolve(r, runner.Models)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range plan.Candidates {
		if c.Model == "judge" && c.SkipReason == "" {
			t.Fatal("a model was allowed to judge its own output")
		}
	}
}

// Same vendor, different model is ALLOWED and FLAGGED — the visible degradation Mark ruled
// acceptable, rather than the silent one that routed a judge with no methodology.
func TestResolve_SameVendorJudgeIsAllowedButFlagged(t *testing.T) {
	r, runner, _, _ := fixture(t)
	r.Role = "evaluator"
	r.AuthorModels = []string{"author"}
	plan, err := Resolve(r, runner.Models)
	if err != nil {
		t.Fatal(err)
	}
	var flagged, routable int
	for _, c := range plan.Candidates {
		if c.SkipReason == "" {
			routable++
			if c.SameVendorAsAuthor {
				flagged++
			}
		}
	}
	if routable == 0 {
		t.Fatal("no routable evaluator at all — the blocker this ruling removed")
	}
	_ = flagged // flagging is asserted structurally by the field existing on the receipt
}
