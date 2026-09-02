package cloudrun

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

	// Assert the base set is PRESENT rather than counting it. A count breaks
	// every time a base var is legitimately added (it did, when
	// M-COORDINATOR-EXECUTION-TRUST M1a added AILANG_WORK_TIER) and says
	// nothing about which var is missing when it fails.
	for _, name := range []string{
		"AILANG_TASK_ID", "AILANG_AGENT_ID", "AILANG_WORKSPACE", "AILANG_PROVIDER",
		"AILANG_DIRECTIVE", "AILANG_REPO_URL", "AILANG_BRANCH", "AILANG_WORK_TIER",
		"AILANG_ACKNOWLEDGE_ONLY",
	} {
		if !hasEnvVar(envVars, name) {
			t.Errorf("base env var %s not set", name)
		}
	}

	// Verify the base env vars carry the right values
	expectedEnv := map[string]string{
		"AILANG_TASK_ID":   "task-12345678",
		"AILANG_AGENT_ID":  "design-doc-creator",
		"AILANG_WORKSPACE": "/workspace/ailang",
		"AILANG_PROVIDER":  "claude",
		"AILANG_DIRECTIVE": "Fix the parser bug",
		"AILANG_REPO_URL":  "https://github.com/sunholo-data/ailang",
		"AILANG_BRANCH":    "dev",
		// Unset on a bare DispatchParams — and empty is exactly what the gate
		// reads as tier 2. The fail-closed default is the assertion here.
		"AILANG_WORK_TIER": "",
		// false on a bare DispatchParams — the Go zero value IS the loud
		// direction, which is why the field is named acknowledge-only.
		"AILANG_ACKNOWLEDGE_ONLY": "false",
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
	// Presence, not arithmetic — see TestDispatchEnvVarOverrides.
	for _, name := range []string{"AILANG_PUSH_BRANCH", "AILANG_PLUGIN_REPO", "AILANG_WORK_TIER"} {
		if !hasEnvVar(envVars, name) {
			t.Errorf("env var %s not set", name)
		}
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

func TestJobSuffixForVariant(t *testing.T) {
	tests := []struct {
		variant  string
		authMode string
		want     string
		wantErr  bool
	}{
		{"", "oauth", "agent-executor", false},
		{"", "apikey", "agent-executor-apikey", false},
		{"default", "oauth", "agent-executor", false},
		{"default", "apikey", "agent-executor-apikey", false},
		{"go", "oauth", "agent-executor-go", false},
		{"go", "apikey", "agent-executor-go-apikey", false},
		{"gemini", "oauth", "agent-executor-gemini", false},
		{"gemini-go", "oauth", "agent-executor-gemini-go", false},
		{"codex", "oauth", "agent-executor-codex", false},
		{"codex-go", "oauth", "agent-executor-codex-go", false},
		{"opencode", "oauth", "agent-executor-opencode", false},
		{"eval", "oauth", "agent-executor-eval", false},
		{"eval-go", "oauth", "agent-executor-eval-go", false},
		{"unknown-xyz", "oauth", "", true},
		{"aider", "oauth", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.variant+"/"+tt.authMode, func(t *testing.T) {
			got, err := jobSuffixForVariant(tt.variant, tt.authMode)
			if tt.wantErr {
				if err == nil {
					t.Errorf("jobSuffixForVariant(%q, %q): expected error, got %q", tt.variant, tt.authMode, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("jobSuffixForVariant(%q, %q): unexpected error: %v", tt.variant, tt.authMode, err)
			}
			if got != tt.want {
				t.Errorf("jobSuffixForVariant(%q, %q) = %q, want %q", tt.variant, tt.authMode, got, tt.want)
			}
		})
	}
}

func TestDispatchGoVariantJobName(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "ailang-multivac-dev", "europe-west1", "ailang-dev")

	err := d.Dispatch(context.Background(), coordinator.DispatchParams{
		TaskID:          "task-sprint-1",
		AgentID:         "sprint-executor",
		ExecutorVariant: "go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "projects/ailang-multivac-dev/locations/europe-west1/jobs/ailang-dev-agent-executor-go"
	if mock.lastReq.Name != want {
		t.Errorf("job name = %q, want %q", mock.lastReq.Name, want)
	}
}

func TestDispatchUnknownVariantReturnsError(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "proj-1", "europe-west1", "ailang")

	err := d.Dispatch(context.Background(), coordinator.DispatchParams{
		TaskID:          "task-bad",
		ExecutorVariant: "not-a-real-variant",
	})
	if err == nil {
		t.Fatal("expected error for unknown variant, got nil")
	}
	if mock.lastReq != nil {
		t.Error("RunJob should not have been called for unknown variant")
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

// TestCheckVariantProviderAgreement pins the guard on the mismatch measured
// 2026-08-28: task-a0628a5f dispatched to ailang-agent-executor-codex, ran the
// opencode executor, and died on "executable file not found in $PATH" only
// after cloning 24,032 files.
func TestCheckVariantProviderAgreement(t *testing.T) {
	tests := []struct {
		name     string
		variant  string
		provider string
		wantErr  bool
	}{
		// The measured production failure.
		{"codex image running opencode -> REFUSE", "codex", "opencode", true},
		{"pi image running codex -> REFUSE", "pi", "codex", true},
		{"default image running pi -> REFUSE", "", "pi", true},
		{"go image running codex -> REFUSE", "go", "codex", true},

		// Correct pairings must dispatch.
		{"codex/codex", "codex", "codex", false},
		{"codex-go/codex", "codex-go", "codex", false},
		{"pi/pi", "pi", "pi", false},
		{"opencode/opencode", "opencode", "opencode", false},
		{"motoko/motoko", "motoko", "motoko", false},
		{"default/claude", "", "claude", false},
		{"default alias/claude", "default", "claude", false},
		{"go/claude", "go", "claude", false},

		// agent-eval installs every CLI — it must never be refused.
		{"eval image runs anything", "eval", "opencode", false},
		{"eval-go image runs anything", "eval-go", "pi", false},

		// managed_agents is a remote API and needs no binary on PATH.
		{"managed_agents in codex image", "codex", "managed_agents", false},
		{"managed_agents in pi image", "pi", "managed_agents", false},

		// An empty provider leaves the image default in charge.
		{"empty provider", "codex", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkVariantProviderAgreement(tc.variant, tc.provider)
			if (err != nil) != tc.wantErr {
				t.Fatalf("checkVariantProviderAgreement(%q,%q) err=%v, wantErr=%v",
					tc.variant, tc.provider, err, tc.wantErr)
			}
			if tc.wantErr && !strings.Contains(err.Error(), tc.provider) {
				t.Errorf("error must name the offending provider %q, got: %v", tc.provider, err)
			}
		})
	}
}

// TestProvidersForVariantCoversKnownVariants keeps the compatibility table and
// the accepted-variant set from drifting apart: a variant that dispatch accepts
// but the table has never heard of would silently skip the check.
func TestProvidersForVariantCoversKnownVariants(t *testing.T) {
	for variant := range knownVariants {
		if _, ok := providersForVariant[variant]; !ok {
			t.Errorf("variant %q is accepted by jobSuffixForVariant but absent from providersForVariant", variant)
		}
	}
}

// countingRunner records whether the Cloud Run API was ever reached.
type countingRunner struct{ calls int }

func (c *countingRunner) RunJob(_ context.Context, _ *runpb.RunJobRequest, _ ...gax.CallOption) (*run.RunJobOperation, error) {
	c.calls++
	return nil, errors.New("countingRunner: API reached")
}

// TestDispatchRefusesVariantProviderMismatch proves the guard is WIRED INTO
// Dispatch, not merely present in the package.
//
// This arm exists because the first version of these tests called
// checkVariantProviderAgreement directly, and a mutant that deleted the call
// from Dispatch left every one of them green — a guard nothing invokes is worth
// nothing. The assertion that matters is that the Cloud Run API is never
// reached: the whole point is to refuse BEFORE a container starts and clones
// 24,032 files to discover the same thing.
func TestDispatchRefusesVariantProviderMismatch(t *testing.T) {
	runner := &countingRunner{}
	d := newDispatcherWithClient(runner, "proj", "europe-west1", "ailang")

	err := d.Dispatch(context.Background(), coordinator.DispatchParams{
		TaskID:          "task-a0628a5f",
		AgentID:         "sprint-planner",
		ExecutorVariant: "codex",
		Provider:        "opencode", // the measured production mismatch
		AuthMode:        "oauth",
	})
	if err == nil {
		t.Fatal("Dispatch accepted a codex image running the opencode executor")
	}
	if !strings.Contains(err.Error(), "opencode") {
		t.Errorf("error should name the offending provider, got: %v", err)
	}
	if runner.calls != 0 {
		t.Errorf("Cloud Run API was called %d time(s); the mismatch must be refused before dispatch", runner.calls)
	}
}

// hasEnvVar reports whether a Cloud Run env var of the given name is present.
func hasEnvVar(envVars []*runpb.EnvVar, name string) bool {
	for _, env := range envVars {
		if env.Name == name {
			return true
		}
	}
	return false
}
