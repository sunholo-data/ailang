package eval_harness

import (
	"context"

	"github.com/sunholo-data/ailang/internal/executor"
)

// MaintenanceAttempt is the native ledger's assignment of observed leaf receipts.
// A nil LocalEdits means unmeasured, never zero repairs or zero bug fixes.
type MaintenanceAttempt struct {
	ID         string   `json:"id"`
	ReceiptIDs []string `json:"receiptIds"`
	LocalEdits *int     `json:"localEdits"`
}
type MaintenanceStageRun struct {
	ID             string               `json:"id"`
	State          string               `json:"state"`
	PriorDigest    *string              `json:"priorDigest"`
	ArtifactDigest *string              `json:"artifactDigest"`
	Visible        string               `json:"visible"`
	Oracle         string               `json:"oracle"`
	OracleDigest   *string              `json:"oracleDigest"`
	OracleArtifact *string              `json:"oracleArtifact"`
	Provenance     string               `json:"provenance"`
	Attempts       []MaintenanceAttempt `json:"attempts"`
	CheckKinds     []string             `json:"checkKinds"`
}
type MaintenanceTrajectoryRun struct {
	ID             string                `json:"id"`
	Model          string                `json:"model"`
	Harness        string                `json:"harness"`
	SettingsDigest string                `json:"settingsDigest"`
	Stages         []MaintenanceStageRun `json:"stages"`
}
type MaintenanceRuns struct {
	Schema       string                     `json:"schema"`
	Trajectories []MaintenanceTrajectoryRun `json:"trajectories"`
}

// MaintenanceGrade is controller evidence. It is never appended to the next
// model's directive or files. Digests must bind the actual candidate and oracle.
type MaintenanceGrade struct {
	Visible        string
	Oracle         string
	ArtifactDigest string
	OracleDigest   string
	CheckKinds     []string
	Provenance     string
}
type MaintenanceTaskBuilder func(MaintenanceTrajectoryPlan, MaintenanceStagePlan, string, string) (*executor.Task, error)
type MaintenanceGrader func(context.Context, MaintenanceTrajectoryPlan, MaintenanceStagePlan, MaintenanceCheckpoint) (MaintenanceGrade, error)

// MaintenanceDispatchStop is a pre-dispatch stop: it must not fabricate a call,
// cost, candidate or attempt. A budget hit after dispatch is banked separately.
type MaintenanceDispatchStop struct {
	Reason       string
	NeverStarted bool
}

func (e *MaintenanceDispatchStop) Error() string { return e.Reason }

type MaintenanceRunOptions struct {
	Execution *MaintenanceExecution
	Root      string
	Executor  executor.Executor
	BuildTask MaintenanceTaskBuilder
	Grade     MaintenanceGrader
	// BeforeDispatch is mandatory. The production adapter qualifies containment
	// here; scripted instrument controls pass an explicit no-model admission hook.
	BeforeDispatch func(context.Context, MaintenanceTrajectoryPlan, MaintenanceStagePlan) error
	CollectAttempt func(string, string, string, *executor.Result, error) (MaintenanceAttempt, error)
	Reap           func(string) ([]int, error)
}

// MaintenanceExecutionAttempt keeps raw executor evidence outside carried app
// files. Retain it even when parsing, grading, cancellation or capture fails.
type MaintenanceExecutionAttempt struct {
	ID             string                 `json:"id"`
	TrajectoryID   string                 `json:"trajectoryId"`
	StageID        string                 `json:"stageId"`
	Workspace      string                 `json:"workspace"`
	Checkpoint     *MaintenanceCheckpoint `json:"checkpoint"`
	Result         *executor.Result       `json:"result"`
	ExecutionError string                 `json:"executionError"`
	EvidenceError  string                 `json:"evidenceError"`
}
type MaintenanceRunReport struct {
	Runs     MaintenanceRuns               `json:"runs"`
	Attempts []MaintenanceExecutionAttempt `json:"attempts"`
}
