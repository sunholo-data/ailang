package iteration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// ReviewOptions describes a new immutable request. Empty authority produces a
// deliberately non-executable draft; calling this API never grants approval.
type ReviewOptions struct {
	NewID         string
	Limits        Limits
	BaseRevision  string
	Evaluator     string
	AuthorityRefs []AuthorityRef
}
type ReviewManifest struct {
	ProvenanceSources map[string]string              `json:"provenance_sources"`
	Source            coordinator.MissionWorkItemKey `json:"source"`
	SourceSpecDigest  string                         `json:"source_spec_digest"`
	AcceptanceDigests map[string]string              `json:"acceptance_digests"`
	Artifacts         []ArtifactRef                  `json:"artifacts"`
}
type ReviewDraft struct {
	Ready           bool                   `json:"ready"`
	Spec            Spec                   `json:"work_item"`
	MissingApproval []ArtifactRef          `json:"missing_approval"`
	Manifest        ReviewManifest         `json:"manifest"`
	Models          *modelreg.ModelsConfig `json:"-"`
}

// PrepareReview only reads local state/Git. It never claims an item or contacts a provider.
func PrepareReview(ctx context.Context, store *coordinator.SQLiteStore, key coordinator.MissionWorkItemKey, opts ReviewOptions) (ReviewDraft, error) {
	d := ReviewDraft{}
	if !validID(opts.NewID) || opts.NewID == key.WorkItemID {
		return d, fmt.Errorf("new distinct work item ID required")
	}
	if err := opts.Limits.validate(1800); err != nil {
		return d, err
	}
	item, err := store.GetMissionWorkItem(ctx, key)
	if err != nil {
		return d, err
	}
	if item.State != "failed" && item.State != "cancelled" {
		return d, fmt.Errorf("review continuation requires terminal failed/cancelled work; inspect or reconcile first")
	}
	if _, err = store.GetMissionWorkItem(ctx, coordinator.MissionWorkItemKey{MissionID: key.MissionID, WorkItemID: opts.NewID}); !errors.Is(err, sql.ErrNoRows) {
		return d, fmt.Errorf("successor ID already exists or cannot be inspected: %v", err)
	}
	for _, id := range item.StageIDs {
		child, e := store.GetMissionAttempt(ctx, coordinator.MissionAttemptKey{MissionID: key.MissionID, WorkItemID: key.WorkItemID, StageID: id})
		if errors.Is(e, sql.ErrNoRows) {
			continue
		}
		if e != nil {
			return d, e
		}
		if child.State == "prepared" || child.State == "running" || child.State == "needs_reconciliation" {
			return d, fmt.Errorf("stage %s still live or ambiguous", id)
		}
	}
	var snap Snapshot
	if digestBytes([]byte(item.SpecJSON)) != item.SpecDigest {
		return d, fmt.Errorf("saved snapshot digest mismatch")
	}
	if err = json.Unmarshal([]byte(item.SpecJSON), &snap); err != nil {
		return d, err
	}
	if err = snap.Spec.Validate(); err != nil {
		return d, fmt.Errorf("saved work contract invalid: %w", err)
	}
	if snap.Spec.MissionID != key.MissionID || snap.Spec.WorkItemID != key.WorkItemID {
		return d, fmt.Errorf("saved work identity mismatch")
	}
	if err = VerifyPrerequisites(ctx, snap.RepositoryPath, snap.Spec); err != nil {
		return d, fmt.Errorf("saved prerequisite authority: %w", err)
	}
	if snap.Models == nil {
		return d, fmt.Errorf("saved model registry missing")
	}
	d.Spec = snap.Spec
	d.Models = snap.Models
	d.Manifest = ReviewManifest{ProvenanceSources: map[string]string{}, Source: key, SourceSpecDigest: item.SpecDigest, AcceptanceDigests: map[string]string{}, Artifacts: []ArtifactRef{}}
	// Snapshot was unmarshalled into independent storage; never mutate original state.
	var candidate, reviewBase string
	for _, role := range []string{"designer", "planner", "executor"} {
		found := false
		for _, p := range d.Spec.Prerequisites {
			if p.Role == role {
				d.Manifest.ProvenanceSources[role] = "imported_prerequisite; original acceptance digest unavailable"
				d.Manifest.Artifacts = append(d.Manifest.Artifacts, p.Artifact)
				if role == "executor" {
					candidate = p.Artifact.Commit
					reviewBase = snap.Spec.ReviewBaseRevision
				}
				found = true
				break
			}
		}
		if found {
			continue
		}
		var stageID string
		for _, st := range snap.Spec.Stages {
			if st.Role == role {
				stageID = st.ID
				break
			}
		}
		a, e := store.GetMissionStageAcceptance(ctx, coordinator.MissionAttemptKey{MissionID: key.MissionID, WorkItemID: key.WorkItemID, StageID: stageID})
		if e != nil {
			return d, fmt.Errorf("accepted %s required: %w", role, e)
		}
		if digestBytes([]byte(a.AcceptanceJSON)) != a.AcceptanceDigest {
			return d, fmt.Errorf("%s acceptance digest mismatch", role)
		}
		var accepted AcceptedStage
		if e = json.Unmarshal([]byte(a.AcceptanceJSON), &accepted); e != nil {
			return d, e
		}
		ev := accepted.Evidence
		if ev == nil || len(ev.Artifacts) == 0 || len(ev.AuthorModels) == 0 || accepted.Report == nil || accepted.Report.Status != "execution_completed" {
			return d, fmt.Errorf("incomplete accepted %s", role)
		}
		child, e := store.GetMissionAttempt(ctx, a.MissionAttemptKey)
		if e != nil {
			return d, e
		}
		if e = validateReviewAcceptance(accepted, a, child, snap.Models); e != nil {
			return d, fmt.Errorf("%s acceptance: %w", role, e)
		}
		p := Prerequisite{Role: role, Artifact: ArtifactRef{Commit: ev.OutputRevision, Path: ev.Artifacts[0].Path, SHA256: ev.Artifacts[0].SHA256}, AuthorModels: append([]string(nil), ev.AuthorModels...)}
		for _, st := range snap.Spec.Stages {
			if st.ID == stageID {
				for _, ref := range st.AuthorityRefs {
					if ref.ArtifactDigest == p.Artifact.SHA256 {
						p.AuthorityRefs = append(p.AuthorityRefs, ref)
					}
				}
			}
		}
		d.Spec.Prerequisites = append(d.Spec.Prerequisites, p)
		d.Manifest.AcceptanceDigests[role] = a.AcceptanceDigest
		d.Manifest.ProvenanceSources[role] = "local_accepted_stage"
		for _, artifact := range ev.Artifacts {
			d.Manifest.Artifacts = append(d.Manifest.Artifacts, ArtifactRef{Commit: ev.OutputRevision, Path: artifact.Path, SHA256: artifact.SHA256})
		}
		if role == "executor" {
			candidate = ev.OutputRevision
			reviewBase = ev.Result.InputRevision
		}
	}
	if reviewBase == "" {
		return d, fmt.Errorf("imported executor has no review_base_revision; prepare from original failed author/evaluator work item with its accepted executor receipt")
	}
	if !validRevision(reviewBase) {
		return d, fmt.Errorf("saved review baseline is not a full commit")
	}
	if _, err = Git(ctx, snap.RepositoryPath, "merge-base", "--is-ancestor", reviewBase, candidate); err != nil {
		return d, fmt.Errorf("review baseline is not an ancestor of accepted candidate: %w", err)
	}
	if candidate == "" {
		return d, fmt.Errorf("accepted executor candidate required")
	}
	d.Spec.WorkItemID = opts.NewID
	d.Spec.BaseRevision = candidate
	d.Spec.ReviewBaseRevision = reviewBase
	d.Spec.Limits = opts.Limits
	if opts.BaseRevision != "" {
		if !validRevision(opts.BaseRevision) {
			return d, fmt.Errorf("base revision must be full commit")
		}
		d.Spec.BaseRevision = opts.BaseRevision
	}
	// A new authority commit may add approval documents, never change the candidate.
	if _, err = Git(ctx, snap.RepositoryPath, "merge-base", "--is-ancestor", candidate, d.Spec.BaseRevision); err != nil {
		return d, err
	}
	changes, e := Git(ctx, snap.RepositoryPath, "diff", "--no-ext-diff", "--name-only", "-z", candidate, d.Spec.BaseRevision, "--")
	if e != nil {
		return d, e
	}
	authorityPaths := map[string]bool{}
	for _, ref := range opts.AuthorityRefs {
		authorityPaths[ref.Path] = true
	}
	for _, p := range strings.Split(changes, "\x00") {
		if p != "" && (!authorityPaths[p] || PathAllowed(p, d.Spec.AllowedPaths)) {
			return d, fmt.Errorf("successor base changes non-authority or product path %q", p)
		}
	}
	for _, a := range d.Manifest.Artifacts {
		artifact, e := inspectArtifact(ctx, snap.RepositoryPath, a.Commit, a.Path)
		if e != nil {
			return d, e
		}
		if artifact.SHA256 != a.SHA256 {
			return d, fmt.Errorf("imported artifact hash mismatch")
		}
	}
	// Preserve existing designer/planner authority, but require a new review
	// approval of the executor output. Locally executed upstream roles may also
	// need their first output-bound approval before they can become prerequisites.
	d.Spec.Prerequisites[2].AuthorityRefs = nil
	if err := validateAuthorities(opts.AuthorityRefs); err != nil {
		return d, err
	}
	for _, ref := range opts.AuthorityRefs {
		matched := false
		for i := range d.Spec.Prerequisites {
			prior := &d.Spec.Prerequisites[i]
			if ref.ArtifactDigest == prior.Artifact.SHA256 {
				matched = true
				present := false
				for _, old := range prior.AuthorityRefs {
					if old == ref {
						present = true
						break
					}
				}
				if !present {
					prior.AuthorityRefs = append(prior.AuthorityRefs, ref)
				}
			}
		}
		if !matched {
			return d, fmt.Errorf("review authority binds no imported prerequisite artifact")
		}
	}
	var evaluator Stage
	for _, st := range snap.Spec.Stages {
		if st.Role == "evaluator" {
			evaluator = st
		}
	}
	if evaluator.ID == "" {
		return d, fmt.Errorf("saved evaluator stage missing")
	}
	evaluator.Limits = opts.Limits
	evaluator.AuthorityRefs = append([]AuthorityRef(nil), opts.AuthorityRefs...)
	evaluator.Instructions += "\nThis is an evaluator-only continuation. The author artifact is already accepted. Inspect the bound review packet, settle each frozen criterion and emit the result. Do not rerun the author. The current explicit request limits supersede historical plan budgets."
	d.Spec.Stages = []Stage{evaluator}
	if opts.Evaluator != "" {
		if _, e := d.Models.GetModel(opts.Evaluator); e != nil {
			return d, e
		}
		d.Models.Roles["evaluator"] = []string{opts.Evaluator}
	}
	routes, e := d.Models.ResolveRole("evaluator", modelreg.LaneLocal)
	if e != nil {
		return d, e
	}
	names := []string{}
	for _, route := range routes {
		names = append(names, route.FriendlyName)
	}
	authors := []string{}
	for _, p := range d.Spec.Prerequisites {
		authors = appendUnique(authors, p.AuthorModels...)
	}
	plan, e := dispatch.Resolve(dispatch.Request{Version: 1, MissionID: d.Spec.MissionID, WorkItemID: d.Spec.WorkItemID, StageID: evaluator.ID, AttemptID: "a1", Role: "evaluator", Workspace: snap.RepositoryPath, InputRevision: d.Spec.BaseRevision, Instructions: evaluator.Instructions, Models: names, AuthorModels: authors, TimeoutSeconds: opts.Limits.TimeoutSeconds, MaxTokens: opts.Limits.MaxTokens, MaxCostUSD: opts.Limits.MaxCostUSD}, d.Models)
	if e != nil {
		return d, e
	}
	eligible := false
	for _, route := range plan.Candidates {
		if route.SkipReason == "" {
			eligible = true
		}
	}
	if !eligible {
		return d, fmt.Errorf("no eligible independent evaluator route; choose an explicit independent evaluator")
	}
	for _, prior := range d.Spec.Prerequisites {
		if len(prior.AuthorityRefs) == 0 {
			d.MissingApproval = append(d.MissingApproval, prior.Artifact)
		}
	}
	if len(d.MissingApproval) > 0 {
		return d, nil
	}
	service := Service{RepositoryPath: snap.RepositoryPath, WorkspaceRoot: snap.WorkspaceRoot, Models: d.Models}
	if _, err = service.Plan(ctx, d.Spec); err != nil {
		return d, err
	}
	d.Ready = true
	return d, nil
}

func DecodeReviewAuthorities(r io.Reader) ([]AuthorityRef, error) {
	var refs []AuthorityRef
	if err := decodeStrict(r, 64*1024, &refs); err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return nil, fmt.Errorf("authority file must contain at least one reference")
	}
	if err := validateAuthorities(refs); err != nil {
		return nil, err
	}
	return refs, nil
}

// Validate provenance against the frozen child request and execution receipt,
// rather than trusting model names copied into an otherwise well-formed blob.
func validateReviewAcceptance(accepted AcceptedStage, a *coordinator.MissionStageAcceptance, child *coordinator.MissionAttempt, models *modelreg.ModelsConfig) error {
	ev := accepted.Evidence
	if child.State != "execution_completed" || child.RequestDigest != a.RequestDigest || child.OutcomeDigest != a.OutcomeDigest {
		return fmt.Errorf("acceptance does not bind completed child")
	}
	var req dispatch.Request
	if err := json.Unmarshal([]byte(child.RequestJSON), &req); err != nil {
		return err
	}
	if req.Digest() != child.RequestDigest || accepted.Report.RequestDigest != child.RequestDigest || accepted.Report.AttemptID != req.AttemptID {
		return fmt.Errorf("request provenance mismatch")
	}
	reportJSON, err := json.Marshal(accepted.Report)
	if err != nil {
		return err
	}
	if digestBytes(reportJSON) != child.OutcomeDigest {
		return fmt.Errorf("execution receipt mismatch")
	}
	route, err := completedRoute(accepted.Report, req.Models)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(route, ev.AuthorRoute) || len(ev.AuthorModels) != 1 || ev.AuthorModels[0] != route.Model {
		return fmt.Errorf("author provenance mismatch")
	}
	vendor, err := models.DispatchOriginVendor(route.Model)
	if err != nil {
		return err
	}
	if vendor != route.Vendor {
		return fmt.Errorf("author vendor differs from frozen registry")
	}
	if ev.Result.InputRevision != req.InputRevision || ev.Result.OutputRevision != ev.OutputRevision || ev.Result.RequestDigest != req.Digest() {
		return fmt.Errorf("candidate does not bind author request")
	}
	if digestBytes(ev.ResultBytes) != ev.ResultSHA256 {
		return fmt.Errorf("result bytes digest mismatch")
	}
	var result StageResult
	if err := json.Unmarshal(ev.ResultBytes, &result); err != nil {
		return err
	}
	if !reflect.DeepEqual(result, ev.Result) {
		return fmt.Errorf("result projection differs from exact bytes")
	}
	return nil
}
