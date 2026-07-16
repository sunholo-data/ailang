package main

import (
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
