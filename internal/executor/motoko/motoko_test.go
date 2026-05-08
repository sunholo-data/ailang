package motoko

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// TestRegistration_Motoko verifies init() registers "motoko" in the global
// factory and that GetExecutor("motoko") returns a *MotokoExecutor. This is
// the EXECUTOR_SHAPE.md §3 contract — without it the coordinator's
// NewExecutorProvider("motoko") would fail to resolve.
func TestRegistration_Motoko(t *testing.T) {
	cfg := executor.DefaultConfig()
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
	executor.SetGlobalFactory(executor.NewFactory(executor.DefaultConfig()))
	Register()
	Register()
	Register()
	// no panic = pass
}

// TestNew_DefaultsApply verifies New() applies default values when the Config
// fields are empty strings (the EXECUTOR_SHAPE.md §2 contract for constructors).
func TestNew_DefaultsApply(t *testing.T) {
	exec, err := New(&executor.Config{
		MotokoPath:    "",
		MotokoModel:   "",
		MotokoProfile: "",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if exec.motokoPath != "motoko" {
		t.Errorf("motokoPath = %q, want \"motoko\"", exec.motokoPath)
	}
	if exec.model == "" {
		t.Errorf("model is empty, want a default openrouter/* string")
	}
	if exec.profile != "dogfood" {
		t.Errorf("profile = %q, want \"dogfood\"", exec.profile)
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
	exec, _ := New(executor.DefaultConfig())
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

// TestHealthCheck_BinaryMissing verifies HealthCheck returns a clear error
// when the configured motoko path does not exist.
func TestHealthCheck_BinaryMissing(t *testing.T) {
	exec, _ := New(&executor.Config{MotokoPath: "/definitely/not/a/real/binary/motoko-xyz"})
	err := exec.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("HealthCheck succeeded for missing binary; expected error")
	}
}

// TestHealthCheck_MockBinary uses a POSIX shell stub that responds to
// --version. Exercises the success path without requiring real motoko on
// PATH (CI-safe).
func TestHealthCheck_MockBinary(t *testing.T) {
	tmpdir := t.TempDir()
	mockPath := filepath.Join(tmpdir, "motoko")
	mockScript := "#!/bin/bash\nif [ \"$1\" = \"--version\" ]; then echo \"motoko mock 0.0.1\"; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(mockPath, []byte(mockScript), 0755); err != nil {
		t.Fatalf("failed to write mock binary: %v", err)
	}

	exec, _ := New(&executor.Config{MotokoPath: mockPath})
	if err := exec.HealthCheck(context.Background()); err != nil {
		t.Errorf("HealthCheck against mock binary failed: %v", err)
	}
}

// TestClose verifies Close is a no-op (no resources to release).
func TestClose(t *testing.T) {
	exec, _ := New(executor.DefaultConfig())
	if err := exec.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}
