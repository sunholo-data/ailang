package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"github.com/sunholo-data/ailang/internal/eval_harness"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func maintenanceTestFlags(args ...string) *flag.FlagSet {
	fs := flag.NewFlagSet("maintenance-test", flag.ContinueOnError)
	fs.Bool("agent", false, "")
	fs.Bool("dry-run", false, "")
	fs.String("maintenance-manifest", "", "")
	fs.String("output", "", "")
	for _, name := range []string{"benchmarks", "tier", "models", "model", "langs", "conditions", "baseline", "browser-provider", "benchmark-dir", "tool-policy", "max-wall-clock", "budget-usd"} {
		fs.String(name, "", "")
	}
	fs.Bool("queue", false, "")
	fs.Int("trials", 1, "")
	_ = fs.Parse(args)
	return fs
}
func TestMaintenanceModeRejectsExplicitSuiteOverrides(t *testing.T) {
	for _, args := range [][]string{
		{"--benchmarks=fizzbuzz"}, {"--tier=smoke"}, {"--models=anything"}, {"--model=anything"}, {"--langs=python,ailang"}, {"--conditions=baseline"}, {"--baseline=v1.0"}, {"--queue=false"}, {"--trials=1"}, {"--browser-provider=local-playwright"}, {"--benchmark-dir=missing"}, {"--tool-policy=full"}, {"--max-wall-clock=5s"}, {"--budget-usd=1"},
	} {
		t.Run(args[0], func(t *testing.T) {
			args = append([]string{"--agent", "--dry-run", "--maintenance-manifest=DO-NOT-LOAD"}, args...)
			fs := maintenanceTestFlags(args...)
			var out bytes.Buffer
			handled, err := handleMaintenanceEvalMode(fs, "DO-NOT-LOAD", true, true, &out)
			if !handled || err == nil || !strings.Contains(err.Error(), "cannot combine") {
				t.Fatalf("want conflict before file read, handled=%v error=%v", handled, err)
			}
			if out.Len() != 0 {
				t.Fatal("unexpected output")
			}
		})
	}
}
func TestMaintenanceModeRequiresAgentAndExplicitPath(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		path  string
		agent bool
	}{
		{[]string{"--maintenance-manifest=missing", "--dry-run"}, "missing", false},
		{[]string{"--agent", "--maintenance-manifest=", "--dry-run"}, "", true},
	} {
		var out bytes.Buffer
		handled, err := handleMaintenanceEvalMode(maintenanceTestFlags(tc.args...), tc.path, tc.agent, true, &out)
		if !handled || err == nil {
			t.Fatalf("accepted invalid flags: %v", tc.args)
		}
	}
}
func TestMaintenanceModeLeavesOrdinarySuiteUntouched(t *testing.T) {
	var out bytes.Buffer
	handled, err := handleMaintenanceEvalMode(maintenanceTestFlags("--models=ordinary"), "", false, false, &out)
	if handled || err != nil || out.Len() != 0 {
		t.Fatalf("ordinary suite changed: %v %v %s", handled, err, out.String())
	}
}

func maintenanceCLIManifestFixture(t *testing.T) (string, eval_harness.MaintenanceExecutionManifest) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file := func(name, content string) eval_harness.MaintenanceFileRef {
		p := filepath.Join(root, name)
		if err := os.WriteFile(p, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256([]byte(content))
		return eval_harness.MaintenanceFileRef{Path: name, SHA256: hex.EncodeToString(h[:])}
	}
	settings := file("settings.json", `{"temperature":0}`)
	analysis := file("analysis.ail", "native analysis fixture")
	tool := file("tool.json", `{"schema":"maintenance.toolchain/1","command":["fixture"],"versionCommand":["fixture","--version"],"version":"fixture-only","binarySHA256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)
	starter := file("main.py", "broken initial source")
	request := file("request.md", "repair real retained source")
	check := file("check.py", "trusted public check")
	oracle := file("oracle.py", "trusted external oracle")
	pipelineBytes, err := json.Marshal([]string{oracle.SHA256})
	if err != nil {
		t.Fatal(err)
	}
	pipelineSum := sha256.Sum256(pipelineBytes)
	oraclePipeline := hex.EncodeToString(pipelineSum[:])
	assets := []eval_harness.MaintenanceAsset{{Path: starter.Path, Destination: "main.py", SHA256: starter.SHA256}}
	starterDigest, err := maintenanceCLIStarterDigest(assets)
	if err != nil {
		t.Fatal(err)
	}
	plan := eval_harness.MaintenanceRunPlan{Schema: "maintenance.trajectory/2", ProtocolID: "fixture", Role: "instrument_control", Model: "scripted-control", Harness: "scripted", SettingsDigest: settings.SHA256, Retention: "fresh-session-owned-artifacts", Treatment: "native-tools", LeftLanguage: "ailang", RightLanguage: "python"}
	m := eval_harness.MaintenanceExecutionManifest{Schema: "maintenance.execution/1", PackageSourceRevision: eval_harness.MaintenancePackageSourceRevision, PackageRoot: root, Settings: settings, Analysis: analysis, ProtectedRoots: []string{root}, Limits: eval_harness.MaintenanceLimits{MaxTrajectories: 2, MaxStages: 6, MaxStageSeconds: 10, MaxCampaignSeconds: 60, MaxRequests: 6, MaxTokens: 100}, Isolation: eval_harness.MaintenanceIsolation{Backend: "pi", Status: "unqualified", PolicyDigest: strings.Repeat("a", 64)}}
	for _, lang := range []string{"ailang", "python"} {
		stages := []eval_harness.MaintenanceStagePlan{}
		bindings := []eval_harness.MaintenanceStageBinding{}
		for _, id := range []string{"S1", "S2", "S3"} {
			stages = append(stages, eval_harness.MaintenanceStagePlan{ID: id, RequirementDigest: request.SHA256, OracleDigest: oraclePipeline})
			bindings = append(bindings, eval_harness.MaintenanceStageBinding{ID: id, Request: request, PublicCheck: eval_harness.MaintenanceCommandBinding{Command: []string{"fixture-check"}, Files: []eval_harness.MaintenanceFileRef{check}}, Oracle: eval_harness.MaintenanceCommandBinding{Command: []string{"fixture-oracle"}, Files: []eval_harness.MaintenanceFileRef{oracle}}})
		}
		plan.Trajectories = append(plan.Trajectories, eval_harness.MaintenanceTrajectoryPlan{ID: lang, Family: "real-family", Variant: "development", Repeat: 1, Language: lang, StarterDigest: starterDigest, NativeToolDigest: tool.SHA256, Stages: stages})
		m.Trajectories = append(m.Trajectories, eval_harness.MaintenanceTrajectoryBinding{ID: lang, Starter: assets, CarryFiles: []string{"main.py", "notes.md"}, Stages: bindings})
		m.Toolchains = append(m.Toolchains, eval_harness.MaintenanceToolchain{Language: lang, Command: []string{"fixture"}, Version: "fixture-only", Evidence: tool})
	}
	b, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	m.Plan = file("plan.json", string(b))
	return filepath.Join(root, "maintenance.execution.json"), m
}

func writeMaintenanceManifest(t *testing.T, path string, m eval_harness.MaintenanceExecutionManifest) {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
}

func maintenanceCLIStarterDigest(assets []eval_harness.MaintenanceAsset) (string, error) {
	type entry struct {
		Path       string `json:"path"`
		SHA256     string `json:"sha256"`
		Executable bool   `json:"executable"`
	}
	files := []entry{{assets[0].Destination, assets[0].SHA256, assets[0].Executable}}
	raw, err := json.Marshal(files)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:]), err
}
func TestMaintenanceModeProcess(t *testing.T) {
	if path := os.Getenv("MAINTENANCE_CLI_TEST_MANIFEST"); path != "" {
		os.Args = []string{"ailang", "eval-suite", "--agent", "--maintenance-manifest", path, "--dry-run"}
		before := os.Getenv("OTEL_RESOURCE_ATTRIBUTES")
		runEvalSuite()
		if os.Getenv("OTEL_RESOURCE_ATTRIBUTES") != before {
			os.Exit(29)
		}
		os.Exit(0)
	}
	path, m := maintenanceCLIManifestFixture(t)
	writeMaintenanceManifest(t, path, m)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMaintenanceModeProcess$")
	cmd.Env = append(os.Environ(), "MAINTENANCE_CLI_TEST_MANIFEST="+path, "OTEL_RESOURCE_ATTRIBUTES=maintenance_test=unchanged")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("dry-run subprocess: %v\n%s", err, stderr.String())
	}
	var report struct {
		DryRun        bool                  `json:"dryRun"`
		Schedule      []maintenanceDryStage `json:"schedule"`
		LiveAdmission string                `json:"liveAdmission"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("invalid dry-run: %v\n%s", err, stdout.String())
	}
	if !report.DryRun || len(report.Schedule) != 6 || report.LiveAdmission == "" {
		t.Fatalf("wrong report: %+v", report)
	}
	if stderr.Len() != 0 {
		t.Fatalf("dry-run produced telemetry/credential noise: %s", stderr.String())
	}
}
