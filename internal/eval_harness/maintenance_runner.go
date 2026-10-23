package eval_harness

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/proctree"
)

// RunMaintenanceStages runs only the frozen three-stage maintenance schedule.
// It neither discovers benchmarks nor resets a failed candidate to a gold file.
// Private grading results are written to controller evidence, never model input.
func RunMaintenanceStages(ctx context.Context, options MaintenanceRunOptions) (*MaintenanceRunReport, error) {
	if options.Execution == nil || options.Executor == nil || options.BuildTask == nil || options.Grade == nil || options.BeforeDispatch == nil {
		return nil, fmt.Errorf("maintenance execution, executor, builder, grader and admission guard required")
	}
	e := options.Execution
	if len(e.Plan.Trajectories) == 0 || len(e.Plan.Trajectories) != len(e.Manifest.Trajectories) {
		return nil, fmt.Errorf("maintenance schedule/binding mismatch")
	}
	for i, tr := range e.Plan.Trajectories {
		if len(tr.Stages) != 3 || len(e.Manifest.Trajectories[i].Stages) != 3 || tr.ID != e.Manifest.Trajectories[i].ID {
			return nil, fmt.Errorf("maintenance requires matched three-stage bindings")
		}
	}
	if options.Reap == nil {
		options.Reap = proctree.ReapWorkspace
	}
	if err := os.Mkdir(options.Root, 0700); err != nil {
		return nil, fmt.Errorf("reserve controller run root: %w", err)
	}
	report := &MaintenanceRunReport{Runs: MaintenanceRuns{Schema: "maintenance.runs/2", Trajectories: []MaintenanceTrajectoryRun{}}, Attempts: []MaintenanceExecutionAttempt{}}
	for i, tr := range e.Plan.Trajectories {
		row := MaintenanceTrajectoryRun{ID: tr.ID, Model: e.Plan.Model, Harness: e.Plan.Harness, SettingsDigest: e.Plan.SettingsDigest, Stages: []MaintenanceStageRun{}}
		for _, stage := range tr.Stages {
			row.Stages = append(row.Stages, maintenanceUnstarted(stage.ID, "not_started", nil, "not dispatched"))
		}
		report.Runs.Trajectories = append(report.Runs.Trajectories, row)
		maintenanceRunTrajectory(ctx, options, report, i)
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return report, err
	}
	if err := maintenanceCreateFile(filepath.Join(options.Root, "execution-report.json"), data, 0600); err != nil {
		return report, err
	}
	data, err = json.MarshalIndent(report.Runs, "", "  ")
	if err != nil {
		return report, err
	}
	if err := maintenanceCreateFile(filepath.Join(options.Root, "runs.json"), data, 0600); err != nil {
		return report, err
	}
	return report, nil
}

func maintenanceRunTrajectory(ctx context.Context, o MaintenanceRunOptions, report *MaintenanceRunReport, index int) {
	tr := o.Execution.Plan.Trajectories[index]
	binding := o.Execution.Manifest.Trajectories[index]
	rows := report.Runs.Trajectories[index].Stages
	root := filepath.Join(o.Root, fmt.Sprintf("trajectory-%03d", index))
	previous := tr.StarterDigest
	cp, err := maintenanceStarterCheckpoint(root, binding)
	blockReason := ""
	if err != nil {
		blockReason = "invalid starter setup: " + err.Error()
	} else if cp.Digest != previous {
		blockReason = "starter checkpoint identity mismatch"
	}
	for i, stage := range tr.Stages {
		if ctx.Err() != nil {
			rows[i] = maintenanceUnstarted(stage.ID, "not_started", nil, "campaign cancelled before dispatch: "+ctx.Err().Error())
			continue
		}
		if blockReason != "" {
			rows[i] = maintenanceUnstarted(stage.ID, "blocked", &previous, blockReason)
			continue
		}
		if err := o.BeforeDispatch(ctx, tr, stage); err != nil {
			var stop *MaintenanceDispatchStop
			if errors.As(err, &stop) && stop.NeverStarted {
				rows[i] = maintenanceUnstarted(stage.ID, "not_started", nil, err.Error())
			} else {
				rows[i] = maintenanceUnstarted(stage.ID, "blocked", &previous, "dispatch refused: "+err.Error())
			}
			blockReason = "earlier dispatch was refused"
			continue
		}
		row, next, stop := maintenanceRunStage(ctx, o, report, tr, stage, binding.Stages[i], binding.CarryFiles, root, i, cp)
		rows[i] = row
		if next != nil {
			cp = *next
			previous = cp.Digest
		}
		if stop != "" {
			blockReason = stop
		}
	}
}

func maintenanceStarterCheckpoint(root string, binding MaintenanceTrajectoryBinding) (MaintenanceCheckpoint, error) {
	cp := MaintenanceCheckpoint{}
	for _, dir := range []string{root, filepath.Join(root, "workspaces"), filepath.Join(root, "checkpoints"), filepath.Join(root, "starter")} {
		if err := os.Mkdir(dir, 0700); err != nil {
			return cp, err
		}
	}
	starter := filepath.Join(root, "starter")
	for _, asset := range binding.Starter {
		if err := maintenanceOwnedPath(asset.Destination); err != nil {
			return cp, err
		}
		data, err := maintenanceReadBoundFile(MaintenanceFileRef{Path: asset.Path, SHA256: asset.SHA256})
		if err != nil {
			return cp, err
		}
		f := MaintenanceCheckpointFile{Path: asset.Destination, SHA256: asset.SHA256, Executable: asset.Executable}
		if err := maintenanceWriteNew(starter, f, data); err != nil {
			return cp, err
		}
	}
	return CaptureMaintenanceCheckpoint(starter, filepath.Join(root, "checkpoints", "starter"), binding.CarryFiles)
}

func maintenanceUnstarted(id, state string, prior *string, reason string) MaintenanceStageRun {
	return MaintenanceStageRun{ID: id, State: state, PriorDigest: prior, Visible: "not_run", Oracle: "not_run", Provenance: reason, Attempts: []MaintenanceAttempt{}, CheckKinds: []string{}}
}
