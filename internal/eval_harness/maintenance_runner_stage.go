package eval_harness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

func maintenanceRunStage(ctx context.Context, o MaintenanceRunOptions, report *MaintenanceRunReport, tr MaintenanceTrajectoryPlan, stage MaintenanceStagePlan, binding MaintenanceStageBinding, carry []string, root string, index int, prior MaintenanceCheckpoint) (MaintenanceStageRun, *MaintenanceCheckpoint, string) {
	row := maintenanceUnstarted(stage.ID, "blocked", &prior.Digest, "stage setup not completed")
	workspace := filepath.Join(root, "workspaces", fmt.Sprintf("stage-%03d", index))
	if err := RestoreMaintenanceCheckpoint(prior, workspace); err != nil {
		row.Provenance = "restore failed: " + err.Error()
		return row, nil, "checkpoint restoration failed"
	}
	// The deferred reap covers setup failures too. The explicit reap below runs
	// before capture/grading; the second reap catches anything grading left behind.
	defer func() { _, _ = o.Reap(workspace) }()
	request, err := maintenanceReadBoundFile(binding.Request)
	if err != nil {
		row.Provenance = "requirement changed: " + err.Error()
		return row, nil, "requirement identity failed"
	}
	if binding.ID != stage.ID || binding.Request.SHA256 != stage.RequirementDigest {
		row.Provenance = "stage requirement binding drift"
		return row, nil, "requirement identity failed"
	}
	task, err := o.BuildTask(tr, stage, workspace, string(request))
	if err != nil {
		row.Provenance = "task setup failed: " + err.Error()
		return row, nil, "task setup failed"
	}
	if task == nil || task.ResumeSessionID != "" || task.Iteration > 1 || filepath.Clean(task.Workspace) != filepath.Clean(workspace) || task.Directive != string(request) {
		row.Provenance = "task violates fresh-session/current-request/workspace contract"
		return row, nil, "task identity failed"
	}
	attemptID := fmt.Sprintf("%s/%s/%s/attempt-1", o.Execution.Plan.ProtocolID, tr.ID, stage.ID)
	task.ID = attemptID
	task.Iteration = 1
	callCtx := ctx
	cancel := func() {}
	if task.Timeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, task.Timeout)
	}
	result, executeErr := o.Executor.Execute(callCtx, task)
	callContextErr := callCtx.Err()
	cancel()
	observed := MaintenanceExecutionAttempt{ID: attemptID, TrajectoryID: tr.ID, StageID: stage.ID, Workspace: workspace, Result: result}
	if executeErr != nil {
		observed.ExecutionError = executeErr.Error()
	}
	if callContextErr != nil && observed.ExecutionError == "" {
		observed.ExecutionError = callContextErr.Error()
	}
	row.State = "invalid"
	row.Visible = "unknown"
	row.Oracle = "not_run"
	row.CheckKinds = []string{"infrastructure"}
	attempt := MaintenanceAttempt{ID: attemptID, ReceiptIDs: []string{}}
	if o.CollectAttempt != nil {
		collected, collectErr := o.CollectAttempt(tr.ID, stage.ID, attemptID, result, executeErr)
		if collectErr != nil {
			observed.EvidenceError = "usage evidence unavailable: " + collectErr.Error()
		} else if collected.ID != attemptID {
			observed.EvidenceError = "usage attempt identity mismatch"
		} else {
			attempt = collected
			if attempt.ReceiptIDs == nil {
				attempt.ReceiptIDs = []string{}
			}
		}
	}
	row.Attempts = []MaintenanceAttempt{attempt}
	if _, err := o.Reap(workspace); err != nil {
		row.Provenance = "workspace reap failed: " + err.Error()
		observed.EvidenceError = maintenanceJoinError(observed.EvidenceError, row.Provenance)
		report.Attempts = append(report.Attempts, observed)
		return row, nil, "workspace reap failed; checkpoint not finalized"
	}
	cp, captureErr := CaptureMaintenanceCheckpoint(workspace, filepath.Join(root, "checkpoints", fmt.Sprintf("stage-%03d", index)), carry)
	if captureErr != nil {
		row.Provenance = "candidate checkpoint unavailable: " + captureErr.Error()
		observed.EvidenceError = maintenanceJoinError(observed.EvidenceError, row.Provenance)
		report.Attempts = append(report.Attempts, observed)
		return row, nil, "candidate checkpoint failed; retain last actual checkpoint"
	}
	row.ArtifactDigest = &cp.Digest
	observed.Checkpoint = &cp
	stop := ""
	switch {
	case result != nil && (result.CostKilledAt > 0 || result.ThrashKilledAt > 0 || result.FinishReason == executor.FinishCostExhausted || result.FinishReason == executor.FinishThrashAborted || result.FinishReason == executor.FinishStepExhausted):
		row.State = "budget_stopped"
		row.Provenance = "in-flight budget stopped; spent attempt retained"
		stop = "trajectory budget stopped"
	case executeErr != nil || callContextErr != nil || ctx.Err() != nil || result == nil:
		row.Provenance = "executor did not complete a valid measured stage: " + observed.ExecutionError
	case result.FinishReason == executor.FinishTimeout || result.FinishReason == executor.FinishIdleKilled || result.FinishReason == executor.FinishError || result.FinishReason == executor.FinishWireDrift || result.FinishReason == executor.FinishContentFilter:
		row.Provenance = "executor terminal signal prevents a valid measured stage: " + result.FinishReason
		observed.ExecutionError = maintenanceJoinError(observed.ExecutionError, row.Provenance)
	default:
		maintenanceApplyGrade(ctx, o, tr, stage, cp, &row)
	}
	report.Attempts = append(report.Attempts, observed)
	return row, &cp, stop
}

func maintenanceApplyGrade(ctx context.Context, o MaintenanceRunOptions, tr MaintenanceTrajectoryPlan, stage MaintenanceStagePlan, cp MaintenanceCheckpoint, row *MaintenanceStageRun) {
	grade, err := o.Grade(ctx, tr, stage, cp)
	if err != nil {
		row.Provenance = "grading infrastructure failed: " + err.Error()
		return
	}
	checked := grade.Oracle == "pass" || grade.Oracle == "fail"
	if grade.ArtifactDigest != cp.Digest || grade.Visible != "pass" && grade.Visible != "fail" || grade.Provenance == "" {
		row.Provenance = "grading evidence identity or verdict is invalid"
		return
	}
	for _, kind := range grade.CheckKinds {
		if !maintenanceHasKind([]string{"parser", "type", "effect", "contract", "public", "oracle", "infrastructure"}, kind) {
			row.Provenance = "grading check kind is invalid"
			return
		}
	}
	if checked {
		if grade.OracleDigest != stage.OracleDigest || !maintenanceHasKind(grade.CheckKinds, "oracle") {
			row.Provenance = "oracle evidence identity is invalid"
			return
		}
		row.OracleDigest = &grade.OracleDigest
		row.OracleArtifact = &cp.Digest
	} else if grade.Visible != "fail" || grade.Oracle != "not_run" || grade.OracleDigest != "" || maintenanceHasKind(grade.CheckKinds, "oracle") || len(grade.CheckKinds) == 0 {
		row.Provenance = "unrun oracle cannot claim checked evidence"
		return
	}
	row.Visible = grade.Visible
	row.Oracle = grade.Oracle
	row.CheckKinds = append([]string{}, grade.CheckKinds...)
	row.Provenance = grade.Provenance
	row.State = "completed"
	if grade.Visible == "fail" || grade.Oracle == "fail" {
		row.State = "failed"
	}
}

func maintenanceReadBoundFile(ref MaintenanceFileRef) ([]byte, error) {
	if err := maintenanceRegularPath(ref.Path, false); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(ref.Path)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != ref.SHA256 {
		return nil, fmt.Errorf("bound source digest mismatch: %s", ref.Path)
	}
	return data, nil
}
func maintenanceHasKind(kinds []string, expected string) bool {
	for _, kind := range kinds {
		if kind == expected {
			return true
		}
	}
	return false
}
func maintenanceJoinError(a, b string) string {
	if a == "" {
		return b
	}
	return strings.Join([]string{a, b}, "; ")
}
