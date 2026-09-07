package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sunholo-data/ailang/internal/executor"
	"io"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"github.com/sunholo-data/ailang/internal/mission/iteration"
)

type iterationStatus struct {
	MissionID         string                    `json:"mission_id"`
	WorkItemID        string                    `json:"work_item_id"`
	Phase             string                    `json:"phase"`
	ReasonCode        string                    `json:"reason_code"`
	NextAction        string                    `json:"next_action"`
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
	StageID       string                `json:"stage_id"`
	Phase         string                `json:"phase"`
	LeaseUntil    int64                 `json:"lease_until"`
	SelectedRoute *dispatch.Candidate   `json:"selected_route,omitempty"`
	Candidates    []dispatch.Candidate  `json:"pending_candidates,omitempty"`
	Usage         *iterationUsageStatus `json:"usage,omitempty"`
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
		stageStatus := iterationStageStatus{StageID: stage, Phase: child.State, LeaseUntil: child.LeaseUntil}
		if child.Outcome != "" {
			var report dispatch.Report
			if err = json.Unmarshal([]byte(child.Outcome), &report); err != nil {
				return err
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
		status.Stages = append(status.Stages, stageStatus)
	}
	if asJSON {
		return json.NewEncoder(out).Encode(status)
	}
	_, err := fmt.Fprintf(out, "%s/%s: %s\nstage: %s  version: %d  lease: %d  deadline: %d\nreason: %s\nnext: %s\naccepted stages: %d\n", status.MissionID, status.WorkItemID, status.Phase, status.Stage, status.Version, status.LeaseUntil, status.Deadline, status.ReasonCode, status.NextAction, len(status.Evidence))
	return err
}
