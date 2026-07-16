package main

import (
	"os"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
	// Blank-imported so the managed_agents executor registers itself in the
	// global factory (mirrors the blank import in exec.go). Required for
	// TestResolveAgenticExecutorNameReachable below.
	_ "github.com/sunholo-data/ailang/internal/executor/managed_agents"
)

// TestResolveAgenticExecutorName verifies the CLI-provider → executor-registry
// name mapping for the agentic path. In particular, "gemini" must resolve to
// "managed_agents" (Gemini CLI was retired in v0.22.0); everything else is
// identity.
func TestResolveAgenticExecutorName(t *testing.T) {
	cases := []struct {
		provider string
		want     string
	}{
		{"gemini", "managed_agents"},
		{"claude", "claude"},
		{"openai", "openai"},
		{"managed_agents", "managed_agents"},
	}
	for _, tc := range cases {
		if got := resolveAgenticExecutorName(tc.provider); got != tc.want {
			t.Errorf("resolveAgenticExecutorName(%q) = %q, want %q", tc.provider, got, tc.want)
		}
	}
}

// TestResolveAgenticExecutorNameReachable proves the gemini agentic lane is now
// reachable: resolving "gemini" and looking it up in the global executor
// factory returns the managed_agents executor with no error. This is a pure
// registry lookup — it makes NO network call to Vertex.
func TestResolveAgenticExecutorNameReachable(t *testing.T) {
	name := resolveAgenticExecutorName("gemini")
	exec, err := executor.GlobalFactory().GetExecutor(name)
	if err != nil {
		t.Fatalf("GetExecutor(%q) returned error: %v", name, err)
	}
	if exec == nil {
		t.Fatalf("GetExecutor(%q) returned nil executor", name)
	}
	if got := exec.Name(); got != "managed_agents" {
		t.Errorf("executor.Name() = %q, want %q", got, "managed_agents")
	}
}

// TestResolveGCPProjectEnv is the regression guard for
// M-GEMINI-EXEC-PROJECT-PLUMBING (Acceptance Criterion 4). It asserts the
// env precedence of resolveGCPProjectEnv() — AILANG_CLOUD_PROJECT wins over
// GOOGLE_CLOUD_PROJECT, falls back to GOOGLE_CLOUD_PROJECT, and is empty when
// neither is set (so the executor fails loud downstream, no silent default).
// Env is injected via t.Setenv (auto-restored per case) — no live GCP call.
func TestResolveGCPProjectEnv(t *testing.T) {
	cases := []struct {
		name       string
		ailangProj string
		googleProj string
		want       string
	}{
		{"ailang wins over google", "ailang-proj", "google-proj", "ailang-proj"},
		{"google fallback when ailang unset", "", "google-proj", "google-proj"},
		{"ailang used when google unset", "ailang-proj", "", "ailang-proj"},
		{"empty when neither set", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AILANG_CLOUD_PROJECT", tc.ailangProj)
			t.Setenv("GOOGLE_CLOUD_PROJECT", tc.googleProj)
			if got := resolveGCPProjectEnv(); got != tc.want {
				t.Errorf("resolveGCPProjectEnv() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestExecTaskGCPFieldsFromEnv asserts that the exec.go Task construction picks
// up the resolved GCP project and the GOOGLE_CLOUD_LOCATION env into the
// executor.Task the same way executeCLI builds it — the concrete plumbing that
// unblocks `ailang exec gemini`. Building the Task here (rather than calling
// executeCLI, which would make a live Vertex call) keeps the test hermetic.
func TestExecTaskGCPFieldsFromEnv(t *testing.T) {
	t.Setenv("AILANG_CLOUD_PROJECT", "ailang-proj")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "google-proj")
	t.Setenv("GOOGLE_CLOUD_LOCATION", "europe-west3")

	task := &executor.Task{
		GCPProject:  resolveGCPProjectEnv(),
		GCPLocation: os.Getenv("GOOGLE_CLOUD_LOCATION"), // mirrors exec.go executeCLI
	}
	if task.GCPProject != "ailang-proj" {
		t.Errorf("task.GCPProject = %q, want %q", task.GCPProject, "ailang-proj")
	}
	if task.GCPLocation != "europe-west3" {
		t.Errorf("task.GCPLocation = %q, want %q", task.GCPLocation, "europe-west3")
	}
}
