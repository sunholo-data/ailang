package eval_harness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// This opt-in gate is executed explicitly by the sprint. Missing runtimes do
// not count as qualified semantic or containment evidence in ordinary Go CI.
func TestMaintenanceRetainedNativeEightByThree(t *testing.T) {
	packageRoot := os.Getenv("AILANG_MAINTENANCE_PACKAGE_ROOT")
	if packageRoot == "" {
		t.Skip("set AILANG_MAINTENANCE_PACKAGE_ROOT to run actual native integration")
	}
	for _, binary := range []string{"python3", "ailang"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Fatal(err)
		}
	}
	packageRoot, err := filepath.Abs(packageRoot)
	if err != nil {
		t.Fatal(err)
	}
	repository := filepath.Dir(filepath.Dir(packageRoot))
	taskpacks := filepath.Join(repository, "research", "real-world-maintenance", "taskpacks", "v1")
	e := maintenanceNativeExecution(t, taskpacks)
	controller := t.TempDir()
	grader, err := NewMaintenanceNativeGrader(e, MaintenanceNativeGradeOptions{EvidenceRoot: filepath.Join(controller, "private-grades"), DevelopmentUnconfined: true})
	if err != nil {
		t.Fatal(err)
	}
	scripted := &maintenanceScriptedExecutor{}
	scripted.run = func(_ context.Context, task *executor.Task) (*executor.Result, error) {
		family, language, stage := task.Metadata["family"], task.Metadata["language"], task.Metadata["stage"]
		id := task.Metadata["trajectory"]
		// Prove the broken prior source reaches the next fresh session before our
		// scripted executor repairs it. This is a model-free instrument control.
		if stage == "S2" && strings.HasPrefix(id, "resource-ownership-1-") {
			entry := "main.py"
			marker := "\"global\""
			if language == "ailang" {
				entry = "main.ail"
				marker = "BROKEN_RETAINED_SOURCE"
			}
			previous, err := os.ReadFile(filepath.Join(task.Workspace, entry))
			if err != nil || !strings.Contains(string(previous), marker) {
				return nil, fmt.Errorf("failed source was silently restored: %s %v", id, err)
			}
		}
		reference := filepath.Join(taskpacks, family, "reference", language)
		if family == "resource-ownership" {
			reference = filepath.Join(taskpacks, family, "references", strings.ToLower(stage), language)
		}
		for _, binding := range e.Manifest.Trajectories {
			if binding.ID != id {
				continue
			}
			for _, name := range binding.CarryFiles {
				original := filepath.Join(reference, filepath.FromSlash(name))
				data, err := os.ReadFile(original)
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					return nil, err
				}
				target := filepath.Join(task.Workspace, filepath.FromSlash(name))
				if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
					return nil, err
				}
				if err = os.WriteFile(target, data, 0600); err != nil {
					return nil, err
				}
			}
		}
		if id == "resource-ownership-1-ailang" && stage == "S1" {
			if err = os.WriteFile(filepath.Join(task.Workspace, "main.ail"), []byte("BROKEN_RETAINED_SOURCE"), 0600); err != nil {
				return nil, err
			}
		}
		if id == "resource-ownership-1-python" && stage == "S1" {
			path := filepath.Join(task.Workspace, "main.py")
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			from := "hashlib.sha256(doc.encode()).hexdigest()"
			if !strings.Contains(string(data), from) {
				return nil, fmt.Errorf("semantic control anchor absent")
			}
			if err = os.WriteFile(path, []byte(strings.ReplaceAll(string(data), from, "\"global\"")), 0600); err != nil {
				return nil, err
			}
		}
		// SYNTHETIC typed parser evidence exercises export/joins. These are not
		// measured provider tokens, prices or model outcomes.
		rawUsage, _ := json.Marshal(map[string]string{"role": "SYNTHETIC_INSTRUMENT_CONTROL", "attempt": task.ID})
		sum := sha256.Sum256(rawUsage)
		eventHash := hex.EncodeToString(sum[:])
		provider, model, request := "SCRIPTED-CONTROL", e.Plan.Model, task.ID+"/synthetic-request"
		input, output, zero := 1, 1, 0
		evidence := &executor.UsageEvidence{Source: "SCRIPTED-INSTRUMENT", Version: "synthetic/1", StreamDigest: eventHash, Settled: true, CoverageComplete: false, UnavailableReasons: []string{"synthetic control; actual provider coverage unmeasured"}, Observations: []executor.UsageObservation{{ImportID: "synthetic-leaf", EventDigest: eventHash, RequestID: &request, Provider: &provider, Model: &model, Input: &input, Output: &output, CacheRead: &zero, CacheWrite: &zero, Reasoning: &zero, Contract: "exclusive/1", OutputScheme: "combined", CostBasis: "unknown", RawUsage: rawUsage}}}
		return &executor.Result{Success: true, FinishReason: executor.FinishStop, ExecutorVersion: e.Plan.Harness, UsageEvidence: evidence}, nil
	}
	builder := func(tr MaintenanceTrajectoryPlan, stage MaintenanceStagePlan, workspace, request string) (*executor.Task, error) {
		return &executor.Task{Workspace: workspace, Directive: request, Model: e.Plan.Model, Timeout: time.Second, Metadata: map[string]string{"family": tr.Family, "language": tr.Language, "stage": stage.ID, "trajectory": tr.ID}}, nil
	}
	collector := NewMaintenanceUsageCollector(e)
	runRoot := filepath.Join(controller, "stages")
	report, err := RunMaintenanceStages(context.Background(), MaintenanceRunOptions{Execution: e, Root: runRoot, Executor: scripted, BuildTask: builder, Grade: grader, BeforeDispatch: maintenanceAdmission, DevelopmentInstrument: true, CollectAttempt: collector.CollectAttempt})
	if err != nil {
		t.Fatal(err)
	}
	if len(scripted.calls) != 24 || len(report.Attempts) != 24 {
		t.Fatalf("schedule/spend disappeared: calls%d attempts%d", len(scripted.calls), len(report.Attempts))
	}
	failures, completed := 0, 0
	for _, trajectory := range report.Runs.Trajectories {
		for _, stage := range trajectory.Stages {
			switch stage.State {
			case "failed":
				failures++
			case "completed":
				completed++
			default:
				t.Fatalf("native stage not measured: %s/%s %+v", trajectory.ID, stage.ID, stage)
			}
		}
	}
	if failures != 2 || completed != 22 {
		t.Fatalf("wrong native outcomes: failures%d completed%d", failures, completed)
	}
	broken := report.Runs.Trajectories[0].Stages[0]
	regression := report.Runs.Trajectories[1].Stages[0]
	if broken.Visible != "fail" || broken.Oracle != "not_run" || regression.Visible != "pass" || regression.Oracle != "fail" {
		t.Fatalf("artifact/semantic failures conflated: %+v %+v", broken, regression)
	}
	usage := filepath.Join(controller, "usage.json")
	if err = collector.WriteNew(usage); err != nil {
		t.Fatal(err)
	}
	leaves, summaries := 0, 0
	for _, receipt := range collector.Bundle().Receipts {
		if receipt.Identity.Source != "SCRIPTED-INSTRUMENT" {
			t.Fatalf("synthetic usage lost source role: %+v", receipt)
		}
		if receipt.Role == "leaf" {
			leaves++
			if receipt.Identity.Provider == nil || *receipt.Identity.Provider != "SCRIPTED-CONTROL" {
				t.Fatal("synthetic usage relabelled as provider measurement")
			}
		}
		if receipt.Role == "summary" {
			summaries++
			if receipt.ChildrenComplete {
				t.Fatal("unmeasured provider coverage became complete")
			}
		}
	}
	if leaves != 24 || summaries != 24 {
		t.Fatalf("attempt usage lost: %d leaves %d summaries", leaves, summaries)
	}
	output := filepath.Join(controller, "native-analysis.json")
	_, err = RunMaintenanceNativeAnalysis(context.Background(), MaintenanceAnalysisInput{Binary: "ailang", PackageRoot: packageRoot, PlanPath: e.Manifest.Plan.Path, RunsPath: filepath.Join(runRoot, "runs.json"), UsagePath: usage, OutputPath: output})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var native struct {
		Data struct {
			Summary struct {
				ScheduledStages int `json:"scheduledStages"`
				Successful      int `json:"successfulTrajectories"`
				Failed          int `json:"knownFailedTrajectories"`
				Regressions     int `json:"visibleGreenRegressions"`
			}
			Comparison struct {
				CostStatus string   `json:"costStatus"`
				CostDelta  *float64 `json:"costPerSuccessDeltaMicros"`
			}
		}
	}
	if err = json.Unmarshal(raw, &native); err != nil {
		t.Fatal(err)
	}
	if native.Data.Summary.ScheduledStages != 24 || native.Data.Summary.Successful != 6 || native.Data.Summary.Failed != 2 || native.Data.Summary.Regressions != 1 {
		t.Fatalf("native analysis did not count actual outcomes: %s", raw)
	}
	if native.Data.Comparison.CostStatus == "available" || native.Data.Comparison.CostDelta != nil {
		t.Fatalf("unreported model usage became economics: %s", raw)
	}
}

func maintenanceNativeExecution(t *testing.T, taskpacks string) *MaintenanceExecution {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	script := `import importlib.util,json,sys,shutil,subprocess
from pathlib import Path
folder=Path(sys.argv[1]);dest=Path(sys.argv[2]);spec=importlib.util.spec_from_file_location('freeze',folder/'execution/write_development_manifest.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
p,e=m.build(folder/'execution/development.ailang-toolchain.json',folder/'execution/development.python-toolchain.json')
def absolute(ref):ref['path']=str((m.HERE/ref['path']).resolve())
for key in ('settings','analysis'):absolute(e[key])
e['packageRoot']=str((m.HERE/e['packageRoot']).resolve());e['protectedRoots']=[str((m.HERE/x).resolve()) for x in e['protectedRoots']]+[str(dest)]
# This is a NEW no-model instrument cohort on this host, not a rewrite of
# the checked-in historical freeze. Observe actual binaries before dispatch.
observed={}
for tool in e['toolchains']:
 command=tool['command']
 query=command+(['version'] if tool['language']=='ailang' else ['--version'])
 result=subprocess.run(query,capture_output=True,text=True,check=True,timeout=30)
 binary=Path(shutil.which(command[0])).resolve(strict=True)
 value={'schema':'maintenance.toolchain/1','command':command,'versionCommand':query,'version':(result.stdout+result.stderr).strip(),'binarySHA256':m.sha(binary.read_bytes())}
 evidence=dest/('observed-'+tool['language']+'-toolchain.json')
 m.write_new(evidence,value)
 tool['command']=value['command'];tool['version']=value['version']
 tool['evidence']={'path':str(evidence),'sha256':m.sha(evidence.read_bytes())}
 observed[tool['language']]=tool['evidence']['sha256']
for trajectory in p['trajectories']:trajectory['nativeToolDigest']=observed[trajectory['language']]
for trajectory in e['trajectories']:
 for asset in trajectory['starter']:absolute(asset)
 for stage in trajectory['stages']:
  absolute(stage['request'])
  for key in ('publicCheck','oracle'):
   for ref in stage[key]['files']:absolute(ref)
plan=dest/'plan.json';m.write_new(plan,p);e['plan']={'path':str(plan),'sha256':m.sha(plan.read_bytes())};m.write_new(dest/'execution.json',e)
`
	command := exec.Command("python3", "-B", "-c", script, taskpacks, root)
	if raw, err := command.CombinedOutput(); err != nil {
		t.Fatalf("native fixture freeze: %v %s", err, raw)
	}
	e, err := LoadMaintenanceExecutionManifest(filepath.Join(root, "execution.json"))
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestMaintenanceCheckpointDigestMatchesPython(t *testing.T) {
	packageRoot := os.Getenv("AILANG_MAINTENANCE_PACKAGE_ROOT")
	if packageRoot == "" {
		t.Skip("set AILANG_MAINTENANCE_PACKAGE_ROOT for cross-language digest check")
	}
	if runtime.GOOS == "windows" {
		t.Skip("HTML punctuation fixture filename is not representable on Windows")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Dir(filepath.Dir(packageRoot))
	moduleRoot := filepath.Join(repo, "research", "real-world-maintenance", "taskpacks", "v1")
	for _, mode := range []os.FileMode{0600, 0644, 0755} {
		root := t.TempDir()
		source := filepath.Join(root, "source")
		if err := os.Mkdir(source, 0700); err != nil {
			t.Fatal(err)
		}
		name := "app&<>🍉\u2028\u2029.py"
		path := filepath.Join(source, name)
		if err := os.WriteFile(path, []byte("owned Unicode source"), mode); err != nil {
			t.Fatal(err)
		}
		cp, err := CaptureMaintenanceCheckpoint(source, filepath.Join(root, "checkpoint"), []string{name})
		if err != nil {
			t.Fatal(err)
		}
		script := `import sys;sys.path.insert(0,sys.argv[1]);from candidate_grade import source_digest;print(source_digest(sys.argv[2]))`
		output, err := exec.Command("python3", "-B", "-c", script, moduleRoot, filepath.Join(cp.Root, "files")).CombinedOutput()
		if err != nil {
			t.Fatalf("Python digest: %v %s", err, output)
		}
		if strings.TrimSpace(string(output)) != cp.Digest {
			t.Fatalf("cross-language digest mismatch mode%o: Go%s Python%s", mode, cp.Digest, output)
		}
	}
}

func TestMaintenanceNativeExecutionFreezesCurrentHost(t *testing.T) {
	packageRoot := os.Getenv("AILANG_MAINTENANCE_PACKAGE_ROOT")
	if packageRoot == "" {
		t.Skip("set AILANG_MAINTENANCE_PACKAGE_ROOT for current-host instrument freeze")
	}
	taskpacks := filepath.Join(filepath.Dir(filepath.Dir(packageRoot)), "research", "real-world-maintenance", "taskpacks", "v1")
	historic := map[string]string{}
	for _, language := range []string{"ailang", "python"} {
		path := filepath.Join(taskpacks, "execution", "development."+language+"-toolchain.json")
		hash, err := maintenanceFileSHA(path)
		if err != nil {
			t.Fatal(err)
		}
		historic[path] = hash
	}
	e := maintenanceNativeExecution(t, taskpacks)
	for _, tc := range e.Manifest.Toolchains {
		frozen, err := maintenanceReadToolchainEvidence(tc)
		if err != nil {
			t.Fatal(err)
		}
		binary, err := exec.LookPath(tc.Command[0])
		if err != nil {
			t.Fatal(err)
		}
		binary, err = filepath.EvalSymlinks(binary)
		if err != nil {
			t.Fatal(err)
		}
		actual, err := maintenanceFileSHA(binary)
		if err != nil {
			t.Fatal(err)
		}
		if frozen.BinarySHA256 != actual {
			t.Fatalf("%s current binary was not frozen: frozen%s actual%s", tc.Language, frozen.BinarySHA256, actual)
		}
		if !maintenancePathWithin(e.Root, tc.Evidence.Path) {
			t.Fatalf("host evidence was not newly created in instrument root: %s", tc.Evidence.Path)
		}
	}
	for path, before := range historic {
		after, err := maintenanceFileSHA(path)
		if err != nil || before != after {
			t.Fatal("historical toolchain evidence was modified")
		}
	}
}
