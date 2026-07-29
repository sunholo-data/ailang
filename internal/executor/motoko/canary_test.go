package motoko

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// TestMotokoImplementsCanaryChecker is the wiring guard: if this type ever
// stops satisfying the optional interface, RunCanary silently degrades to a
// no-op and the gate quietly stops protecting anything. That silent-downgrade
// is exactly the failure class this sprint exists to kill, so assert it.
func TestMotokoImplementsCanaryChecker(t *testing.T) {
	var e any = &MotokoExecutor{}
	if _, ok := e.(executor.CanaryChecker); !ok {
		t.Fatal("*MotokoExecutor must implement executor.CanaryChecker, otherwise RunCanary no-ops and the gate is dead")
	}
}

func TestEvaluateCanaryResult(t *testing.T) {
	tests := []struct {
		name    string
		result  *executor.Result
		err     error
		wantErr string // substring; "" means expect success
	}{
		{
			name:    "execute error propagates",
			err:     errors.New("effect checking failed in src/core/tool_runtime: Missing effects: Env"),
			wantErr: "Missing effects: Env",
		},
		{
			name:    "nil result is a failure, not a pass",
			result:  nil,
			wantErr: "no result",
		},
		{
			// The July 2026 outage signature: the process comes back, but it
			// never got far enough to do anything.
			name:    "zero turns means the subject never ran a step",
			result:  &executor.Result{NumTurns: 0, ToolCalls: map[string]int{}},
			wantErr: "no steps",
		},
		{
			// A subject can stream text and still be unable to act. Tool
			// dispatch is the property benchmarks actually depend on.
			name:    "turns but no tool call is still a failure",
			result:  &executor.Result{NumTurns: 3, ToolCalls: map[string]int{}},
			wantErr: "no tool calls",
		},
		{
			name:   "turns plus a tool call passes",
			result: &executor.Result{NumTurns: 2, ToolCalls: map[string]int{"ReadFile": 1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := evaluateCanaryResult(tt.result, tt.err)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected pass, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected failure containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestCanaryTaskIsTrivial guards R1/R4: the canary must stay the simplest
// possible task. If it grows into something a healthy-but-slow model can fail,
// it starts skipping good models — worse than not having a gate.
func TestCanaryTaskIsTrivial(t *testing.T) {
	task := newCanaryTask("/tmp/canary-workspace", "some-model")

	if task.Workspace != "/tmp/canary-workspace" {
		t.Errorf("workspace = %q, want the passed workspace", task.Workspace)
	}
	if task.Model != "some-model" {
		t.Errorf("model = %q, want the passed model", task.Model)
	}
	if task.Timeout <= 0 {
		t.Error("canary MUST bound its own runtime — an unbounded canary can hang the whole suite")
	}
	if task.Timeout > canaryMaxTimeout {
		t.Errorf("timeout %v exceeds the %v ceiling; a slow canary defeats its purpose", task.Timeout, canaryMaxTimeout)
	}
	if task.Directive == "" {
		t.Error("canary needs a directive")
	}

	// THE bug that made the canary condemn healthy subjects: Task.TTFTTimeout
	// defaults to 30s, and a cold local 35B needs longer than that just to emit
	// its first event. The executor killed motoko before anything was parsed
	// and reported "ran no steps" — while the abandoned process went on to
	// complete the task correctly.
	if task.TTFTTimeout < time.Minute {
		t.Errorf("TTFTTimeout = %v; must be generous enough for a COLD local model to reach first token, or the gate kills healthy subjects", task.TTFTTimeout)
	}
	if task.IdleTimeout < time.Minute {
		t.Errorf("IdleTimeout = %v; too tight for a slow local model mid-run", task.IdleTimeout)
	}
}

// TestCanaryTaskIDsAreUnique guards the second bug found in the same run:
// motoko names its session log after the task ID, so a constant "canary" made
// every attempt APPEND to one shared session_canary.jsonl — four runs deep by
// the time it was noticed. Repeated or concurrent canaries would parse each
// other's events.
func TestCanaryTaskIDsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		id := newCanaryTask("/tmp/w", "m").ID
		if seen[id] {
			t.Fatalf("duplicate canary task ID %q — session logs would collide", id)
		}
		seen[id] = true
	}
}

// TestGetModel_NilTaskDoesNotPanic guards a real regression: the canary called
// getModel(nil), which dereferenced the task and segfaulted the WHOLE eval
// suite the first time a canary ran against a real executor. The unit tests
// missed it because they exercised newCanaryTask in isolation and never reached
// the Execute path — a gap the measurement contract itself then caught, by
// making the first real fmt A/B run fail loudly instead of silently.
func TestGetModel_NilTaskDoesNotPanic(t *testing.T) {
	e := &MotokoExecutor{model: "fallback-model"}

	if got := e.getModel(nil); got != "fallback-model" {
		t.Errorf("getModel(nil) = %q, want the executor default", got)
	}
	if got := e.getModel(&executor.Task{}); got != "fallback-model" {
		t.Errorf("getModel(empty task) = %q, want the executor default", got)
	}
	if got := e.getModel(&executor.Task{Model: "override"}); got != "override" {
		t.Errorf("getModel(task with model) = %q, want the override", got)
	}
}
