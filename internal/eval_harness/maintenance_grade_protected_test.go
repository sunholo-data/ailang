package eval_harness

import (
	"context"
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
	"github.com/sunholo-data/ailang/internal/executor/pi"
)

func TestMaintenanceProtectedNativeGrading(t *testing.T) {
	packageRoot := os.Getenv("AILANG_MAINTENANCE_PACKAGE_ROOT")
	if packageRoot == "" {
		t.Skip("set AILANG_MAINTENANCE_PACKAGE_ROOT for actual protected native grading")
	}
	if runtime.GOOS != "darwin" {
		t.Fatal("this recorded qualification is macOS-specific; no unsupported-platform skip")
	}
	sdkRoot := os.Getenv("AILANG_MAINTENANCE_SANDBOX_ROOT")
	if sdkRoot == "" {
		sdkRoot = filepath.Join("..", "..", "tools", "pi-extensions", "sandbox")
	}
	sdkRoot, err := filepath.EvalSymlinks(sdkRoot)
	if err != nil {
		t.Fatal(err)
	}
	sdkRoot, err = filepath.Abs(sdkRoot)
	if err != nil {
		t.Fatal(err)
	}
	packageRoot, err = filepath.EvalSymlinks(packageRoot)
	if err != nil {
		t.Fatal(err)
	}
	repository := filepath.Dir(filepath.Dir(packageRoot))
	taskpacks := filepath.Join(repository, "research", "real-world-maintenance", "taskpacks", "v1")
	e := maintenanceNativeExecution(t, taskpacks)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(root, "app")
	controller := filepath.Join(root, "controller")
	for _, path := range []string{app, controller} {
		if err = os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	private := filepath.Join(controller, "private-oracle.txt")
	privateBytes := "PRIVATE_GRADER_ORACLE_CANARY"
	if err = os.WriteFile(private, []byte(privateBytes), 0600); err != nil {
		t.Fatal(err)
	}
	owned := filepath.Join(app, "owned.txt")
	if err = os.WriteFile(owned, []byte("OWNED_APP_CANARY"), 0600); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(sdkRoot, "package-lock.json")
	lockHash, err := maintenanceFileSHA(lock)
	if err != nil {
		t.Fatal(err)
	}
	p := executor.EvalContainmentPolicy{Schema: "maintenance.containment/1", WorkspaceRoot: app, ProtectedRoots: append(append([]string{}, e.Manifest.ProtectedRoots...), controller), ReadOnlyRoots: []string{}, RuntimeModule: filepath.Join(sdkRoot, "node_modules", "@anthropic-ai", "sandbox-runtime", "dist", "index.js"), RuntimeVersion: "0.0.71", RuntimeLock: lock, RuntimeLockDigest: lockHash, InferenceDomains: []string{}}
	bytes, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	policy := filepath.Join(root, "policy.json")
	if err = os.WriteFile(policy, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	policyHash, err := maintenanceFileSHA(policy)
	if err != nil {
		t.Fatal(err)
	}
	prefix, err := pi.EvalContainmentPrefix(&executor.EvalContainment{PolicyPath: policy, PolicyDigest: policyHash}, app)
	if err != nil {
		t.Fatal(err)
	}
	admitted := false
	admission := func(ctx context.Context) error {
		if admitted {
			return nil
		}
		for _, probe := range []struct {
			path, want string
			allowed    bool
		}{{owned, "OWNED_APP_CANARY", true}, {private, privateBytes, false}} {
			argv := make([]string, len(prefix))
			for i, s := range prefix {
				argv[i] = strings.ReplaceAll(s, "{runtimeRoot}", app)
			}
			argv = append(argv, "--", "/bin/cat", probe.path)
			bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
			command := exec.CommandContext(bounded, argv[0], argv[1:]...)
			command.Dir = app
			output, err := command.CombinedOutput()
			cancel()
			if probe.allowed && (err != nil || strings.TrimSpace(string(output)) != probe.want) {
				return fmt.Errorf("owned positive control denied: %v %s", err, output)
			}
			if !probe.allowed && (err == nil || strings.Contains(string(output), probe.want)) {
				return fmt.Errorf("private oracle read was admitted: %v %s", err, output)
			}
		}
		admitted = true
		return nil
	}
	grader, err := NewMaintenanceNativeGrader(e, MaintenanceNativeGradeOptions{EvidenceRoot: filepath.Join(controller, "grades"), CommandPrefix: prefix, ContainmentDigest: policyHash, Admission: admission})
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for index, tr := range e.Plan.Trajectories {
		if tr.Repeat != 1 {
			continue
		}
		binding := e.Manifest.Trajectories[index]
		reference := filepath.Join(taskpacks, tr.Family, "reference", tr.Language)
		if tr.Family == "resource-ownership" {
			reference = filepath.Join(taskpacks, tr.Family, "references", "s1", tr.Language)
		}
		cp, err := CaptureMaintenanceCheckpoint(reference, filepath.Join(controller, fmt.Sprintf("reference-%d", index)), binding.CarryFiles)
		if err != nil {
			t.Fatal(err)
		}
		got, err := grader(context.Background(), tr, tr.Stages[0], cp)
		if err != nil || got.Visible != "pass" || got.Oracle != "pass" {
			t.Fatalf("normal confined native %s: %+v %v", tr.ID, got, err)
		}
		malicious := filepath.Join(root, fmt.Sprintf("malicious-%d", index))
		if err = RestoreMaintenanceCheckpoint(cp, malicious); err != nil {
			t.Fatal(err)
		}
		entry := "main.py"
		body := fmt.Sprintf("print(open(%q).read())\n", private)
		if tr.Language == "ailang" {
			entry = "main.ail"
			before, err := os.ReadFile(filepath.Join(malicious, entry))
			if err != nil {
				t.Fatal(err)
			}
			module := strings.SplitN(string(before), "\n", 2)[0]
			encoded, _ := json.Marshal(private)
			body = fmt.Sprintf("%s\nimport std/fs (readFile)\nexport func main() -> () ! {IO, FS} { println(readFile(%s)) }\n", module, encoded)
		}
		if err = os.WriteFile(filepath.Join(malicious, entry), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		bad, err := CaptureMaintenanceCheckpoint(malicious, filepath.Join(controller, fmt.Sprintf("attack-%d", index)), binding.CarryFiles)
		if err != nil {
			t.Fatal(err)
		}
		rejected, err := grader(context.Background(), tr, tr.Stages[0], bad)
		if err != nil || rejected.Visible != "fail" || rejected.Oracle != "not_run" {
			t.Fatalf("confined oracle read %s: %+v %v", tr.ID, rejected, err)
		}
		checked++
	}
	if checked != 4 || !admitted {
		t.Fatalf("protected smoke coverage %d admitted%v", checked, admitted)
	}
	actual, err := os.ReadFile(private)
	if err != nil || string(actual) != privateBytes {
		t.Fatal("oracle data changed")
	}
}
