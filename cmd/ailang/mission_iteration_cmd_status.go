package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"github.com/sunholo-data/ailang/internal/mission/iteration"
)

type iterationStatus struct {
	MissionID         string                    `json:"mission_id"`
	WorkItemID        string                    `json:"work_item_id"`
	Phase             string                    `json:"phase"`
	ReasonCode        string                    `json:"reason_code"`
	NextAction        string                    `json:"next_action"`
	Diagnostic        iterationDiagnostic       `json:"diagnostic"`
	Stage             string                    `json:"stage"`
	Version           int64                     `json:"version"`
	LeaseUntil        int64                     `json:"lease_until"`
	Deadline          int64                     `json:"deadline"`
	Evidence          []iterationEvidenceStatus `json:"evidence"`
	Routes            []dispatch.Candidate      `json:"routes"`
	Stages            []iterationStageStatus    `json:"stages"`
	EvidenceDirectory string                    `json:"evidence_directory"`
	ReceiptDirectory  string                    `json:"receipt_directory"`
}

type iterationUsageStatus struct {
	InputTokens    int                     `json:"input_tokens"`
	OutputTokens   int                     `json:"output_tokens"`
	CostUSD        float64                 `json:"cost_usd"`
	CostProvenance executor.CostProvenance `json:"cost_provenance"`
}
type iterationStageStatus struct {
	StageID        string                `json:"stage_id"`
	Phase          string                `json:"phase"`
	LeaseUntil     int64                 `json:"lease_until"`
	SelectedRoute  *dispatch.Candidate   `json:"selected_route,omitempty"`
	Candidates     []dispatch.Candidate  `json:"pending_candidates,omitempty"`
	Usage          *iterationUsageStatus `json:"usage,omitempty"`
	Progress       *dispatch.Progress    `json:"progress,omitempty"`
	ProgressStatus string                `json:"progress_status"`
}

type iterationEvidenceStatus struct {
	StageID          string                       `json:"stage_id"`
	AcceptanceDigest string                       `json:"acceptance_digest"`
	OutputRevision   string                       `json:"output_revision"`
	Artifacts        []iteration.ArtifactEvidence `json:"artifacts"`
}

func writeIterationStatus(ctx context.Context, out io.Writer, store *coordinator.SQLiteStore, item *coordinator.MissionWorkItem, asJSON bool) error {
	status := iterationStatus{MissionID: item.MissionID, WorkItemID: item.WorkItemID, Phase: item.State, ReasonCode: item.ReasonCode, NextAction: item.NextAction, Version: item.Version, LeaseUntil: item.LeaseUntil, Deadline: item.Deadline, Evidence: []iterationEvidenceStatus{}, Routes: []dispatch.Candidate{}}
	var snapshot iteration.Snapshot
	if err := json.Unmarshal([]byte(item.SpecJSON), &snapshot); err != nil {
		return err
	}
	status.EvidenceDirectory = filepath.Join(snapshot.WorkspaceRoot, item.MissionID, item.WorkItemID, "evidence")
	status.ReceiptDirectory = filepath.Join(snapshot.WorkspaceRoot, item.MissionID, item.WorkItemID, "receipts")
	if item.NextStage >= 0 && item.NextStage < len(item.StageIDs) {
		status.Stage = item.StageIDs[item.NextStage]
	}
	var failedResult *executor.Result
	verificationFailed := false
	for _, stage := range item.StageIDs {
		key := coordinator.MissionAttemptKey{MissionID: item.MissionID, WorkItemID: item.WorkItemID, StageID: stage}
		a, err := store.GetMissionStageAcceptance(ctx, key)
		if err == nil {
			var accepted iteration.AcceptedStage
			if err = json.Unmarshal([]byte(a.AcceptanceJSON), &accepted); err != nil {
				return err
			}
			if accepted.Evidence == nil {
				return fmt.Errorf("stage %s acceptance lacks evidence", stage)
			}
			status.Evidence = append(status.Evidence, iterationEvidenceStatus{stage, a.AcceptanceDigest, accepted.Evidence.OutputRevision, accepted.Evidence.Artifacts})
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		child, err := store.GetMissionAttempt(ctx, key)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		if stage == status.Stage && child.State == "execution_completed" && item.State == "failed" {
			verificationFailed = true
		}
		stageStatus := iterationStageStatus{StageID: stage, Phase: child.State, LeaseUntil: child.LeaseUntil}
		if child.Outcome != "" {
			var report dispatch.Report
			if err = json.Unmarshal([]byte(child.Outcome), &report); err != nil {
				return err
			}
			stageStatus.Progress = report.Progress
			if stage == status.Stage && child.State == "execution_failed" {
				failedResult = report.Result
			}
			for _, attempt := range report.Attempts {
				if attempt.Status == "completed" || attempt.Status == "failed" || attempt.Status == "executing" {
					candidate := attempt.Candidate
					stageStatus.SelectedRoute = &candidate
					status.Routes = append(status.Routes, candidate)
				}
			}
			if report.Result != nil {
				r := report.Result
				stageStatus.Usage = &iterationUsageStatus{r.InputTokens, r.OutputTokens, r.CostUSD, r.CostProvenance}
			}
		} else {
			var snapshot iteration.Snapshot
			var request dispatch.Request
			if err = json.Unmarshal([]byte(item.SpecJSON), &snapshot); err != nil {
				return err
			}
			if err = json.Unmarshal([]byte(child.RequestJSON), &request); err != nil {
				return err
			}
			plan, err := dispatch.Resolve(request, snapshot.Models)
			if err != nil {
				return err
			}
			stageStatus.Candidates = plan.Candidates
		}
		if stageStatus.Progress != nil {
			stageStatus.ProgressStatus = "final report"
		} else {
			stageStatus.Progress, stageStatus.ProgressStatus = readIterationReceiptProgress(status.ReceiptDirectory, stage, child.RequestDigest)
		}
		status.Stages = append(status.Stages, stageStatus)
	}
	status.Diagnostic = diagnoseIteration(item, failedResult, verificationFailed)
	if status.NextAction == "" {
		status.NextAction = status.Diagnostic.NextAction
	}
	if asJSON {
		return json.NewEncoder(out).Encode(status)
	}
	_, err := fmt.Fprintf(out, "%s/%s: %s\nstage: %s  version: %d  lease: %d  deadline: %d\nreason: %s\ndiagnostic: %s\nnext: %s\naccepted stages: %d\n", status.MissionID, status.WorkItemID, status.Phase, status.Stage, status.Version, status.LeaseUntil, status.Deadline, status.ReasonCode, status.Diagnostic.Category, status.Diagnostic.NextAction, len(status.Evidence))
	return err
}

// iterationDiagnostic deliberately excludes provider text. The original reason_code
// is preserved separately; unknown causes are never guessed from stderr substrings.
type iterationDiagnostic struct {
	Category         string `json:"category"`
	NextAction       string `json:"next_action"`
	SuggestedCommand string `json:"suggested_command,omitempty"`
}

func diagnoseIteration(item *coordinator.MissionWorkItem, r *executor.Result, verificationFailed bool) iterationDiagnostic {
	statusCommand := fmt.Sprintf("ailang mission status %s --work-item %s --json", item.MissionID, item.WorkItemID)
	d := iterationDiagnostic{Category: "unknown", NextAction: "Inspect retained evidence; no automatic execution is valid until the cause is established.", SuggestedCommand: statusCommand}
	switch item.State {
	case "completed":
		return iterationDiagnostic{"completed", "Inspect accepted stage digests and artifact revisions in evidence.", statusCommand}
	case "needs_reconciliation":
		d.Category = "ambiguous_execution"
		d.NextAction = "Investigate exact child process and outcome evidence; generic ambiguity cannot be cleared by confirm-stopped."
		d.SuggestedCommand = ""
		if item.ReasonCode == "operator_cancelled" || item.ReasonCode == "deadline_exceeded" {
			d.NextAction = "Verify all child processes and descendants are stopped, then record explicit operator attestation at the current version."
			d.SuggestedCommand = fmt.Sprintf("ailang mission confirm-stopped %s --work-item %s --version %d --attestation <verified-stop-evidence>", item.MissionID, item.WorkItemID, item.Version)
		}
		return d
	case "running", "validating":
		return iterationDiagnostic{"in_progress", "Observe the current attempt; do not dispatch a duplicate. If its owner exited, resume to reconcile retained evidence.", statusCommand}
	case "ready":
		return iterationDiagnostic{"ready", "Resume the saved work item through its admitted runtime binding.", fmt.Sprintf("ailang mission resume %s --work-item %s", item.MissionID, item.WorkItemID)}
	case "waiting":
		switch item.ReasonCode {
		case "decision":
			return iterationDiagnostic{"decision", "Review the retained decision and supply approved successor authority; no automatic retry is valid.", statusCommand}
		case "availability", "quota", "quota_wait":
			return iterationDiagnostic{"admission", "Resolve the recorded admission condition, then resume the saved work item.", fmt.Sprintf("ailang mission resume %s --work-item %s", item.MissionID, item.WorkItemID)}
		}
	case "cancelled":
		return iterationDiagnostic{"cancelled", "Inspect retained artifacts and prepare an approved successor if more work is needed; this work item will not rerun.", statusCommand}
	}
	if item.ReasonCode == "deadline_exceeded" {
		d.Category = "deadline"
	}
	if verificationFailed {
		d.Category = "verification"
	}
	if r != nil {
		switch {
		case r.CostKilledAt > 0 || r.ThrashKilledAt > 0:
			d.Category = "budget"
		case r.FinishReason == executor.FinishCostExhausted || r.FinishReason == executor.FinishThrashAborted || r.FinishReason == executor.FinishLength || r.FinishReason == executor.FinishStepExhausted:
			d.Category = "budget"
		case r.FinishReason == executor.FinishTimeout:
			d.Category = "deadline"
		case r.FinishReason == executor.FinishError || r.FinishReason == executor.FinishContentFilter:
			d.Category = "provider_failure"
		}
	}
	if item.State == "failed" {
		d.NextAction = "Inspect retained stage evidence and prepare a reviewed successor; terminal work cannot resume. Use retry-review only when an accepted author artifact is present."
	}
	return d
}
