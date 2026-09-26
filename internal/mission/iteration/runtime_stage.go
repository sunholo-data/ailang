package iteration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

type AcceptedStage struct {
	Evidence *Evidence        `json:"evidence"`
	Report   *dispatch.Report `json:"report"`
}

func digestBytes(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }

func (s *Service) request(ctx context.Context, spec Spec, stage Stage, index int, item *coordinator.MissionWorkItem) (dispatch.Request, error) {
	// Existing attempt requests are immutable; do not rebuild packets on recovery.
	if s.Store != nil {
		child, err := s.Store.GetMissionAttempt(ctx, coordinator.MissionAttemptKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID, StageID: stage.ID})
		if err == nil {
			var saved dispatch.Request
			if err := json.Unmarshal([]byte(child.RequestJSON), &saved); err != nil {
				return saved, err
			}
			if saved.Digest() != child.RequestDigest {
				return saved, fmt.Errorf("saved request digest mismatch")
			}
			return saved, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return dispatch.Request{}, err
		}
	}
	base := spec.BaseRevision
	authors := []string{}
	var reviewEvidence []*Evidence
	tokens := spec.Limits.MaxTokens
	cost := spec.Limits.MaxCostUSD
	for _, p := range spec.Prerequisites {
		authors = appendUnique(authors, p.AuthorModels...)
	}
	for _, previous := range spec.Stages[:index] {
		a, err := s.Store.GetMissionStageAcceptance(ctx, coordinator.MissionAttemptKey{MissionID: spec.MissionID, WorkItemID: spec.WorkItemID, StageID: previous.ID})
		if err != nil {
			return dispatch.Request{}, err
		}
		var accepted AcceptedStage
		if err = json.Unmarshal([]byte(a.AcceptanceJSON), &accepted); err != nil {
			return dispatch.Request{}, err
		}
		if accepted.Evidence == nil || accepted.Report == nil || accepted.Report.Result == nil {
			return dispatch.Request{}, fmt.Errorf("incomplete accepted stage")
		}
		reviewEvidence = append(reviewEvidence, accepted.Evidence)
		base = accepted.Evidence.OutputRevision
		authors = appendUnique(authors, accepted.Evidence.AuthorModels...)
		result := accepted.Report.Result
		tokens -= result.InputTokens + result.OutputTokens
		if result.CostProvenance == executor.CostMetered {
			cost -= result.CostUSD
		}
		if result.CostProvenance == executor.CostProvenanceUnknown && accepted.Evidence.AuthorRoute.Transport == "openrouter" {
			return dispatch.Request{}, fmt.Errorf("unknown metered cost blocks further work")
		}
	}
	if tokens <= 0 || cost <= 0 {
		return dispatch.Request{}, fmt.Errorf("iteration resource budget exhausted")
	}
	routes, err := s.Models.ResolveRole(stage.Role, modelreg.LaneLocal)
	if err != nil {
		return dispatch.Request{}, err
	}
	names := []string{}
	for _, route := range routes {
		names = append(names, route.FriendlyName)
	}
	instruction, err := json.Marshal(struct {
		Brief    string      `json:"brief"`
		Criteria []Criterion `json:"acceptance_criteria"`
		Required []string    `json:"required_artifacts"`
		Allowed  []string    `json:"allowed_paths"`
	}{spec.Brief, spec.AcceptanceCriteria, stage.RequiredArtifacts, spec.AllowedPaths})
	if err != nil {
		return dispatch.Request{}, err
	}
	limits := stage.Limits
	if limits.MaxTokens > tokens {
		limits.MaxTokens = tokens
	}
	if limits.MaxCostUSD > cost {
		limits.MaxCostUSD = cost
	}
	req := dispatch.Request{Version: 1, MissionID: spec.MissionID, WorkItemID: spec.WorkItemID, StageID: stage.ID, AttemptID: "a1", Role: stage.Role, Workspace: filepath.Join(s.WorkspaceRoot, spec.MissionID, spec.WorkItemID, "stages", stage.ID), InputRevision: base, Instructions: stage.Instructions + "\nFrozen work contract: " + string(instruction) + "\n" + resultInstructions, Models: names, TimeoutSeconds: limits.TimeoutSeconds, MaxTokens: limits.MaxTokens, MaxCostUSD: limits.MaxCostUSD}
	if stage.Role == "evaluator" {
		req.AuthorModels = authors
		packet, path, err := s.reviewPacket(ctx, spec, stage, base, reviewEvidence)
		if err != nil {
			return dispatch.Request{}, err
		}
		req.Instructions = focusedReviewInstructions + "\nStage instructions: " + stage.Instructions + "\nReview packet path: " + path + "\nReview packet SHA256: " + digestBytes([]byte(packet)) + "\n" + packet
	}
	return req, nil
}

const resultInstructions = `Commit product artifacts, then write untracked stage-result.json (never commit it). Strict JSON keys: version (1), request_digest (from system prompt), input_revision (full supplied commit), output_revision (full resulting HEAD), artifact_paths (array), outcome (produced for authors, pass/fail for evaluator, or needs_decision when human input is required), criteria (evaluator: map every frozen ID to {outcome:pass|fail,evidence:nonempty string}; authors: {}), blocking_findings (array). Evaluator must preserve HEAD and tracked files. Do not publish, merge, change policy or spawn additional author/reviewer roles. Execution success is not artifact acceptance.`

func (s *Service) ensureWorkspace(ctx context.Context, path, base string) error {
	if err := s.workspacePlacement(path); err != nil {
		return err
	}
	if _, err := os.Lstat(path); err == nil {
		if err := s.workspaceIdentity(ctx, path); err != nil {
			return err
		}
		out, e := runtimeGit(ctx, path, "status", "--porcelain", "--untracked-files=all")
		if e != nil {
			return e
		}
		if strings.TrimSpace(out) != "" {
			return fmt.Errorf("unstarted workspace is dirty; preserve and inspect %s", path)
		}
		head, e := runtimeGit(ctx, path, "rev-parse", "HEAD")
		if e != nil || strings.TrimSpace(head) != base {
			return fmt.Errorf("workspace base changed")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if _, err := runtimeGit(ctx, s.RepositoryPath, "worktree", "add", "--detach", path, base); err != nil {
		return err
	}
	return s.workspaceIdentity(ctx, path)
}

// Check placement before touching Git: neither the leaf nor its ancestors may
// redirect an admitted stage into the source checkout or another worktree.
func (s *Service) workspacePlacement(path string) error {
	root, err := filepath.Abs(s.WorkspaceRoot)
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("stage workspace must be beneath runtime root")
	}
	canonicalRoot, err := resolvePlacement(root)
	if err != nil {
		return err
	}
	canonicalPath, err := resolvePlacement(abs)
	if err != nil {
		return err
	}
	if canonicalPath != filepath.Join(canonicalRoot, rel) {
		return fmt.Errorf("stage workspace path redirects through a symlink")
	}
	source, err := filepath.EvalSymlinks(s.RepositoryPath)
	if err != nil {
		return err
	}
	source, err = filepath.Abs(source)
	if err != nil {
		return err
	}
	if within(source, canonicalPath) || within(canonicalPath, source) {
		return fmt.Errorf("stage workspace must be separate from source repository")
	}
	return nil
}

// A matching HEAD is insufficient: require a distinct linked worktree owned by
// the pinned repository and an administrative back-reference to this exact path.
func (s *Service) workspaceIdentity(ctx context.Context, path string) error {
	if err := s.workspacePlacement(path); err != nil {
		return err
	}
	actual, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	actual, err = filepath.Abs(actual)
	if err != nil {
		return err
	}
	gitPath := func(repo string, args ...string) (string, error) {
		out, e := runtimeGit(ctx, repo, args...)
		if e != nil {
			return "", e
		}
		p := strings.TrimSpace(out)
		if !filepath.IsAbs(p) {
			p = filepath.Join(repo, p)
		}
		p, e = filepath.EvalSymlinks(p)
		if e != nil {
			return "", e
		}
		return filepath.Abs(p)
	}
	top, err := gitPath(path, "rev-parse", "--show-toplevel")
	if err != nil || top != actual {
		return fmt.Errorf("stage workspace is not its own Git root: %v", err)
	}
	gitdir, err := gitPath(path, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return err
	}
	common, err := gitPath(path, "rev-parse", "--git-common-dir")
	if err != nil {
		return err
	}
	sourceCommon, err := gitPath(s.RepositoryPath, "rev-parse", "--git-common-dir")
	if err != nil {
		return err
	}
	sourceGit, err := gitPath(s.RepositoryPath, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return err
	}
	if common != sourceCommon || gitdir == common || gitdir == sourceGit {
		return fmt.Errorf("stage workspace is not a distinct linked worktree of source repository")
	}
	entry, err := os.Lstat(filepath.Join(path, ".git"))
	if err != nil || !entry.Mode().IsRegular() {
		return fmt.Errorf("stage worktree requires a regular .git administrative link")
	}
	backref, err := os.ReadFile(filepath.Join(gitdir, "gitdir"))
	if err != nil {
		return err
	}
	back, err := filepath.EvalSymlinks(strings.TrimSpace(string(backref)))
	if err != nil {
		return err
	}
	if back != filepath.Join(actual, ".git") {
		return fmt.Errorf("stage worktree administrative link belongs to another workspace")
	}
	return nil
}
func runtimeGit(ctx context.Context, repo string, args ...string) (string, error) {
	return Git(ctx, repo, args...)
}

// Stage expiry is a durable transition. A fresh bounded context persists it
// even when the stage context has already expired; live ambiguity stays held.
func (s *Service) expireStage(ctx context.Context, key coordinator.MissionWorkItemKey, owner, stageID string, cause error) error {
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return errors.Join(cause, s.Store.ExpireMissionStage(finish, key, owner, stageID))
}

func (s *Service) execute(ctx context.Context, key coordinator.MissionWorkItemKey, owner string, req dispatch.Request) (*coordinator.MissionAttempt, error) {
	deadline, err := s.Store.BeginMissionStage(ctx, key, owner, req.StageID, req.TimeoutSeconds)
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithDeadline(ctx, time.Unix(deadline, 0))
	defer cancel()
	if err := runCtx.Err(); err != nil {
		return nil, s.expireStage(ctx, key, owner, req.StageID, fmt.Errorf("stage deadline exhausted before dispatch: %w", err))
	}
	dir := filepath.Join(s.WorkspaceRoot, key.MissionID, key.WorkItemID, "receipts")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	journal, err := dispatch.OpenJournal(filepath.Join(dir, req.StageID+"-"+hex.EncodeToString(nonce)+".jsonl"))
	if err != nil {
		return nil, err
	}
	defer journal.Close()
	runner := dispatch.Runner{Models: s.Models, Executors: s.Factory, Admit: s.Admit, Record: journal.Record}
	if err := runner.Preflight(runCtx, req); err != nil {
		if runCtx.Err() != nil {
			return nil, s.expireStage(ctx, key, owner, req.StageID, fmt.Errorf("stage deadline exhausted during preflight: %w", runCtx.Err()))
		}
		reason := "availability"
		if strings.Contains(err.Error(), "admission") {
			reason = "quota"
		}
		return nil, s.Store.WaitMissionWorkItem(ctx, key, owner, reason, err.Error())
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	spec := coordinator.MissionAttemptSpec{MissionID: key.MissionID, WorkItemID: key.WorkItemID, StageID: req.StageID, AttemptID: req.AttemptID, RequestJSON: string(body), RequestDigest: req.Digest()}
	child, err := s.Store.ClaimMissionChild(ctx, key, owner, spec, 30)
	if err != nil {
		return nil, err
	}
	s.at("prepared")
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
				if s.Store.RenewMissionAttempt(runCtx, spec.Key(), child.OwnerToken, 30) != nil {
					cancel()
					return
				}
			}
		}
	}()
	runner.Record = func(event dispatch.Event) error {
		if event.Kind == "dispatching" {
			if err := s.ensureWorkspace(runCtx, req.Workspace, req.InputRevision); err != nil {
				return err
			}
		}
		if err := journal.Record(event); err != nil {
			return err
		}
		switch event.Kind {
		case "dispatching":
			if err := s.Store.StartMissionChild(runCtx, key, owner, spec.Key(), child.OwnerToken); err != nil {
				return err
			}
			s.at("dispatching")
			return nil
		case "finished":
			if event.Report == nil {
				return fmt.Errorf("completion missing report")
			}
			if event.Report.Status == "blocked" {
				return nil
			}
			outcome, err := json.Marshal(event.Report)
			if err != nil {
				return err
			}
			finish, c := context.WithTimeout(context.Background(), 5*time.Second)
			defer c()
			s.at("receipt_finished")
			if err := s.Store.CompleteMissionAttempt(finish, spec.Key(), child.OwnerToken, event.Report.Status, string(outcome)); err != nil {
				return err
			}
			s.at("execution_completed")
			return nil
		}
		return nil
	}
	report, runErr := runner.Run(runCtx, req)
	close(stop)
	<-done
	finish, c := context.WithTimeout(context.Background(), 5*time.Second)
	defer c()
	if report != nil && report.Status == "blocked" {
		if runCtx.Err() != nil {
			return nil, s.expireStage(ctx, key, owner, req.StageID, runCtx.Err())
		}
		reason := "availability"
		var findings []string
		dispatchRejected := false
		for _, attempt := range report.Attempts {
			if strings.HasPrefix(attempt.Reason, "admission ") || strings.HasPrefix(attempt.Reason, "quota admission:") {
				reason = "quota"
			}
			if attempt.Reason != "" {
				findings = append(findings, attempt.Candidate.Model+": "+attempt.Reason)
			}
			if attempt.Status == "executing" {
				dispatchRejected = true
			}
		}
		action := "candidate blocked before execution; inspect receipt and resume after resolving: " + strings.Join(findings, "; ")
		releaseErr := s.Store.ReleaseMissionPreparedChild(finish, key, owner, spec.Key(), child.OwnerToken, reason, action)
		if dispatchRejected {
			return nil, errors.Join(runErr, releaseErr)
		}
		return nil, releaseErr
	}
	got, getErr := s.Store.GetMissionAttempt(finish, spec.Key())
	if runErr != nil {
		if got != nil && got.State == "execution_failed" {
			return nil, s.fail(finish, key, owner, runErr)
		}
		return got, errors.Join(runErr, getErr)
	}
	return got, getErr
}
