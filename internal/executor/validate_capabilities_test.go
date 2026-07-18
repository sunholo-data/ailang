package executor

import (
	"context"
	"strings"
	"testing"
)

// capExecutor is a fake executor whose advertised capabilities are
// configurable and which records whether any dispatch method was invoked.
// It lets us assert the pre-dispatch gate fails BEFORE any Execute call.
type capExecutor struct {
	name     string
	caps     []Capability
	executed bool
}

func (c *capExecutor) Name() string { return c.name }
func (c *capExecutor) Execute(ctx context.Context, task *Task) (*Result, error) {
	c.executed = true
	return &Result{Success: true}, nil
}
func (c *capExecutor) ExecuteStreaming(ctx context.Context, task *Task, handler EventHandler) (*Result, error) {
	c.executed = true
	return &Result{Success: true}, nil
}
func (c *capExecutor) Capabilities() []Capability            { return c.caps }
func (c *capExecutor) CostModel() *CostModel                 { return &CostModel{ProviderName: c.name} }
func (c *capExecutor) HealthCheck(ctx context.Context) error { return nil }
func (c *capExecutor) Close() error                          { return nil }

func TestValidateTaskCapabilities_EgressOnNonCapableExecutor_LoudReject(t *testing.T) {
	exec := &capExecutor{name: "claude", caps: []Capability{CapStreaming, CapLocalWorkspace}}
	task := &Task{ID: "t1", RequiresEgress: true}

	err := ValidateTaskCapabilities(task, exec)
	if err == nil {
		t.Fatal("expected loud error when RequiresEgress set on executor lacking CapNetworkEgress, got nil")
	}
	// Error must be specific and actionable.
	for _, want := range []string{"egress", string(CapNetworkEgress), exec.Name()} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing expected substring %q", err.Error(), want)
		}
	}
	// The gate is PRE-dispatch: no Execute/ExecuteStreaming may have run.
	if exec.executed {
		t.Error("executor was dispatched despite failing the capability gate")
	}
}

func TestValidateTaskCapabilities_EgressOnCapableExecutor_OK(t *testing.T) {
	exec := &capExecutor{name: "managed_agents", caps: []Capability{CapRemoteSandbox, CapNetworkEgress}}
	task := &Task{ID: "t2", RequiresEgress: true}

	if err := ValidateTaskCapabilities(task, exec); err != nil {
		t.Fatalf("expected nil error for egress-capable executor, got %v", err)
	}
}

func TestValidateTaskCapabilities_NoEgress_NoOpOnAnyExecutor(t *testing.T) {
	// A non-capability executor must pass when RequiresEgress is false.
	exec := &capExecutor{name: "claude", caps: []Capability{CapStreaming}}
	task := &Task{ID: "t3", RequiresEgress: false}

	if err := ValidateTaskCapabilities(task, exec); err != nil {
		t.Fatalf("expected no-op nil error when RequiresEgress is false, got %v", err)
	}
	if exec.executed {
		t.Error("validation must not dispatch the executor")
	}
}

func TestValidateTaskCapabilities_NilInputs(t *testing.T) {
	if err := ValidateTaskCapabilities(nil, &capExecutor{}); err != nil {
		t.Errorf("nil task should be a no-op, got %v", err)
	}
	if err := ValidateTaskCapabilities(&Task{RequiresEgress: true}, nil); err != nil {
		t.Errorf("nil executor should be a no-op, got %v", err)
	}
}
