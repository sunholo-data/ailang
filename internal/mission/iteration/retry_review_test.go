package iteration

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRetryReviewReusesAcceptedAuthorAndRequiresNewAuthority(t *testing.T) {
	s, spec, f := runtimeFixture(t)
	f.before = func(task *executor.Task) {
		if task.Metadata["stage_id"] == "evaluator" {
			f.fail = true
		}
	}
	item, err := s.Run(context.Background(), spec)
	if err == nil || item.State != "failed" {
		t.Fatalf("expected failed reviewer: %v %+v", err, item)
	}
	key := coordinator.MissionWorkItemKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID}
	before := item.SpecJSON
	draft, err := PrepareReview(context.Background(), s.Store, key, ReviewOptions{NewID: "review-again", Limits: Limits{1200, 100000, 2}})
	if err != nil {
		t.Fatal(err)
	}
	if draft.Ready || len(draft.MissingApproval) != 1 || len(draft.Spec.Stages) != 1 || len(draft.Spec.Prerequisites) != 3 {
		t.Fatalf("bad draft %+v", draft)
	}
	if !reflect.DeepEqual(draft.Spec.AcceptanceCriteria, spec.AcceptanceCriteria) || !reflect.DeepEqual(draft.Spec.Verification, spec.Verification) || !reflect.DeepEqual(draft.Spec.AllowedPaths, spec.AllowedPaths) {
		t.Fatal("changed frozen work scope")
	}
	if draft.Spec.Prerequisites[2].AuthorModels[0] != "author" || f.calls != 2 {
		t.Fatal("lost actual author or dispatched")
	}
	if draft.Spec.Validate() == nil {
		t.Fatal("unapproved draft became executable")
	}
	current, _ := s.Store.GetMissionWorkItem(context.Background(), key)
	if current.SpecJSON != before {
		t.Fatal("mutated source")
	}
	candidate := draft.Spec.BaseRevision
	gitTest(t, s.RepositoryPath, "merge", "--ff-only", candidate)
	writeTest(t, s.RepositoryPath, "decisions/review-approval.md", "review-again approved\n")
	gitTest(t, s.RepositoryPath, "add", "decisions/review-approval.md")
	gitTest(t, s.RepositoryPath, "commit", "-qm", "approve review")
	base := gitTest(t, s.RepositoryPath, "rev-parse", "HEAD")
	ref := AuthorityRef{Revision: base, Path: "decisions/review-approval.md", Locator: "review-again approved", SHA256: shaBytes([]byte("review-again approved\n")), ArtifactDigest: draft.Spec.Prerequisites[2].Artifact.SHA256}
	ready, err := PrepareReview(context.Background(), s.Store, key, ReviewOptions{NewID: "review-again", Limits: Limits{1200, 100000, 2}, BaseRevision: base, AuthorityRefs: []AuthorityRef{ref}})
	if err != nil {
		t.Fatal(err)
	}
	if !ready.Ready || ready.Spec.Validate() != nil {
		t.Fatal("approved successor invalid")
	}
	if ready.Spec.Stages[0].Limits.MaxTokens != 100000 {
		t.Fatal("lost new budget")
	}
	f.fail = false
	f.before = nil
	// Service retains its original repository, and only the evaluator executes.
	s.Models = ready.Models
	done, err := s.Run(context.Background(), ready.Spec)
	if err != nil || done.State != "completed" || f.calls != 3 {
		t.Fatalf("retry: %v %+v calls %d", err, done, f.calls)
	}
}
func TestRetryReviewRefusesNonterminal(t *testing.T) {
	s, spec, _ := runtimeFixture(t)
	_, err := PrepareReview(context.Background(), s.Store, coordinator.MissionWorkItemKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID}, ReviewOptions{NewID: "new", Limits: Limits{100, 1000, 1}})
	if err == nil {
		t.Fatal("missing work accepted")
	}
}

func TestRetryReviewRefusesAuthorAsEvaluator(t *testing.T) {
	s, spec, f := runtimeFixture(t)
	f.before = func(task *executor.Task) {
		if task.Metadata["stage_id"] == "evaluator" {
			f.fail = true
		}
	}
	_, _ = s.Run(context.Background(), spec)
	_, err := PrepareReview(context.Background(), s.Store, coordinator.MissionWorkItemKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID}, ReviewOptions{NewID: "bad-reviewer", Limits: Limits{100, 10000, 1}, Evaluator: "author"})
	if err == nil || !strings.Contains(err.Error(), "eligible independent evaluator") {
		t.Fatalf("same vendor admitted: %v", err)
	}
}

func TestRetryReviewActualNonterminalAndAmbiguous(t *testing.T) {
	for _, phase := range []string{"prepared", "dispatching"} {
		t.Run(phase, func(t *testing.T) {
			s, spec, _ := runtimeFixture(t)
			s.checkpoint = func(got string) {
				if got == phase {
					panic("attended crash fixture")
				}
			}
			func() {
				defer func() {
					if recover() == nil {
						t.Error("did not reach checkpoint")
					}
				}()
				_, _ = s.Run(context.Background(), spec)
			}()
			key := coordinator.MissionWorkItemKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID}
			current, err := s.Store.GetMissionWorkItem(context.Background(), key)
			if err != nil {
				t.Fatal(err)
			}
			if current.State == "failed" {
				t.Fatal("fixture not live")
			}
			opts := ReviewOptions{NewID: "later", Limits: Limits{100, 10000, 1}}
			if _, err := PrepareReview(context.Background(), s.Store, key, opts); err == nil {
				t.Fatal("live parent accepted")
			}
			if phase == "dispatching" {
				if err := s.Store.CancelMissionWorkItem(context.Background(), key, current.Version); err != nil {
					t.Fatal(err)
				}
				current, err = s.Store.GetMissionWorkItem(context.Background(), key)
				if err != nil || current.State != "needs_reconciliation" {
					t.Fatalf("not ambiguous: %+v %v", current, err)
				}
				if _, err := PrepareReview(context.Background(), s.Store, key, opts); err == nil {
					t.Fatal("ambiguous parent accepted")
				}
			}
		})
	}
}

func TestRetryReviewImportedBaselineRequiredAndVerified(t *testing.T) {
	for _, baseline := range []string{"", strings.Repeat("a", 40)} {
		t.Run(baseline, func(t *testing.T) {
			s, spec, _ := runtimeFixture(t)
			spec.Stages = spec.Stages[1:]
			p := spec.Prerequisites[0]
			p.Role = "executor"
			spec.Prerequisites = append(spec.Prerequisites, p)
			spec.ReviewBaseRevision = baseline
			snapshot := Snapshot{Spec: spec, Models: s.Models, RepositoryPath: s.RepositoryPath, WorkspaceRoot: s.WorkspaceRoot}
			body, _ := json.Marshal(snapshot)
			key := coordinator.MissionWorkItemKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID}
			item, err := s.Store.ClaimMissionWorkItem(context.Background(), coordinator.MissionWorkItemSpec{MissionWorkItemKey: key, SpecJSON: string(body), SpecDigest: digestBytes(body), StageIDs: []string{"evaluator"}, TimeoutSeconds: 100}, 30)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Store.FinishMissionWorkItem(context.Background(), key, item.OwnerToken, "failed", "fixture failure"); err != nil {
				t.Fatal(err)
			}
			_, err = PrepareReview(context.Background(), s.Store, key, ReviewOptions{NewID: "new-review", Limits: Limits{100, 10000, 1}})
			if err == nil {
				t.Fatal("missing/invalid review baseline accepted")
			}
			if baseline == "" && !strings.Contains(err.Error(), "no review_base_revision") {
				t.Fatalf("not actionable: %v", err)
			}
		})
	}
}

func TestRetryReviewRejectsTamperedAuthorProvenance(t *testing.T) {
	s, spec, f := runtimeFixture(t)
	f.before = func(task *executor.Task) {
		if task.Metadata["stage_id"] == "evaluator" {
			f.fail = true
		}
	}
	_, _ = s.Run(context.Background(), spec)
	key := coordinator.MissionAttemptKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID, StageID: "executor"}
	a, err := s.Store.GetMissionStageAcceptance(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	child, err := s.Store.GetMissionAttempt(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	var accepted AcceptedStage
	if err := json.Unmarshal([]byte(a.AcceptanceJSON), &accepted); err != nil {
		t.Fatal(err)
	}
	if err := validateReviewAcceptance(accepted, a, child, s.Models); err != nil {
		t.Fatalf("valid acceptance failed: %v", err)
	}
	accepted.Evidence.AuthorModels = []string{"judge"}
	if err := validateReviewAcceptance(accepted, a, child, s.Models); err == nil {
		t.Fatal("rewritten author accepted")
	}
	accepted.Evidence.AuthorModels = []string{"author"}
	accepted.Evidence.Result.InputRevision = strings.Repeat("a", 40)
	if err := validateReviewAcceptance(accepted, a, child, s.Models); err == nil {
		t.Fatal("rewritten review baseline accepted")
	}
}

type fullReviewExecutor struct {
	*iterationExecutor
	failReview bool
}

func (f *fullReviewExecutor) GetExecutor(string) (executor.Executor, error) { return f, nil }
func (f *fullReviewExecutor) ExecuteStreaming(ctx context.Context, task *executor.Task, _ executor.EventHandler) (*executor.Result, error) {
	f.calls++
	role := task.Metadata["stage_id"]
	if role == "evaluator" && f.failReview {
		return nil, fmt.Errorf("fixture evaluator failed")
	}
	base := gitTest(f.t, task.Workspace, "rev-parse", "HEAD")
	result := StageResult{Version: 1, RequestDigest: task.Metadata["request_digest"], InputRevision: base, OutputRevision: base, ArtifactPaths: []string{"docs/executor.md"}, Outcome: "pass", Criteria: map[string]CriterionResult{"clear": {Outcome: "pass", Evidence: "checked author outputs"}}}
	if role != "evaluator" {
		path := "docs/" + role + ".md"
		writeTest(f.t, task.Workspace, path, "Accepted "+role+" output\n")
		gitTest(f.t, task.Workspace, "add", path)
		gitTest(f.t, task.Workspace, "commit", "-qm", role)
		result.OutputRevision = gitTest(f.t, task.Workspace, "rev-parse", "HEAD")
		result.Outcome = "produced"
		result.Criteria = nil
		result.ArtifactPaths = []string{path}
	}
	body, _ := json.Marshal(result)
	writeTest(f.t, task.Workspace, ResultFile, string(body))
	return &executor.Result{Success: true, FinishReason: executor.FinishStop, Output: "artifact", InputTokens: 10, OutputTokens: 10, CostUSD: .001, CostProvenance: executor.CostMetered}, ctx.Err()
}

func TestRetryReviewAllExecutedAuthorsNeedBoundApprovals(t *testing.T) {
	s, initial, f := runtimeFixture(t)
	spec := validSpec()
	spec.BaseRevision = initial.BaseRevision
	s.Models.Roles["designer"] = []string{"author"}
	s.Models.Roles["planner"] = []string{"author"}
	for i := range spec.Stages {
		role := spec.Stages[i].Role
		if role == "evaluator" {
			role = "executor"
		}
		spec.Stages[i].RequiredArtifacts = []string{"docs/" + role + ".md"}
	}
	full := &fullReviewExecutor{iterationExecutor: f, failReview: true}
	s.Factory = full
	item, err := seedFullReviewAcceptedStages(t, s, spec)
	if err == nil || item.State != "failed" || f.calls != 4 {
		t.Fatalf("full author fixture: %+v %v calls=%d", item, err, f.calls)
	}
	key := coordinator.MissionWorkItemKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID}
	opts := ReviewOptions{NewID: "full-review", Limits: Limits{100, 10000, 1}}
	draft, err := PrepareReview(context.Background(), s.Store, key, opts)
	if err != nil {
		t.Fatal(err)
	}
	if draft.Ready || len(draft.MissingApproval) != 3 {
		t.Fatalf("must report all three missing output approvals: %+v", draft.MissingApproval)
	}
	gitTest(t, s.RepositoryPath, "merge", "--ff-only", draft.Spec.BaseRevision)
	approval := "full-review approved\n"
	writeTest(t, s.RepositoryPath, "decisions/full-review.md", approval)
	gitTest(t, s.RepositoryPath, "add", "decisions/full-review.md")
	gitTest(t, s.RepositoryPath, "commit", "-qm", "approve imported outputs")
	opts.BaseRevision = gitTest(t, s.RepositoryPath, "rev-parse", "HEAD")
	for _, artifact := range draft.MissingApproval {
		opts.AuthorityRefs = append(opts.AuthorityRefs, AuthorityRef{Revision: opts.BaseRevision, Path: "decisions/full-review.md", Locator: "full-review approved", SHA256: shaBytes([]byte(approval)), ArtifactDigest: artifact.SHA256})
	}
	partial := opts
	partial.AuthorityRefs = opts.AuthorityRefs[2:]
	missing, err := PrepareReview(context.Background(), s.Store, key, partial)
	if err != nil || missing.Ready || len(missing.MissingApproval) != 2 {
		t.Fatalf("partial approval: %+v %v", missing, err)
	}
	unrelated := opts
	unrelated.AuthorityRefs = append(append([]AuthorityRef(nil), opts.AuthorityRefs...), opts.AuthorityRefs[0])
	unrelated.AuthorityRefs[3].ArtifactDigest = strings.Repeat("a", 64)
	if _, err := PrepareReview(context.Background(), s.Store, key, unrelated); err == nil {
		t.Fatal("unrelated approval digest accepted")
	}
	ready, err := PrepareReview(context.Background(), s.Store, key, opts)
	if err != nil || !ready.Ready {
		t.Fatalf("all approvals rejected: %+v %v", ready, err)
	}
	full.failReview = false
	s.Models = ready.Models
	done, err := s.Run(context.Background(), ready.Spec)
	if err != nil || done.State != "completed" || f.calls != 5 {
		t.Fatalf("evaluator-only rerun: %+v %v calls=%d", done, err, f.calls)
	}
	old, err := s.Store.GetMissionWorkItem(context.Background(), key)
	if err != nil || old.SpecJSON != item.SpecJSON || old.State != "failed" {
		t.Fatal("source modified")
	}
}

// Seed accepted outputs through coordinator APIs, deliberately without the
// runtime's designer/planner approval gate. This models an imported historical
// acceptance, not an authority bypass in production execution.
func seedFullReviewAcceptedStages(t *testing.T, s *Service, spec Spec) (*coordinator.MissionWorkItem, error) {
	t.Helper()
	ctx := context.Background()
	snap, err := s.Plan(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(snap)
	key := coordinator.MissionWorkItemKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID}
	ids := []string{}
	for _, stage := range spec.Stages {
		ids = append(ids, stage.ID)
	}
	item, err := s.Store.ClaimMissionWorkItem(ctx, coordinator.MissionWorkItemSpec{MissionWorkItemKey: key, SpecJSON: string(body), SpecDigest: digestBytes(body), StageIDs: ids, TimeoutSeconds: 120}, 30)
	if err != nil {
		t.Fatal(err)
	}
	for i, stage := range spec.Stages {
		req, err := s.request(ctx, spec, stage, i, item)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.ensureWorkspace(ctx, req.Workspace, req.InputRevision); err != nil {
			t.Fatal(err)
		}
		child, err := s.execute(ctx, key, item.OwnerToken, req)
		if err != nil {
			current, getErr := s.Store.GetMissionWorkItem(ctx, key)
			if getErr != nil {
				t.Fatal(getErr)
			}
			return current, err
		}
		if child == nil {
			current, _ := s.Store.GetMissionWorkItem(ctx, key)
			t.Fatalf("fixture dispatch waiting: %+v", current)
		}
		var report dispatch.Report
		if err := json.Unmarshal([]byte(child.Outcome), &report); err != nil {
			t.Fatal(err)
		}
		evidence, err := ValidateArtifacts(ctx, spec, stage, req, &report, req.Workspace, filepath.Join(s.WorkspaceRoot, "seed-verification"))
		if err != nil {
			t.Fatal(err)
		}
		accepted, _ := json.Marshal(AcceptedStage{Evidence: evidence, Report: &report})
		if err := s.Store.AcceptMissionStage(ctx, key, item.OwnerToken, coordinator.MissionStageAcceptance{MissionAttemptKey: child.MissionAttemptSpec.Key(), RequestDigest: child.RequestDigest, OutcomeDigest: child.OutcomeDigest, AcceptanceJSON: string(accepted), AcceptanceDigest: digestBytes(accepted)}); err != nil {
			t.Fatal(err)
		}
	}
	return item, nil
}

func TestRetryReviewPreservesExecutedDesignPlanApprovals(t *testing.T) {
	s, initial, f := runtimeFixture(t)
	spec := validSpec()
	spec.BaseRevision = initial.BaseRevision
	s.Models.Roles["designer"] = []string{"author"}
	s.Models.Roles["planner"] = []string{"author"}
	for i := range spec.Stages {
		role := spec.Stages[i].Role
		if role == "evaluator" {
			role = "executor"
		}
		spec.Stages[i].RequiredArtifacts = []string{"docs/" + role + ".md"}
		if i < 2 {
			spec.Stages[i].AuthorityRefs = []AuthorityRef{{Revision: spec.BaseRevision, Path: "docs/approval.md", Locator: "decision-1", SHA256: shaBytes([]byte("decision-1: execute sprint\n")), ArtifactDigest: shaBytes([]byte("Accepted " + role + " output\n"))}}
		}
	}
	s.Factory = &fullReviewExecutor{iterationExecutor: f, failReview: true}
	item, err := s.Run(context.Background(), spec)
	if err == nil || item.State != "failed" || f.calls != 4 {
		t.Fatalf("full runtime did not reach failed review: %v calls=%d", err, f.calls)
	}
	draft, err := PrepareReview(context.Background(), s.Store, coordinator.MissionWorkItemKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID}, ReviewOptions{NewID: "retained-approval", Limits: Limits{100, 10000, 1}})
	if err != nil || draft.Ready || len(draft.MissingApproval) != 1 {
		t.Fatalf("draft: %v %+v", err, draft.MissingApproval)
	}
	for i := 0; i < 2; i++ {
		if !reflect.DeepEqual(draft.Spec.Prerequisites[i].AuthorityRefs, spec.Stages[i].AuthorityRefs) {
			t.Fatal("valid previous authority dropped")
		}
	}
}
