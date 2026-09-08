package iteration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// Snapshot is the exact execution input persisted at admission. Registry changes
// never silently change a resumed stage's route or pricing.
type Snapshot struct {
	Spec           Spec                   `json:"spec"`
	Models         *modelreg.ModelsConfig `json:"models"`
	RepositoryPath string                 `json:"repository_path"`
	WorkspaceRoot  string                 `json:"workspace_root"`
}

type Service struct {
	checkpoint func(string) // package-local crash fixture seam; production leaves nil

	Store          *coordinator.SQLiteStore
	Models         *modelreg.ModelsConfig
	Factory        dispatch.Factory
	Admit          func(context.Context, dispatch.Candidate) (dispatch.Admission, error)
	RepositoryPath string
	WorkspaceRoot  string
}

// Plan is read-only: model resolution and local Git verification, without probing.
func (s *Service) Plan(ctx context.Context, spec Spec) (Snapshot, error) {
	if err := spec.Validate(); err != nil {
		return Snapshot{}, err
	}
	if s.Models == nil {
		return Snapshot{}, fmt.Errorf("model registry required")
	}
	repo, err := filepath.EvalSymlinks(s.RepositoryPath)
	if err != nil {
		return Snapshot{}, err
	}
	repo, err = filepath.Abs(repo)
	if err != nil {
		return Snapshot{}, err
	}
	root, err := resolvePlacement(s.WorkspaceRoot)
	if err != nil || s.WorkspaceRoot == "" {
		return Snapshot{}, fmt.Errorf("workspace root required")
	}
	if within(repo, root) || within(root, repo) {
		return Snapshot{}, fmt.Errorf("workspace root must be outside the source repository")
	}
	if err := VerifyPrerequisites(ctx, repo, spec); err != nil {
		return Snapshot{}, err
	}
	for _, stage := range spec.Stages {
		routes, err := s.Models.ResolveRole(stage.Role, modelreg.LaneLocal)
		if err != nil {
			return Snapshot{}, err
		}
		names := make([]string, len(routes))
		for i, r := range routes {
			names[i] = r.FriendlyName
		}
		req := dispatch.Request{Version: 1, MissionID: spec.MissionID, WorkItemID: spec.WorkItemID, StageID: stage.ID, AttemptID: "a1", Role: stage.Role, Workspace: repo, InputRevision: spec.BaseRevision, Instructions: stage.Instructions, Models: names, TimeoutSeconds: stage.Limits.TimeoutSeconds, MaxTokens: stage.Limits.MaxTokens, MaxCostUSD: stage.Limits.MaxCostUSD}
		for _, prior := range spec.Prerequisites {
			req.AuthorModels = appendUnique(req.AuthorModels, prior.AuthorModels...)
		}
		// Future author routes have not executed; only validate model definitions now.
		if stage.Role == "evaluator" && len(req.AuthorModels) == 0 {
			req.AuthorModels = []string{names[0]}
		}
		if _, err := dispatch.Resolve(req, s.Models); err != nil {
			return Snapshot{}, err
		}
	}
	return Snapshot{spec, s.Models, repo, root}, nil
}

func (s *Service) Run(ctx context.Context, spec Spec) (*coordinator.MissionWorkItem, error) {
	if s.Store == nil || s.Factory == nil || s.Admit == nil {
		return nil, fmt.Errorf("store, executor factory and admission policy required")
	}
	if runtime.GOOS == "windows" {
		return nil, fmt.Errorf("mission iteration requires Unix receipt sync and process-tree cancellation")
	}
	key := coordinator.MissionWorkItemKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID}
	saved, err := s.Store.GetMissionWorkItem(ctx, key)
	var snapshot Snapshot
	if err == nil {
		if err = json.Unmarshal([]byte(saved.SpecJSON), &snapshot); err != nil {
			return saved, err
		}
		if snapshot.Spec.Digest() != spec.Digest() {
			return saved, fmt.Errorf("work item input changed; use reviewed successor")
		}
		repo, e := filepath.EvalSymlinks(s.RepositoryPath)
		if e != nil {
			return saved, e
		}
		repo, e = filepath.Abs(repo)
		if e != nil {
			return saved, e
		}
		root, e := resolvePlacement(s.WorkspaceRoot)
		if e != nil {
			return saved, e
		}
		if repo != snapshot.RepositoryPath || root != snapshot.WorkspaceRoot {
			return saved, fmt.Errorf("work item placement changed")
		}
		if saved.State == "completed" || saved.State == "failed" || saved.State == "cancelled" || saved.State == "needs_reconciliation" {
			return saved, nil
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		snapshot, err = s.Plan(ctx, spec)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}
	body, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(spec.Stages))
	for i, st := range spec.Stages {
		ids[i] = st.ID
	}
	admitted, err := s.Store.ClaimMissionWorkItem(ctx, coordinator.MissionWorkItemSpec{MissionWorkItemKey: key, SpecJSON: string(body), SpecDigest: digestBytes(body), StageIDs: ids, TimeoutSeconds: spec.Limits.TimeoutSeconds}, 30)
	if err != nil {
		return saved, err
	}
	local := *s
	local.Models = snapshot.Models
	local.RepositoryPath = snapshot.RepositoryPath
	local.WorkspaceRoot = snapshot.WorkspaceRoot
	runCtx, cancel := context.WithDeadline(ctx, time.Unix(admitted.Deadline, 0))
	defer cancel()
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-stop:
				return
			case <-runCtx.Done():
				return
			case <-tick.C:
				renew, c := context.WithTimeout(runCtx, 3*time.Second)
				err := s.Store.RenewMissionWorkItem(renew, key, admitted.OwnerToken, 30)
				c()
				if err != nil {
					cancel()
					return
				}
			}
		}
	}()
	runErr := local.advance(runCtx, spec, admitted)
	close(stop)
	<-done
	readCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
	defer c()
	if runCtx.Err() != nil && time.Now().Unix() >= admitted.Deadline {
		runErr = errors.Join(runErr, s.Store.ExpireMissionWorkItem(readCtx, key, admitted.OwnerToken))
	}
	current, readErr := s.Store.GetMissionWorkItem(readCtx, key)
	return current, errors.Join(runErr, readErr)
}

func appendUnique(dst []string, values ...string) []string {
	for _, v := range values {
		found := false
		for _, old := range dst {
			if old == v {
				found = true
				break
			}
		}
		if !found {
			dst = append(dst, v)
		}
	}
	return dst
}

func (s *Service) advance(ctx context.Context, spec Spec, item *coordinator.MissionWorkItem) error {
	key := coordinator.MissionWorkItemKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID}
	for index := item.NextStage; index < len(spec.Stages); index++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		stage := spec.Stages[index]
		deadline, err := s.Store.BeginMissionStage(ctx, key, item.OwnerToken, stage.ID, stage.Limits.TimeoutSeconds)
		if err != nil {
			return err
		}
		ctx, stageCancel := context.WithDeadline(ctx, time.Unix(deadline, 0))
		defer stageCancel()
		stageError := func(cause error) error {
			if ctx.Err() != nil && time.Now().Unix() >= deadline {
				return s.expireStage(ctx, key, item.OwnerToken, stage.ID, cause)
			}
			return cause
		}
		if err := ctx.Err(); err != nil {
			return s.expireStage(ctx, key, item.OwnerToken, stage.ID, err)
		}
		req, err := s.request(ctx, spec, stage, index, item)
		if err != nil {
			return s.fail(ctx, key, item.OwnerToken, err)
		}
		childKey := coordinator.MissionAttemptKey{MissionID: key.MissionID, WorkItemID: key.WorkItemID, StageID: stage.ID}
		child, err := s.Store.GetMissionAttempt(ctx, childKey)
		if errors.Is(err, sql.ErrNoRows) {
			child = nil
		} else if err != nil {
			return stageError(err)
		}
		if child != nil {
			if err := json.Unmarshal([]byte(child.RequestJSON), &req); err != nil {
				return stageError(err)
			}
			switch child.State {
			case "running", "needs_reconciliation", "cancelled":
				return s.Store.ReconcileMissionWorkItem(ctx, key)
			case "execution_failed":
				return s.fail(ctx, key, item.OwnerToken, fmt.Errorf("stage execution failed; review retained artifacts"))
			}
		}
		if child == nil || child.State == "prepared" {
			if err := s.ensureWorkspace(ctx, req.Workspace, req.InputRevision); err != nil {
				return s.fail(ctx, key, item.OwnerToken, err)
			}
			child, err = s.execute(ctx, key, item.OwnerToken, req)
			if err != nil {
				return stageError(err)
			}
			if child == nil {
				return nil
			} // Explicit waiting transition, no dispatched work.
		}
		if err = s.Store.MarkMissionWorkItemValidating(ctx, key, item.OwnerToken); err != nil {
			return stageError(err)
		}
		var report dispatch.Report
		if err = json.Unmarshal([]byte(child.Outcome), &report); err != nil {
			return stageError(err)
		}
		evidence, verifyErr := ValidateArtifacts(ctx, spec, stage, req, &report, req.Workspace, filepath.Join(s.WorkspaceRoot, spec.MissionID, spec.WorkItemID, "verification"))
		if evidence != nil {
			if err := s.saveEvidence(spec, stage, evidence); err != nil {
				return stageError(err)
			}
		}
		if errors.Is(verifyErr, ErrNeedsDecision) {
			return s.Store.WaitMissionWorkItem(ctx, key, item.OwnerToken, "decision", "review retained stage decision and create approved successor")
		}
		if verifyErr != nil {
			if ctx.Err() != nil && time.Now().Unix() >= deadline {
				return s.expireStage(ctx, key, item.OwnerToken, stage.ID, verifyErr)
			}
			return s.fail(ctx, key, item.OwnerToken, verifyErr)
		}
		if err := validateUsage(req, &report, evidence); err != nil {
			return s.fail(ctx, key, item.OwnerToken, err)
		}
		if stage.Role == "designer" || stage.Role == "planner" {
			approved := true
			for _, artifact := range evidence.Artifacts {
				found := false
				for _, ref := range stage.AuthorityRefs {
					if ref.ArtifactDigest == artifact.SHA256 {
						found = true
					}
				}
				if !found {
					approved = false
				}
			}
			if !approved {
				return s.Store.WaitMissionWorkItem(ctx, key, item.OwnerToken, "decision", "review produced artifact and supply approved prerequisite in a successor work item")
			}
		}
		data, err := json.Marshal(AcceptedStage{Evidence: evidence, Report: &report})
		if err != nil {
			return stageError(err)
		}
		if len(data) > 16*1024*1024 {
			return s.fail(ctx, key, item.OwnerToken, fmt.Errorf("accepted evidence exceeds 16 MiB; retained files require review"))
		}
		acceptance := coordinator.MissionStageAcceptance{MissionAttemptKey: childKey, RequestDigest: child.RequestDigest, OutcomeDigest: child.OutcomeDigest, AcceptanceJSON: string(data), AcceptanceDigest: digestBytes(data)}
		if err = s.Store.AcceptMissionStage(ctx, key, item.OwnerToken, acceptance); err != nil {
			return stageError(err)
		}
		s.at("accepted")
	}
	return s.Store.FinishMissionWorkItem(ctx, key, item.OwnerToken, "completed", "")
}

func (s *Service) fail(ctx context.Context, key coordinator.MissionWorkItemKey, owner string, cause error) error {
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return errors.Join(cause, s.Store.FinishMissionWorkItem(finish, key, owner, "failed", cause.Error()))
}
func (s *Service) saveEvidence(spec Spec, stage Stage, e *Evidence) error {
	dir := filepath.Join(s.WorkspaceRoot, spec.MissionID, spec.WorkItemID, "evidence")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	body, err := json.Marshal(e)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, stage.ID+"-"+digestBytes(body)+".json")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if errors.Is(err, os.ErrExist) {
		old, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		if string(old) != string(body) {
			return fmt.Errorf("evidence collision")
		}
		return nil
	}
	if err != nil {
		return err
	}
	_, err = f.Write(body)
	return errors.Join(err, f.Sync(), f.Close())
}

func (s *Service) at(phase string) {
	if s.checkpoint != nil {
		s.checkpoint(phase)
	}
}

func validateUsage(req dispatch.Request, report *dispatch.Report, evidence *Evidence) error {
	if report.Result == nil || evidence == nil {
		return fmt.Errorf("resource evidence missing")
	}
	result := report.Result
	if result.InputTokens < 0 || result.OutputTokens < 0 || result.InputTokens > req.MaxTokens || result.OutputTokens > req.MaxTokens-result.InputTokens {
		return fmt.Errorf("stage token cap exceeded or invalid usage")
	}
	if evidence.AuthorRoute.Transport == "openrouter" && result.CostProvenance != executor.CostMetered {
		return fmt.Errorf("unknown metered cost blocks acceptance")
	}
	if math.IsNaN(result.CostUSD) || math.IsInf(result.CostUSD, 0) || result.CostUSD < 0 || result.CostUSD > req.MaxCostUSD {
		return fmt.Errorf("stage cost cap exceeded or invalid usage")
	}
	return nil
}
