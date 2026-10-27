package eval_harness

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

type maintenanceScriptedExecutor struct {
	calls []*executor.Task
	run   func(context.Context, *executor.Task) (*executor.Result, error)
}

func (s *maintenanceScriptedExecutor) Name() string { return "scripted-control" }
func (s *maintenanceScriptedExecutor) Execute(ctx context.Context, task *executor.Task) (*executor.Result, error) {
	s.calls = append(s.calls, task)
	return s.run(ctx, task)
}
func (s *maintenanceScriptedExecutor) ExecuteStreaming(ctx context.Context, t *executor.Task, _ executor.EventHandler) (*executor.Result, error) {
	return s.Execute(ctx, t)
}
func (s *maintenanceScriptedExecutor) Capabilities() []executor.Capability {
	return []executor.Capability{executor.CapLocalWorkspace}
}
func (s *maintenanceScriptedExecutor) CostModel() *executor.CostModel    { return nil }
func (s *maintenanceScriptedExecutor) HealthCheck(context.Context) error { return nil }
func (s *maintenanceScriptedExecutor) Close() error                      { return nil }

func maintenanceRunnerFixture(t *testing.T, count int) (*MaintenanceExecution, string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	inputs := filepath.Join(root, "inputs")
	if err := os.Mkdir(inputs, 0700); err != nil {
		t.Fatal(err)
	}
	e := &MaintenanceExecution{Plan: MaintenanceRunPlan{Schema: "maintenance.trajectory/2", ProtocolID: "scripted-test", Role: "instrument_control", Model: "scripted-control", Harness: "scripted-control", SettingsDigest: strings.Repeat("a", 64)}}
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("trajectory-%d", i)
		starter := fmt.Sprintf("%s-starter.txt", id)
		maintenanceWrite(t, inputs, starter, "starter")
		identity := maintenanceFileIdentity("main.txt", []byte("starter"), 0600)
		digest, err := MaintenanceCheckpointDigest([]MaintenanceCheckpointFile{identity})
		if err != nil {
			t.Fatal(err)
		}
		trajectory := MaintenanceTrajectoryPlan{ID: id, Family: "text", Variant: "test", Repeat: i + 1, Language: "ailang", StarterDigest: digest}
		binding := MaintenanceTrajectoryBinding{ID: id, Starter: []MaintenanceAsset{{Path: filepath.Join(inputs, starter), Destination: "main.txt", SHA256: identity.SHA256}}, CarryFiles: []string{"main.txt", "notes.md"}}
		for stage := 1; stage <= 3; stage++ {
			sid := fmt.Sprintf("S%d", stage)
			request := fmt.Sprintf("%s-%s.md", id, sid)
			body := "current request " + sid
			maintenanceWrite(t, inputs, request, body)
			requestID := maintenanceFileIdentity(request, []byte(body), 0600)
			trajectory.Stages = append(trajectory.Stages, MaintenanceStagePlan{ID: sid, RequirementDigest: requestID.SHA256, OracleDigest: strings.Repeat("b", 64)})
			binding.Stages = append(binding.Stages, MaintenanceStageBinding{ID: sid, Request: MaintenanceFileRef{Path: filepath.Join(inputs, request), SHA256: requestID.SHA256}})
		}
		e.Plan.Trajectories = append(e.Plan.Trajectories, trajectory)
		e.Manifest.Trajectories = append(e.Manifest.Trajectories, binding)
	}
	return e, filepath.Join(root, "run")
}
func maintenanceTask(_ MaintenanceTrajectoryPlan, _ MaintenanceStagePlan, workspace, directive string) (*executor.Task, error) {
	return &executor.Task{Workspace: workspace, Directive: directive, Model: "scripted-control", Timeout: time.Second}, nil
}
func maintenanceAdmission(context.Context, MaintenanceTrajectoryPlan, MaintenanceStagePlan) error {
	return nil
}
func maintenanceGoodGrade(_ context.Context, _ MaintenanceTrajectoryPlan, stage MaintenanceStagePlan, cp MaintenanceCheckpoint) (MaintenanceGrade, error) {
	return MaintenanceGrade{Visible: "pass", Oracle: "pass", OracleDigest: stage.OracleDigest, ArtifactDigest: cp.Digest, CheckKinds: []string{"public", "oracle"}, Provenance: "scripted native grader"}, nil
}

func TestMaintenanceRunnerRetainsFailedSourceFreshSessionAndFixedSchedule(t *testing.T) {
	e, root := maintenanceRunnerFixture(t, 8)
	reaped := map[string]int{}
	s := &maintenanceScriptedExecutor{}
	s.run = func(_ context.Context, task *executor.Task) (*executor.Result, error) {
		if task.ResumeSessionID != "" || task.Iteration > 1 {
			t.Fatal("resumed old session")
		}
		body, err := os.ReadFile(filepath.Join(task.Workspace, "main.txt"))
		if err != nil {
			t.Fatal(err)
		}
		stage := (len(s.calls) - 1) % 3
		if stage == 0 {
			if string(body) != "starter" {
				t.Fatal("wrong starter")
			}
			maintenanceWrite(t, task.Workspace, "main.txt", "broken source from S1")
		} else if string(body) != "broken source from S1" {
			t.Fatalf("gold reset at stage %d: %q", stage, body)
		}
		if strings.Contains(task.Directive, "oracle") || strings.Contains(task.Directive, "S3") && stage < 2 {
			t.Fatal("oracle/future stage exposed")
		}
		maintenanceWrite(t, task.Workspace, "logs/provider.txt", "private log")
		return &executor.Result{Success: true, CostUSD: 0.125, SessionID: "old-provider-session", Transcript: "private old conversation"}, nil
	}
	grades := 0
	report, err := RunMaintenanceStages(context.Background(), MaintenanceRunOptions{DevelopmentInstrument: true, Execution: e, Root: root, Executor: s, BuildTask: maintenanceTask, BeforeDispatch: maintenanceAdmission, Reap: func(p string) ([]int, error) { reaped[p]++; return nil, nil }, Grade: func(ctx context.Context, tr MaintenanceTrajectoryPlan, stage MaintenanceStagePlan, cp MaintenanceCheckpoint) (MaintenanceGrade, error) {
		grades++
		if reaped[filepath.Join(root, fmt.Sprintf("trajectory-%03d", (grades-1)/3), "workspaces", fmt.Sprintf("stage-%03d", (grades-1)%3))] == 0 {
			t.Fatal("grade before reap")
		}
		g, _ := maintenanceGoodGrade(ctx, tr, stage, cp)
		if stage.ID == "S1" {
			g.Visible = "fail"
			g.Oracle = "fail"
		}
		return g, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.calls) != 24 || len(report.Attempts) != 24 || len(report.Runs.Trajectories) != 8 {
		t.Fatalf("lost schedule/attempts: %d/%d/%d", len(s.calls), len(report.Attempts), len(report.Runs.Trajectories))
	}
	for _, tr := range report.Runs.Trajectories {
		if len(tr.Stages) != 3 || tr.Stages[0].State != "failed" || tr.Stages[1].State != "completed" {
			t.Fatalf("stage outcomes %+v", tr.Stages)
		}
		if *tr.Stages[1].PriorDigest != *tr.Stages[0].ArtifactDigest {
			t.Fatal("broken candidate not retained")
		}
	}
	raw, err := json.Marshal(report.Runs)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "Transcript") || strings.Contains(string(raw), "private old conversation") {
		t.Fatal("transcript in native ledger")
	}
	if report.Attempts[0].Result.CostUSD != 0.125 {
		t.Fatal("failed spend dropped")
	}
}

func TestMaintenanceRunnerAccountsBudgetCancellationAndSetup(t *testing.T) {
	for _, kind := range []string{"budget", "cancel", "setup", "timeout", "grade_error", "capture_error"} {
		t.Run(kind, func(t *testing.T) {
			e, root := maintenanceRunnerFixture(t, 1)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			reaps := 0
			s := &maintenanceScriptedExecutor{run: func(_ context.Context, task *executor.Task) (*executor.Result, error) {
				switch kind {
				case "budget":
					return &executor.Result{CostUSD: 0.45, FinishReason: executor.FinishCostExhausted}, nil
				case "cancel":
					cancel()
					return &executor.Result{CostUSD: 0.2}, context.Canceled
				case "timeout":
					return &executor.Result{CostUSD: 0.2, FinishReason: executor.FinishTimeout}, context.DeadlineExceeded
				case "capture_error":
					if err := os.Mkdir(filepath.Join(task.Workspace, "notes.md"), 0700); err != nil {
						t.Fatal(err)
					}
				}
				return &executor.Result{Success: true}, nil
			}}
			builder := maintenanceTask
			if kind == "setup" {
				builder = func(MaintenanceTrajectoryPlan, MaintenanceStagePlan, string, string) (*executor.Task, error) {
					return nil, errors.New("missing trusted tool")
				}
			}
			grade := maintenanceGoodGrade
			if kind == "grade_error" {
				grade = func(context.Context, MaintenanceTrajectoryPlan, MaintenanceStagePlan, MaintenanceCheckpoint) (MaintenanceGrade, error) {
					return MaintenanceGrade{}, errors.New("oracle infrastructure failed")
				}
			}
			report, err := RunMaintenanceStages(ctx, MaintenanceRunOptions{DevelopmentInstrument: true, Execution: e, Root: root, Executor: s, BuildTask: builder, Grade: grade, BeforeDispatch: maintenanceAdmission, Reap: func(string) ([]int, error) { reaps++; return nil, nil }})
			if err != nil {
				t.Fatal(err)
			}
			stages := report.Runs.Trajectories[0].Stages
			if len(stages) != 3 {
				t.Fatal("lost later stage rows")
			}
			switch kind {
			case "budget":
				if stages[0].State != "budget_stopped" || stages[1].State != "blocked" || report.Attempts[0].Result.CostUSD != 0.45 {
					t.Fatalf("budget: %+v", stages)
				}
			case "cancel":
				if stages[0].State != "invalid" || stages[1].State != "not_started" {
					t.Fatalf("cancel: %+v", stages)
				}
			case "setup":
				if stages[0].State != "blocked" || len(report.Attempts) != 0 {
					t.Fatalf("setup fabricated execution: %+v", stages)
				}
			default:
				if stages[0].State != "invalid" {
					t.Fatalf("%s: %+v", kind, stages)
				}
			}
			if reaps == 0 {
				t.Fatal("workspace not reaped on failure")
			}
		})
	}
}

func TestMaintenanceRunnerRejectsUnqualifiedDispatchAndResume(t *testing.T) {
	e, root := maintenanceRunnerFixture(t, 1)
	s := &maintenanceScriptedExecutor{run: func(context.Context, *executor.Task) (*executor.Result, error) {
		t.Fatal("should not dispatch")
		return nil, nil
	}}
	if _, err := RunMaintenanceStages(context.Background(), MaintenanceRunOptions{DevelopmentInstrument: true, Execution: e, Root: root, Executor: s, BuildTask: maintenanceTask, Grade: maintenanceGoodGrade}); err == nil {
		t.Fatal("missing admission guard accepted")
	}
	e, root = maintenanceRunnerFixture(t, 1)
	report, err := RunMaintenanceStages(context.Background(), MaintenanceRunOptions{DevelopmentInstrument: true, Execution: e, Root: root, Executor: s, BuildTask: func(tr MaintenanceTrajectoryPlan, stage MaintenanceStagePlan, ws, prompt string) (*executor.Task, error) {
		task, _ := maintenanceTask(tr, stage, ws, prompt)
		task.ResumeSessionID = "old"
		return task, nil
	}, Grade: maintenanceGoodGrade, BeforeDispatch: maintenanceAdmission, Reap: func(string) ([]int, error) { return nil, nil }})
	if err != nil {
		t.Fatal(err)
	}
	if report.Runs.Trajectories[0].Stages[0].State != "blocked" {
		t.Fatal("resume admitted")
	}
}

func TestMaintenanceRunnerRejectsAlteredPrivateEvidenceAndUncompiledIsFailure(t *testing.T) {
	for _, kind := range []string{"oracle_digest", "candidate_digest", "uncompiled", "unmeasured_oracle", "reap_failure", "receipt_error", "missing_request", "pre_budget", "final_reap_failure"} {
		t.Run(kind, func(t *testing.T) {
			e, root := maintenanceRunnerFixture(t, 1)
			if kind == "missing_request" {
				if err := os.Remove(e.Manifest.Trajectories[0].Stages[0].Request.Path); err != nil {
					t.Fatal(err)
				}
			}
			s := &maintenanceScriptedExecutor{run: func(context.Context, *executor.Task) (*executor.Result, error) {
				return &executor.Result{Success: true, CostUSD: 0.3}, nil
			}}
			grade := func(ctx context.Context, tr MaintenanceTrajectoryPlan, stage MaintenanceStagePlan, cp MaintenanceCheckpoint) (MaintenanceGrade, error) {
				g, _ := maintenanceGoodGrade(ctx, tr, stage, cp)
				switch kind {
				case "oracle_digest":
					g.OracleDigest = strings.Repeat("c", 64)
				case "candidate_digest":
					g.ArtifactDigest = strings.Repeat("c", 64)
				case "uncompiled":
					g.Visible = "fail"
					g.Oracle = "not_run"
					g.OracleDigest = ""
					g.CheckKinds = []string{"parser"}
				case "unmeasured_oracle":
					g.Visible = "pass"
					g.Oracle = "unknown"
					g.OracleDigest = ""
					g.CheckKinds = []string{"public"}
				}
				return g, nil
			}
			admission := maintenanceAdmission
			if kind == "pre_budget" {
				admission = func(context.Context, MaintenanceTrajectoryPlan, MaintenanceStagePlan) error {
					return &MaintenanceDispatchStop{Reason: "budget exhausted before dispatch", NeverStarted: true}
				}
			}
			reapCalls := 0
			reaper := func(string) ([]int, error) {
				reapCalls++
				if kind == "reap_failure" || kind == "final_reap_failure" && reapCalls == 2 {
					return nil, errors.New("process boundary unavailable")
				}
				return nil, nil
			}
			collector := func(_, _, id string, _ *executor.Result, _ error) (MaintenanceAttempt, error) {
				if kind == "receipt_error" {
					return MaintenanceAttempt{}, errors.New("provider did not report leaf coverage")
				}
				return MaintenanceAttempt{ID: id, ReceiptIDs: []string{}}, nil
			}
			report, err := RunMaintenanceStages(context.Background(), MaintenanceRunOptions{DevelopmentInstrument: true, Execution: e, Root: root, Executor: s, BuildTask: maintenanceTask, Grade: grade, BeforeDispatch: admission, Reap: reaper, CollectAttempt: collector})
			if err != nil {
				t.Fatal(err)
			}
			stage := report.Runs.Trajectories[0].Stages[0]
			switch kind {
			case "uncompiled":
				if stage.State != "failed" || stage.Oracle != "not_run" || stage.OracleArtifact != nil {
					t.Fatalf("uncompiled %+v", stage)
				}
			case "receipt_error":
				if stage.State != "completed" || !strings.Contains(report.Attempts[0].EvidenceError, "coverage") {
					t.Fatalf("receipt unavailability lost: %+v", report)
				}
			case "missing_request":
				if stage.State != "blocked" || len(report.Attempts) != 0 {
					t.Fatalf("missing requirement fabricated call: %+v", stage)
				}
			case "pre_budget":
				if stage.State != "not_started" || stage.PriorDigest != nil || len(report.Attempts) != 0 {
					t.Fatalf("pre-budget fabricated call: %+v", stage)
				}
			default:
				if stage.State != "invalid" || stage.OracleDigest != nil {
					t.Fatalf("invalid evidence accepted: %+v", stage)
				}
			}
		})
	}
}

func TestMaintenanceRunnerRejectsObservedMeasurementDrift(t *testing.T) {
	for _, kind := range []string{"false_success_empty_finish", "false_success_unknown_finish", "task_model", "harness_version", "missing_version"} {
		t.Run(kind, func(t *testing.T) {
			e, root := maintenanceRunnerFixture(t, 1)
			e.Plan.Role = "model_run"
			gradeCalls := 0
			s := &maintenanceScriptedExecutor{run: func(_ context.Context, task *executor.Task) (*executor.Result, error) {
				maintenanceWrite(t, task.Workspace, "main.txt", "actual candidate after charged call")
				result := &executor.Result{Success: true, CostUSD: 0.75, ExecutorVersion: e.Plan.Harness}
				switch kind {
				case "false_success_empty_finish":
					result.Success = false
				case "false_success_unknown_finish":
					result.Success = false
					result.FinishReason = "new_unrecognized_failure"
				case "harness_version":
					result.ExecutorVersion = "scripted-control@different"
				case "missing_version":
					result.ExecutorVersion = ""
				}
				return result, nil
			}}
			builder := func(tr MaintenanceTrajectoryPlan, stage MaintenanceStagePlan, ws, prompt string) (*executor.Task, error) {
				task, _ := maintenanceTask(tr, stage, ws, prompt)
				if kind == "task_model" {
					task.Model = "different-model"
				}
				return task, nil
			}
			report, err := RunMaintenanceStages(context.Background(), MaintenanceRunOptions{Execution: e, Root: root, Executor: s, BuildTask: builder, Grade: func(ctx context.Context, tr MaintenanceTrajectoryPlan, stage MaintenanceStagePlan, cp MaintenanceCheckpoint) (MaintenanceGrade, error) {
				gradeCalls++
				return maintenanceGoodGrade(ctx, tr, stage, cp)
			}, BeforeDispatch: maintenanceAdmission, Reap: func(string) ([]int, error) { return nil, nil }})
			if err != nil {
				t.Fatal(err)
			}
			stage := report.Runs.Trajectories[0].Stages[0]
			if gradeCalls != 0 {
				t.Fatalf("%s reached correctness grading despite invalid measurement", kind)
			}
			if kind == "task_model" {
				if stage.State != "blocked" || len(s.calls) != 0 || len(report.Attempts) != 0 {
					t.Fatalf("wrong model dispatched: %+v", stage)
				}
			} else {
				if stage.State != "invalid" || stage.ArtifactDigest == nil || len(stage.Attempts) != 1 {
					t.Fatalf("invalid measurement accepted: %+v", stage)
				}
				if report.Attempts[0].Result.CostUSD != 0.75 || report.Attempts[0].Checkpoint == nil {
					t.Fatal("actual charged source/spend lost")
				}
			}
		})
	}
}

type maintenanceNamedExecutor struct {
	*maintenanceScriptedExecutor
	name string
}

func (s *maintenanceNamedExecutor) Name() string { return s.name }

func TestMaintenanceRunnerInstrumentExceptionIsExplicitAndNarrow(t *testing.T) {
	for _, kind := range []string{"explicit_control", "implicit_control", "wrong_role", "wrong_model", "wrong_executor", "reported_mismatch", "budget_version_drift"} {
		t.Run(kind, func(t *testing.T) {
			e, root := maintenanceRunnerFixture(t, 1)
			if kind == "wrong_role" {
				e.Plan.Role = "model_run"
			}
			if kind == "wrong_model" {
				e.Plan.Model = "paid-real-model"
			}
			name := "scripted-control"
			if kind == "wrong_executor" {
				name = "pi"
			}
			s := &maintenanceNamedExecutor{name: name, maintenanceScriptedExecutor: &maintenanceScriptedExecutor{run: func(context.Context, *executor.Task) (*executor.Result, error) {
				result := &executor.Result{Success: true, CostUSD: 0.55}
				if kind == "reported_mismatch" || kind == "budget_version_drift" {
					result.ExecutorVersion = "changed-harness"
				}
				if kind == "budget_version_drift" {
					result.Success = false
					result.FinishReason = executor.FinishCostExhausted
					result.CostKilledAt = 0.5
				}
				return result, nil
			}}}
			report, err := RunMaintenanceStages(context.Background(), MaintenanceRunOptions{DevelopmentInstrument: kind != "implicit_control", Execution: e, Root: root, Executor: s, BuildTask: maintenanceTask, Grade: maintenanceGoodGrade, BeforeDispatch: maintenanceAdmission, Reap: func(string) ([]int, error) { return nil, nil }})
			switch kind {
			case "wrong_role", "wrong_model", "wrong_executor":
				if err == nil || len(s.calls) != 0 {
					t.Fatalf("invalid instrument bypass accepted: %v", err)
				}
			default:
				if err != nil {
					t.Fatal(err)
				}
				stage := report.Runs.Trajectories[0].Stages[0]
				want := "invalid"
				if kind == "explicit_control" {
					want = "completed"
				}
				if stage.State != want {
					t.Fatalf("%s state=%s want=%s", kind, stage.State, want)
				}
				if kind == "budget_version_drift" && report.Attempts[0].Result.CostUSD != 0.55 {
					t.Fatal("budget identity drift lost actual spend")
				}
			}
		})
	}
}
