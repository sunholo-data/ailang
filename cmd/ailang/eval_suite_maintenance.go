package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// handleMaintenanceEvalMode runs immediately after flag parsing, before
// telemetry, registry/discovery, canary, rig admission or executor construction.
// Explicit suite flags are rejected even when their value equals a default.
func handleMaintenanceEvalMode(fs *flag.FlagSet, path string, agent, dryRun bool, out io.Writer) (bool, error) {
	if !flagWasSet(fs, "maintenance-manifest") {
		return false, nil
	}
	if path == "" || !agent {
		return true, fmt.Errorf("--maintenance-manifest requires a non-empty path and --agent")
	}
	allowed := map[string]bool{"maintenance-manifest": true, "agent": true, "dry-run": true, "output": true}
	var conflicts []string
	fs.Visit(func(f *flag.Flag) {
		if !allowed[f.Name] {
			conflicts = append(conflicts, "--"+f.Name)
		}
	})
	if len(conflicts) > 0 {
		return true, fmt.Errorf("--maintenance-manifest cannot combine with suite flags %v; allowed flags: --agent, --dry-run, --output", conflicts)
	}
	if fs.NArg() != 0 {
		return true, fmt.Errorf("maintenance mode does not accept positional arguments")
	}
	execution, err := eval_harness.LoadMaintenanceExecutionManifest(path)
	if err != nil {
		return true, err
	}
	if !dryRun {
		return true, fmt.Errorf("maintenance live admission is closed: require qualified containment and reviewed launch bounds; use --dry-run")
	}
	return true, printMaintenanceDryRun(out, execution)
}

type maintenanceDryStage struct {
	Trajectory        string `json:"trajectory"`
	Language          string `json:"language"`
	Stage             string `json:"stage"`
	StarterDigest     string `json:"starterDigest"`
	NativeToolDigest  string `json:"nativeToolDigest"`
	RequirementDigest string `json:"requirementDigest"`
	OracleDigest      string `json:"oracleDigest"`
}

func printMaintenanceDryRun(out io.Writer, e *eval_harness.MaintenanceExecution) error {
	schedule := make([]maintenanceDryStage, 0, len(e.Plan.Trajectories)*3)
	for _, trajectory := range e.Plan.Trajectories {
		for _, stage := range trajectory.Stages {
			schedule = append(schedule, maintenanceDryStage{trajectory.ID, trajectory.Language, stage.ID, trajectory.StarterDigest, trajectory.NativeToolDigest, stage.RequirementDigest, stage.OracleDigest})
		}
	}
	report := struct {
		Schema                string                            `json:"schema"`
		DryRun                bool                              `json:"dryRun"`
		Model                 string                            `json:"model"`
		Harness               string                            `json:"harness"`
		SettingsDigest        string                            `json:"settingsDigest"`
		ManifestDigest        string                            `json:"manifestDigest"`
		PlanDigest            string                            `json:"planDigest"`
		PackageSourceRevision string                            `json:"packageSourceRevision"`
		AnalysisDigest        string                            `json:"analysisDigest"`
		Limits                eval_harness.MaintenanceLimits    `json:"declaredLimits"`
		Isolation             eval_harness.MaintenanceIsolation `json:"declaredIsolation"`
		LiveAdmission         string                            `json:"liveAdmission"`
		UsageAvailability     string                            `json:"usageAvailability"`
		Schedule              []maintenanceDryStage             `json:"schedule"`
	}{"maintenance.dry-run/1", true, e.Plan.Model, e.Plan.Harness, e.Plan.SettingsDigest, e.ManifestSHA256, e.Manifest.Plan.SHA256, e.Manifest.PackageSourceRevision, e.Manifest.Analysis.SHA256, e.Manifest.Limits, e.Manifest.Isolation, "closed-until-measured-qualification-and-launch-review", "unavailable-before-execution", schedule}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
