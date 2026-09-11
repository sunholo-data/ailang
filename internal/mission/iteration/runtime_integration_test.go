package iteration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

type iterationExecutor struct {
	t      *testing.T
	calls  int
	fail   bool
	before func(*executor.Task)
	result func(*executor.Task, *executor.Result)
}

func (f *iterationExecutor) Name() string { return "pi" }
func (f *iterationExecutor) Capabilities() []executor.Capability {
	return []executor.Capability{executor.CapLocalWorkspace}
}
func (f *iterationExecutor) HealthCheck(context.Context) error             { return nil }
func (f *iterationExecutor) CostModel() *executor.CostModel                { return nil }
func (f *iterationExecutor) Close() error                                  { return nil }
func (f *iterationExecutor) GetExecutor(string) (executor.Executor, error) { return f, nil }
func (f *iterationExecutor) Execute(ctx context.Context, t *executor.Task) (*executor.Result, error) {
	return f.ExecuteStreaming(ctx, t, nil)
}
func (f *iterationExecutor) ExecuteStreaming(ctx context.Context, task *executor.Task, _ executor.EventHandler) (*executor.Result, error) {
	f.calls++
	if f.before != nil {
		f.before(task)
	}
	if f.fail {
		return nil, fmt.Errorf("provider lost after execution started")
	}
	base := gitTest(f.t, task.Workspace, "rev-parse", "HEAD")
	out := base
	outcome := "pass"
	criteria := map[string]CriterionResult{"clear": {Outcome: "pass", Evidence: "checked exact candidate"}}
	if task.Metadata["stage_id"] == "executor" {
		writeTest(f.t, task.Workspace, "docs/result.md", "Useful documentation\n")
		gitTest(f.t, task.Workspace, "add", "docs/result.md")
		gitTest(f.t, task.Workspace, "commit", "-qm", "docs: useful change")
		out = gitTest(f.t, task.Workspace, "rev-parse", "HEAD")
		outcome = "produced"
		criteria = nil
	}
	r := StageResult{Version: 1, RequestDigest: task.Metadata["request_digest"], InputRevision: base, OutputRevision: out, ArtifactPaths: []string{"docs/result.md"}, Outcome: outcome, Criteria: criteria}
	b, _ := json.Marshal(r)
	writeTest(f.t, task.Workspace, ResultFile, string(b))
	result := &executor.Result{Success: true, FinishReason: executor.FinishStop, Output: "artifact", InputTokens: 10, OutputTokens: 10, CostUSD: .001, CostProvenance: executor.CostMetered}
	if f.result != nil {
		f.result(task, result)
	}
	return result, ctx.Err()
}
func runtimeFixture(t *testing.T) (*Service, Spec, *iterationExecutor) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("runtime Unix-only")
	}
	repo := t.TempDir()
	gitTest(t, repo, "init", "-q")
	gitTest(t, repo, "remote", "add", "origin", "https://github.com/example/project.git")
	writeTest(t, repo, "docs/plan.md", "approved plan\n")
	writeTest(t, repo, "docs/approval.md", "decision-1: execute sprint\n")
	writeTest(t, repo, "docs/result.md", "old\n")
	gitTest(t, repo, "add", ".")
	gitTest(t, repo, "commit", "-qm", "approved baseline")
	spec := validSpec()
	spec.BaseRevision = gitTest(t, repo, "rev-parse", "HEAD")
	spec.Stages = spec.Stages[2:]
	for _, role := range []string{"designer", "planner"} {
		hash := digestBytes([]byte("approved plan\n"))
		spec.Prerequisites = append(spec.Prerequisites, Prerequisite{Role: role, Artifact: ArtifactRef{Commit: spec.BaseRevision, Path: "docs/plan.md", SHA256: hash}, AuthorModels: []string{"author"}, AuthorityRefs: []AuthorityRef{{Revision: spec.BaseRevision, Path: "docs/approval.md", Locator: "decision-1", SHA256: digestBytes([]byte("decision-1: execute sprint\n")), ArtifactDigest: hash}}})
	}
	cli := "pi"
	models := &modelreg.ModelsConfig{Models: map[string]modelreg.ModelConfig{}, Roles: map[string][]string{"executor": {"author"}, "evaluator": {"judge"}}}
	for _, name := range []string{"author", "judge"} {
		vendor := "deepseek"
		if name == "judge" {
			vendor = "minimax"
		}
		wire := "openrouter/" + vendor + "/model"
		models.Models[name] = modelreg.ModelConfig{Provider: "openrouter", APIName: vendor + "/model", AgentCLI: &cli, AgentModelName: &wire, Pricing: modelreg.Pricing{InputPer1K: .001, OutputPer1K: .002}}
	}
	store, err := coordinator.NewSQLiteStore(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	fake := &iterationExecutor{t: t}
	s := &Service{Store: store, Models: models, Factory: fake, RepositoryPath: repo, WorkspaceRoot: filepath.Join(t.TempDir(), "runtime"), Admit: func(context.Context, dispatch.Candidate) (dispatch.Admission, error) {
		return dispatch.Admission{Allowed: true, Policy: "fixture", ObservedAt: time.Now()}, nil
	}}
	return s, spec, fake
}
func TestIterationUsefulWorkAndIdempotentRestart(t *testing.T) {
	s, spec, f := runtimeFixture(t)
	ctx, c := context.WithTimeout(context.Background(), 20*time.Second)
	defer c()
	item, err := s.Run(ctx, spec)
	if err != nil {
		t.Fatalf("run: %+v %v", item, err)
	}
	if item.State != "completed" || item.NextStage != 2 || f.calls != 2 {
		t.Fatalf("false completion %+v calls=%d", item, f.calls)
	}
	item, err = s.Run(ctx, spec)
	if err != nil || item.State != "completed" || f.calls != 2 {
		t.Fatalf("repeated work %+v %v calls=%d", item, err, f.calls)
	}
	a, err := s.Store.GetMissionStageAcceptance(ctx, coordinator.MissionAttemptKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID, StageID: "evaluator"})
	if err != nil {
		t.Fatal(err)
	}
	var accepted AcceptedStage
	if err = json.Unmarshal([]byte(a.AcceptanceJSON), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.Evidence.AuthorRoute.Model != "judge" || len(accepted.Evidence.Checks) != 1 {
		t.Fatal("lost evaluation/check evidence")
	}
}
func TestIterationQuotaWaitThenResume(t *testing.T) {
	s, spec, f := runtimeFixture(t)
	allow := s.Admit
	s.Admit = func(context.Context, dispatch.Candidate) (dispatch.Admission, error) {
		return dispatch.Admission{Policy: "fixture", ObservedAt: time.Now(), Reason: "exhausted"}, nil
	}
	item, err := s.Run(context.Background(), spec)
	if err != nil || item.State != "waiting" || item.ReasonCode != "quota" || f.calls != 0 {
		t.Fatalf("quota %+v %v", item, err)
	}
	deadline := item.Deadline
	s.Admit = allow
	item, err = s.Run(context.Background(), spec)
	if err != nil || item.State != "completed" || item.Deadline != deadline {
		t.Fatalf("resume %+v %v", item, err)
	}
}
func TestIterationDryPlanAndFailureNeverRetry(t *testing.T) {
	s, spec, f := runtimeFixture(t)
	if _, err := s.Plan(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.WorkspaceRoot); !os.IsNotExist(err) {
		t.Fatal("dry plan created workspace")
	}
	f.fail = true
	item, err := s.Run(context.Background(), spec)
	if err == nil || item.State != "failed" || f.calls != 1 {
		t.Fatalf("failed execution %+v %v calls=%d", item, err, f.calls)
	}
	item, err = s.Run(context.Background(), spec)
	if err != nil || item.State != "failed" || f.calls != 1 {
		t.Fatalf("failed work reran %+v %v calls=%d", item, err, f.calls)
	}
}

func TestIterationQuotaChangesAfterChildClaim(t *testing.T) {
	s, spec, f := runtimeFixture(t)
	checks := 0
	s.Admit = func(context.Context, dispatch.Candidate) (dispatch.Admission, error) {
		checks++
		return dispatch.Admission{Allowed: checks == 1, Policy: "fixture", ObservedAt: time.Now(), Reason: "changed"}, nil
	}
	item, err := s.Run(context.Background(), spec)
	if err != nil || item.State != "waiting" || f.calls != 0 {
		t.Fatalf("late quota %+v %v calls=%d", item, err, f.calls)
	}
	child, err := s.Store.GetMissionAttempt(context.Background(), coordinator.MissionAttemptKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID, StageID: "executor"})
	if err != nil || child.State != "prepared" {
		t.Fatalf("lost prepared state %+v %v", child, err)
	}
	s.Admit = func(context.Context, dispatch.Candidate) (dispatch.Admission, error) {
		return dispatch.Admission{Allowed: true, Policy: "fixture", ObservedAt: time.Now()}, nil
	}
	item, err = s.Run(context.Background(), spec)
	if err != nil || item.State != "completed" || f.calls != 2 {
		t.Fatalf("prepared resume %+v %v calls=%d", item, err, f.calls)
	}
}
func TestIterationCancellationCannotAcceptLateResult(t *testing.T) {
	s, spec, f := runtimeFixture(t)
	f.before = func(*executor.Task) {
		key := coordinator.MissionWorkItemKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID}
		item, err := s.Store.GetMissionWorkItem(context.Background(), key)
		if err != nil {
			t.Fatal(err)
		}
		if err = s.Store.CancelMissionWorkItem(context.Background(), key, item.Version); err != nil {
			t.Fatal(err)
		}
	}
	item, err := s.Run(context.Background(), spec)
	if err == nil || item.State != "needs_reconciliation" || f.calls != 1 {
		t.Fatalf("cancel fence %+v %v calls=%d", item, err, f.calls)
	}
	if _, err := s.Store.GetMissionStageAcceptance(context.Background(), coordinator.MissionAttemptKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID, StageID: "executor"}); err == nil {
		t.Fatal("late acceptance")
	}
}

func TestIterationFinalUsageCannotBypassBudget(t *testing.T) {
	for _, kind := range []string{"unknown_cost", "tokens", "cost"} {
		t.Run(kind, func(t *testing.T) {
			s, spec, f := runtimeFixture(t)
			f.result = func(task *executor.Task, r *executor.Result) {
				if task.Metadata["stage_id"] != "evaluator" {
					return
				}
				switch kind {
				case "unknown_cost":
					r.CostProvenance = executor.CostProvenanceUnknown
				case "tokens":
					r.OutputTokens = spec.Limits.MaxTokens + 1
				case "cost":
					r.CostUSD = spec.Limits.MaxCostUSD + 1
				}
			}
			item, err := s.Run(context.Background(), spec)
			if err == nil || item.State != "failed" || item.NextStage != 1 {
				t.Fatalf("unproven final usage accepted: %+v %v", item, err)
			}
		})
	}
}
func TestIterationWaitingDeadlineExpiresDurably(t *testing.T) {
	s, spec, f := runtimeFixture(t)
	// The deadline has to outlive this test's own SETUP, not just be short.
	//
	// With a 1s budget the first Run below had to finish preflight and return
	// "waiting" inside the same second the deadline was measuring, so on a loaded
	// runner preflight overran and the first Run came back
	// failed/deadline_exceeded — failing at line ~250 with "wait: ... stage
	// deadline exhausted during preflight". That is the setup racing its own
	// clock, not the behaviour under test. Measured flaky ~1 in 3 in a
	// full-package run locally and red on macos in CI (run 34360400780).
	//
	// The budget only needs to be comfortably longer than preflight; the test
	// still expires it deliberately below, so nothing about what is asserted
	// changes.
	const timeoutSeconds = 5
	spec.Limits.TimeoutSeconds = timeoutSeconds
	for i := range spec.Stages {
		spec.Stages[i].Limits.TimeoutSeconds = timeoutSeconds
	}
	s.Admit = func(context.Context, dispatch.Candidate) (dispatch.Admission, error) {
		return dispatch.Admission{Policy: "fixture", ObservedAt: time.Now(), Reason: "quota"}, nil
	}
	item, err := s.Run(context.Background(), spec)
	if err != nil || item.State != "waiting" {
		t.Fatalf("wait: %+v %v", item, err)
	}
	// Sleep past the NEXT whole second, not deadline+10ms. Deadlines are stored
	// as integer Unix seconds (coordinator's missionNow truncates), so a 10ms
	// margin leaves time.Now().Unix() EQUAL to the deadline rather than past it,
	// which puts the expiry checks on a boundary for no benefit.
	time.Sleep(time.Until(time.Unix(item.Deadline+1, 0)))
	item, err = s.Run(context.Background(), spec)
	if item == nil || item.State != "failed" || f.calls != 0 {
		t.Fatalf("expired admission retained: %+v %v", item, err)
	}
}

func TestIterationDecisionRetainsProtocolWithoutAcceptance(t *testing.T) {
	s, spec, f := runtimeFixture(t)
	f.result = func(task *executor.Task, _ *executor.Result) {
		path := filepath.Join(task.Workspace, ResultFile)
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var result StageResult
		if err = json.Unmarshal(b, &result); err != nil {
			t.Fatal(err)
		}
		result.Outcome = "needs_decision"
		result.BlockingFindings = []string{"approval needed"}
		b, _ = json.Marshal(result)
		writeTest(t, task.Workspace, ResultFile, string(b))
	}
	item, err := s.Run(context.Background(), spec)
	if err != nil || item.State != "waiting" || item.ReasonCode != "decision" || item.NextStage != 0 || f.calls != 1 {
		t.Fatalf("decision lost: %+v %v", item, err)
	}
	files, _ := filepath.Glob(filepath.Join(s.WorkspaceRoot, spec.MissionID, spec.WorkItemID, "evidence", "*.json"))
	if len(files) != 1 {
		t.Fatalf("protocol not preserved: %v", files)
	}
	item, err = s.Run(context.Background(), spec)
	if err != nil || item.State != "waiting" || f.calls != 1 {
		t.Fatalf("decision redispatched: %+v %v", item, err)
	}
}
