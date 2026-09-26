package cloudrun

import (
	"context"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
)

// The bug was never in the idle timeout itself — it was that the value stopped
// at the dispatcher. The agent registry declared it, `coordinator agents <id>`
// printed it as EFFECTIVE, and the Cloud Run job never received it, so all 41
// agents declaring 5m/6m/10m ran on the executor's hardcoded 3m. Assert on the
// env the job will actually see, which is the side the old code got wrong.
func envValue(t *testing.T, mock *mockJobRunner, name string) (string, bool) {
	t.Helper()
	if mock.lastReq == nil || mock.lastReq.Overrides == nil || len(mock.lastReq.Overrides.ContainerOverrides) == 0 {
		t.Fatal("no container overrides recorded")
	}
	for _, e := range mock.lastReq.Overrides.ContainerOverrides[0].Env {
		if e.Name == name {
			return e.GetValue(), true
		}
	}
	return "", false
}

func baseParams() coordinator.DispatchParams {
	return coordinator.DispatchParams{
		TaskID:    "task-12345678",
		AgentID:   "sprint-evaluator",
		Workspace: "/workspace/ailang",
		Provider:  "claude",
		Directive: "Evaluate the sprint",
		RepoURL:   "https://github.com/sunholo-data/ailang",
		Branch:    "dev",
	}
}

func TestDispatch_IdleTimeoutReachesTheJob(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "proj-1", "europe-west1", "ailang")

	params := baseParams()
	params.IdleTimeout = "5m" // what sprint-evaluator declares

	if err := d.Dispatch(context.Background(), params); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	got, ok := envValue(t, mock, config.EnvIdleTimeout)
	if !ok {
		t.Fatalf("%s absent from the job env — the declared idle_timeout does not travel, so the job runs at the hardcoded 3m", config.EnvIdleTimeout)
	}
	if got != "5m" {
		t.Errorf("%s = %q, want %q", config.EnvIdleTimeout, got, "5m")
	}
}

// Control: an agent that declares nothing must not have one invented for it —
// the job's own default has to stay reachable.
func TestDispatch_NoIdleTimeoutDeclaredSendsNoOverride(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "proj-1", "europe-west1", "ailang")

	if err := d.Dispatch(context.Background(), baseParams()); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if got, ok := envValue(t, mock, config.EnvIdleTimeout); ok {
		t.Errorf("%s = %q, want absent when the agent declares none", config.EnvIdleTimeout, got)
	}
}

// The two timeouts are different quantities and must not be conflated: the
// ceiling was raised to 2h precisely BECAUSE idle_timeout was believed to be
// the liveness guard.
func TestDispatch_IdleTimeoutIsSeparateFromTheWallClock(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "proj-1", "europe-west1", "ailang")

	params := baseParams()
	params.Timeout = "2h"
	params.IdleTimeout = "10m"

	if err := d.Dispatch(context.Background(), params); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	wall, _ := envValue(t, mock, "AILANG_TIMEOUT")
	idle, _ := envValue(t, mock, config.EnvIdleTimeout)
	if wall != "2h" {
		t.Errorf("AILANG_TIMEOUT = %q, want 2h", wall)
	}
	if idle != "10m" {
		t.Errorf("%s = %q, want 10m", config.EnvIdleTimeout, idle)
	}
}
