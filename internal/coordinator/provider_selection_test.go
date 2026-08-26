package coordinator

import (
	"context"
	"strings"
	"testing"
)

// M-PIPELINE-RECONCILIATION follow-up, found live 2026-08-26: local provider
// selection ignored the agent's `provider:` config entirely. selectProvider
// returned the FIRST registered provider whose CanHandle was true — and
// ExecutorProvider.CanHandle is unconditionally true — with registration order
// varying per process. An eval-rig task (`provider: claude`) executed under
// motoko-cli on one restart and would have under opencode-cli on another. The
// agent's declared provider was decorative in the local path.

type selProbeProvider struct{ name string }

func (f *selProbeProvider) Name() string                      { return f.name }
func (f *selProbeProvider) CanHandle(task *AnalyzedTask) bool { return true }
func (f *selProbeProvider) Execute(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error) {
	return &ExecuteResult{Success: true, Provider: f.name}, nil
}

func TestSelectProvider_HonorsAgentConfig(t *testing.T) {
	te := &TaskExecutor{providers: []Provider{
		&selProbeProvider{name: "motoko-cli"},
		&selProbeProvider{name: "claude-cli"},
		&selProbeProvider{name: "gemini-api"},
	}}
	task := &AnalyzedTask{Type: TaskTypeTest}

	// The agent's declared provider wins regardless of registration order.
	opts := &ExecuteOptions{AgentConfig: &AgentConfig{ID: "a", Provider: "claude"}}
	res, err := te.Execute(context.Background(), task, opts)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Provider != "claude-cli" {
		t.Errorf("agent declared claude; got provider %q", res.Provider)
	}

	// "gemini" matches gemini-api (suffix varies by provider kind).
	opts = &ExecuteOptions{AgentConfig: &AgentConfig{ID: "b", Provider: "gemini"}}
	res, err = te.Execute(context.Background(), task, opts)
	if err != nil || res.Provider != "gemini-api" {
		t.Errorf("expected gemini-api, got %q (%v)", res.Provider, err)
	}
}

// A DECLARED provider that is not registered is an ERROR, not a fallback: an
// agent that says claude must get claude or fail saying why.
func TestSelectProvider_MissingDeclaredProviderIsLoud(t *testing.T) {
	te := &TaskExecutor{providers: []Provider{&selProbeProvider{name: "motoko-cli"}}}
	task := &AnalyzedTask{Type: TaskTypeTest}
	opts := &ExecuteOptions{AgentConfig: &AgentConfig{ID: "a", Provider: "claude"}}

	res, err := te.Execute(context.Background(), task, opts)
	if err != nil {
		t.Fatalf("execute returned transport error: %v", err)
	}
	if res.Success {
		t.Fatal("a missing declared provider must not silently run on another")
	}
	if !strings.Contains(res.Error, "claude") {
		t.Errorf("error should name the declared provider, got %q", res.Error)
	}
}

// No declared provider keeps the existing behavior (first capable provider).
func TestSelectProvider_NoDeclarationKeepsLegacyBehavior(t *testing.T) {
	te := &TaskExecutor{providers: []Provider{&selProbeProvider{name: "motoko-cli"}}}
	res, err := te.Execute(context.Background(), &AnalyzedTask{Type: TaskTypeTest}, &ExecuteOptions{})
	if err != nil || !res.Success {
		t.Fatalf("legacy path broke: %+v %v", res, err)
	}
}
