package motoko

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// TestRegistration_Motoko verifies init() registers "motoko" in the global
// factory and that GetExecutor("motoko") returns a *MotokoExecutor. This is
// the EXECUTOR_SHAPE.md §3 contract — without it the coordinator's
// NewExecutorProvider("motoko") would fail to resolve.
func TestRegistration_Motoko(t *testing.T) {
	cfg := testConfig()
	executor.SetGlobalFactory(executor.NewFactory(cfg))
	Register()

	exec, err := executor.GlobalFactory().GetExecutor("motoko")
	if err != nil {
		t.Fatalf("GetExecutor(\"motoko\") failed: %v", err)
	}
	if exec.Name() != "motoko" {
		t.Errorf("Name() = %q, want \"motoko\"", exec.Name())
	}
	if _, ok := exec.(*MotokoExecutor); !ok {
		t.Errorf("expected *MotokoExecutor, got %T", exec)
	}
}

// TestRegister_Idempotent verifies Register() can be called multiple times
// without panic (factory.Register overwrites cleanly).
func TestRegister_Idempotent(t *testing.T) {
	executor.SetGlobalFactory(executor.NewFactory(testConfig()))
	Register()
	Register()
	Register()
	// no panic = pass
}

// TestNew_DefaultsApply verifies New() applies default values when the Config
// fields are empty strings (the EXECUTOR_SHAPE.md §2 contract for constructors).
func TestNew_PathDefaultsApplyButModelDoesNot(t *testing.T) {
	// M-MODEL-REGISTRY-SINGLE-SOURCE M6 (D2(a)). Path and profile still default;
	// the model does not. Finding the `motoko` binary on PATH is a lookup that
	// decides nothing, while choosing a model decides billing and availability —
	// motoko's default was openrouter/anthropic/claude-haiku-4-5, on the provider
	// the fleet has migrated off.
	exec, err := New(&executor.Config{MotokoPath: "", MotokoModel: "", MotokoProfile: ""})
	if err != nil {
		t.Fatalf("construction must succeed with no model (the coordinator supplies it per task): %v", err)
	}
	if exec.motokoPath != "motoko" {
		t.Errorf("motokoPath = %q, want \"motoko\"", exec.motokoPath)
	}
	if exec.model != "" {
		t.Errorf("model = %q, want empty: no default may be substituted", exec.model)
	}
	// And with no model anywhere, execution — not construction — is what fails.
	if err := exec.requireModel(&executor.Task{ID: "t"}); err == nil {
		t.Error("requireModel must fail when neither task nor executor names a model")
	}
}

// TestNew_ConfigOverrides verifies non-empty Config fields override defaults.
func TestNew_ConfigOverrides(t *testing.T) {
	exec, err := New(&executor.Config{
		MotokoPath:    "/custom/path/motoko",
		MotokoModel:   "openrouter/z-ai/glm-5",
		MotokoProfile: "stress-test",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if exec.motokoPath != "/custom/path/motoko" {
		t.Errorf("motokoPath = %q, want override", exec.motokoPath)
	}
	if exec.model != "openrouter/z-ai/glm-5" {
		t.Errorf("model = %q, want override", exec.model)
	}
	if exec.profile != "stress-test" {
		t.Errorf("profile = %q, want override", exec.profile)
	}
}

// TestCapabilities verifies the Capabilities() set matches what motoko
// actually supports (streaming via JSONL file growth, local workspace,
// structured JSONL output).
func TestCapabilities(t *testing.T) {
	exec, _ := New(testConfig())
	caps := exec.Capabilities()

	want := map[executor.Capability]bool{
		executor.CapStreaming:        true,
		executor.CapLocalWorkspace:   true,
		executor.CapStructuredOutput: true,
	}
	got := make(map[executor.Capability]bool)
	for _, c := range caps {
		got[c] = true
	}
	for cap := range want {
		if !got[cap] {
			t.Errorf("missing capability %q", cap)
		}
	}
}

// TestHealthCheck_BinaryMissing verifies HealthCheck refuses a missing
// binary, and the error names + QUOTES the configured path (the refusal-branch
// rule: an error value is reachable from every branch, so the observable must
// be the path — plan §3.1, row T-B1).
func TestHealthCheck_BinaryMissing(t *testing.T) {
	exec, _ := New(&executor.Config{MotokoPath: "/definitely/not/a/real/binary/motoko-xyz",
		MotokoModel: "openrouter/anthropic/claude-haiku-4-5", // D2(a): model is required
	})
	err := exec.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("HealthCheck succeeded for missing binary; expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "motoko CLI not found at") ||
		!strings.Contains(msg, `"/definitely/not/a/real/binary/motoko-xyz"`) {
		t.Errorf("missing-binary error must name + quote the configured path; got: %q", msg)
	}
}

// TestHealthCheck_PathIsDirectory covers the info.IsDir() branch (T-B2): the
// error must name the configured path AND the "is a directory" diagnosis.
func TestHealthCheck_PathIsDirectory(t *testing.T) {
	tmpdir := t.TempDir()
	dirPath := filepath.Join(tmpdir, "motoko")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	exec, _ := New(&executor.Config{MotokoPath: dirPath,
		MotokoModel: "openrouter/anthropic/claude-haiku-4-5", // D2(a): model is required
	})
	err := exec.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("HealthCheck succeeded against a directory; expected error")
	}
	msg := err.Error()
	// Compare against the %q-QUOTED path, not the raw one. The refusal formats the
	// path with %q, and on windows a path contains backslashes which %q escapes —
	// so `strings.Contains(msg, dirPath)` can never match there even though the
	// message names the path correctly. Caught by Gate 3b's windows leg, which is
	// the only instrument that sees the whole matrix.
	if !strings.Contains(msg, "is a directory") || !strings.Contains(msg, strconv.Quote(dirPath)) {
		t.Errorf("directory-path error must name the path and diagnosis; got: %q (wanted quoted path %s)", msg, strconv.Quote(dirPath))
	}
}

// TestHealthCheck_NotExecutable covers the exec-bit branch (T-B3). The branch
// is guarded runtime.GOOS != "windows", so this row is darwin/posix-only by
// construction — same skip pattern the plan assigns to it.
func TestHealthCheck_NotExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("exec-bit refusal branch is darwin/posix-only (runtime.GOOS != windows)")
	}
	tmpdir := t.TempDir()
	mockPath := filepath.Join(tmpdir, "motoko")
	if err := os.WriteFile(mockPath, []byte("#!/bin/bash\nnot a real binary\n"), 0o644); err != nil {
		t.Fatalf("write non-executable file: %v", err)
	}
	exec, _ := New(&executor.Config{MotokoPath: mockPath,
		MotokoModel: "openrouter/anthropic/claude-haiku-4-5", // D2(a): model is required
	})
	err := exec.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("HealthCheck succeeded for a non-executable file; expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "is not executable (chmod +x)") || !strings.Contains(msg, mockPath) {
		t.Errorf("non-executable error must name the diagnosis and path; got: %q", msg)
	}
}

// TestHealthCheck_MockBinary_VersionQueryAndNoKeyRefusal covers T-B4: on a
// mock that answers `motoko --version`, HealthCheck returns NIL even with the
// key UNSET (the behaviour change — this is the exact refusal M1 deleted), and
// the version query still fires -> e.motokoRepo is populated. The motokoRepo
// assertion is the positive half: "no error" is satisfiable by a HealthCheck
// that never ran its version query at all.
func TestHealthCheck_MockBinary_VersionQueryAndNoKeyRefusal(t *testing.T) {
	// The mock is a `#!/bin/bash` script named `motoko` with no `.exe` suffix, so
	// windows cannot execute it: the version query fails and motokoRepo stays
	// empty, failing for the PLATFORM rather than for the code. This row is
	// therefore darwin/posix-only by construction — the same skip the 10 bash-mock
	// arms in execute_test.go already carry. The no-key-refusal half of T-B4 is
	// covered platform-independently by the provider_preflight tests.
	if runtime.GOOS == "windows" {
		t.Skip("bash mock binary requires POSIX shell")
	}
	tmpdir := t.TempDir()
	mockPath := filepath.Join(tmpdir, "motoko")
	// Deterministic: whatever the ambient shell holds, this row runs key-unset.
	t.Setenv("OPENROUTER_API_KEY", "")
	mock := "#!/bin/bash\n" +
		"echo \"tui_version=1.2.3\"\n" +
		"echo \"git_rev=abc123\"\n" +
		"echo \"ailang_built=1\"\n" +
		"echo \"motoko_repo=/tmp/fake-motoko-repo\"\n" +
		"exit 0\n"
	if err := os.WriteFile(mockPath, []byte(mock), 0755); err != nil {
		t.Fatalf("failed to write mock binary: %v", err)
	}

	exec, _ := New(&executor.Config{MotokoPath: mockPath,
		MotokoModel: "openrouter/anthropic/claude-haiku-4-5", // D2(a): model is required
	})
	if err := exec.HealthCheck(context.Background()); err != nil {
		t.Errorf("HealthCheck failed with key unset (was nil refusal): %v", err)
	}
	if exec.motokoRepo != "/tmp/fake-motoko-repo" {
		t.Errorf("version query did not populate motokoRepo (got %q, want /tmp/fake-motoko-repo)",
			exec.motokoRepo)
	}
}

// TestClose verifies Close is a no-op (no resources to release).
func TestClose(t *testing.T) {
	exec, _ := New(testConfig())
	if err := exec.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}
