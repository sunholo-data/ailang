package cloudrun

import (
	"context"
	"fmt"
	"testing"

	run "cloud.google.com/go/run/apiv2"
	runpb "cloud.google.com/go/run/apiv2/runpb"
	"github.com/googleapis/gax-go/v2"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// mockJobRunner records RunJob calls for verification.
type mockJobRunner struct {
	lastReq *runpb.RunJobRequest
	err     error
}

func (m *mockJobRunner) RunJob(ctx context.Context, req *runpb.RunJobRequest, opts ...gax.CallOption) (*run.RunJobOperation, error) {
	m.lastReq = req
	if m.err != nil {
		return nil, m.err
	}
	return nil, nil // Operation handle not needed — we only check errors
}

func TestDispatcherImplementsInterface(t *testing.T) {
	// Compile-time check via var _ above, but verify at runtime too
	var d interface{} = &Dispatcher{}
	if _, ok := d.(coordinator.CloudDispatcher); !ok {
		t.Fatal("Dispatcher does not implement coordinator.CloudDispatcher")
	}
}

func TestDispatchJobName(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "my-project", "europe-west1", "ailang")

	err := d.Dispatch(context.Background(), coordinator.DispatchParams{
		TaskID:  "task-abc123",
		AgentID: "sprint-executor",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "projects/my-project/locations/europe-west1/jobs/ailang-agent-executor"
	if mock.lastReq.Name != expected {
		t.Errorf("job name = %q, want %q", mock.lastReq.Name, expected)
	}
}

func TestDispatchEnvVarOverrides(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "proj-1", "us-central1", "test")

	params := coordinator.DispatchParams{
		TaskID:    "task-12345678",
		AgentID:   "design-doc-creator",
		Workspace: "/workspace/ailang",
		Provider:  "claude",
		Directive: "Fix the parser bug",
		RepoURL:   "https://github.com/sunholo-data/ailang",
		Branch:    "dev",
	}

	err := d.Dispatch(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify overrides structure
	overrides := mock.lastReq.Overrides
	if overrides == nil {
		t.Fatal("overrides is nil")
	}
	if len(overrides.ContainerOverrides) != 1 {
		t.Fatalf("expected 1 container override, got %d", len(overrides.ContainerOverrides))
	}

	envVars := overrides.ContainerOverrides[0].Env
	if len(envVars) != 7 {
		t.Fatalf("expected 7 env vars, got %d", len(envVars))
	}

	// Verify all 7 env vars are set correctly
	expectedEnv := map[string]string{
		"AILANG_TASK_ID":   "task-12345678",
		"AILANG_AGENT_ID":  "design-doc-creator",
		"AILANG_WORKSPACE": "/workspace/ailang",
		"AILANG_PROVIDER":  "claude",
		"AILANG_DIRECTIVE": "Fix the parser bug",
		"AILANG_REPO_URL":  "https://github.com/sunholo-data/ailang",
		"AILANG_BRANCH":    "dev",
	}

	for _, env := range envVars {
		expected, ok := expectedEnv[env.Name]
		if !ok {
			t.Errorf("unexpected env var: %s", env.Name)
			continue
		}
		val, ok := env.Values.(*runpb.EnvVar_Value)
		if !ok {
			t.Errorf("env var %s: expected *EnvVar_Value, got %T", env.Name, env.Values)
			continue
		}
		if val.Value != expected {
			t.Errorf("env var %s = %q, want %q", env.Name, val.Value, expected)
		}
		delete(expectedEnv, env.Name)
	}

	if len(expectedEnv) > 0 {
		for name := range expectedEnv {
			t.Errorf("missing env var: %s", name)
		}
	}
}

func TestDispatchErrorPropagation(t *testing.T) {
	mock := &mockJobRunner{
		err: fmt.Errorf("permission denied: caller does not have run.jobs.run"),
	}
	d := newDispatcherWithClient(mock, "proj-1", "us-central1", "test")

	err := d.Dispatch(context.Background(), coordinator.DispatchParams{
		TaskID: "task-fail",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should wrap the original error
	if got := err.Error(); got != "failed to trigger Cloud Run Job projects/proj-1/locations/us-central1/jobs/test-agent-executor: permission denied: caller does not have run.jobs.run" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestDispatchPluginRepoEnvVar(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "proj-1", "us-central1", "test")

	params := coordinator.DispatchParams{
		TaskID:     "task-plugin-test",
		AgentID:    "website-builder",
		Workspace:  "sunholo-data/websites",
		Provider:   "claude",
		Directive:  "Build the website",
		RepoURL:    "https://github.com/sunholo-data/websites",
		Branch:     "main",
		PushBranch: "main",
		PluginRepo: "https://github.com/sunholo-data/ailang_bootstrap.git",
	}

	err := d.Dispatch(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envVars := mock.lastReq.Overrides.ContainerOverrides[0].Env
	// 7 base + PushBranch + PluginRepo = 9
	if len(envVars) != 9 {
		t.Fatalf("expected 9 env vars (7 base + PushBranch + PluginRepo), got %d", len(envVars))
	}

	// Verify PluginRepo is set
	found := false
	for _, env := range envVars {
		if env.Name == "AILANG_PLUGIN_REPO" {
			val, ok := env.Values.(*runpb.EnvVar_Value)
			if !ok {
				t.Fatalf("AILANG_PLUGIN_REPO: expected *EnvVar_Value, got %T", env.Values)
			}
			if val.Value != params.PluginRepo {
				t.Errorf("AILANG_PLUGIN_REPO = %q, want %q", val.Value, params.PluginRepo)
			}
			found = true
		}
	}
	if !found {
		t.Error("AILANG_PLUGIN_REPO env var not found in overrides")
	}
}

func TestDispatchWithoutPluginRepo(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "proj-1", "us-central1", "test")

	// No PluginRepo set — should NOT include AILANG_PLUGIN_REPO env var
	params := coordinator.DispatchParams{
		TaskID:  "task-no-plugin",
		AgentID: "sprint-executor",
	}

	err := d.Dispatch(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envVars := mock.lastReq.Overrides.ContainerOverrides[0].Env
	for _, env := range envVars {
		if env.Name == "AILANG_PLUGIN_REPO" {
			t.Error("AILANG_PLUGIN_REPO should not be set when PluginRepo is empty")
		}
	}
}

func TestDispatchAPIKeyMode(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "proj-1", "europe-west1", "ailang")

	params := coordinator.DispatchParams{
		TaskID:   "task-apikey-1",
		AgentID:  "external-agent",
		AuthMode: "apikey",
		APIKey:   "sk-ant-test-key-123",
	}

	err := d.Dispatch(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should select the apikey job template
	expectedJob := "projects/proj-1/locations/europe-west1/jobs/ailang-agent-executor-apikey"
	if mock.lastReq.Name != expectedJob {
		t.Errorf("job name = %q, want %q", mock.lastReq.Name, expectedJob)
	}

	// Should include AILANG_AUTH_MODE and ANTHROPIC_API_KEY in env overrides
	envVars := mock.lastReq.Overrides.ContainerOverrides[0].Env
	foundAuthMode := false
	foundAPIKey := false
	for _, env := range envVars {
		val, _ := env.Values.(*runpb.EnvVar_Value)
		switch env.Name {
		case "AILANG_AUTH_MODE":
			foundAuthMode = true
			if val.Value != "apikey" {
				t.Errorf("AILANG_AUTH_MODE = %q, want %q", val.Value, "apikey")
			}
		case "ANTHROPIC_API_KEY":
			foundAPIKey = true
			if val.Value != "sk-ant-test-key-123" {
				t.Errorf("ANTHROPIC_API_KEY = %q, want %q", val.Value, "sk-ant-test-key-123")
			}
		}
	}
	if !foundAuthMode {
		t.Error("AILANG_AUTH_MODE env var not found")
	}
	if !foundAPIKey {
		t.Error("ANTHROPIC_API_KEY env var not found")
	}
}

func TestDispatchOAuthModeDefault(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "proj-1", "europe-west1", "ailang")

	// Default (no AuthMode) should use oauth job template
	params := coordinator.DispatchParams{
		TaskID:  "task-oauth-1",
		AgentID: "internal-agent",
	}

	err := d.Dispatch(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedJob := "projects/proj-1/locations/europe-west1/jobs/ailang-agent-executor"
	if mock.lastReq.Name != expectedJob {
		t.Errorf("job name = %q, want %q", mock.lastReq.Name, expectedJob)
	}

	// Should NOT include AILANG_AUTH_MODE or ANTHROPIC_API_KEY
	envVars := mock.lastReq.Overrides.ContainerOverrides[0].Env
	for _, env := range envVars {
		if env.Name == "AILANG_AUTH_MODE" || env.Name == "ANTHROPIC_API_KEY" {
			t.Errorf("unexpected env var in oauth mode: %s", env.Name)
		}
	}
}

func TestDispatchDifferentRegions(t *testing.T) {
	tests := []struct {
		region   string
		prefix   string
		wantName string
	}{
		{"europe-west1", "ailang", "projects/p/locations/europe-west1/jobs/ailang-agent-executor"},
		{"us-central1", "prod", "projects/p/locations/us-central1/jobs/prod-agent-executor"},
		{"asia-east1", "staging", "projects/p/locations/asia-east1/jobs/staging-agent-executor"},
	}

	for _, tt := range tests {
		t.Run(tt.region+"/"+tt.prefix, func(t *testing.T) {
			mock := &mockJobRunner{}
			d := newDispatcherWithClient(mock, "p", tt.region, tt.prefix)

			err := d.Dispatch(context.Background(), coordinator.DispatchParams{TaskID: "task-1"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if mock.lastReq.Name != tt.wantName {
				t.Errorf("job name = %q, want %q", mock.lastReq.Name, tt.wantName)
			}
		})
	}
}
