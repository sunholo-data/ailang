package executor

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// canaryStub implements only CanaryChecker (plus the Executor bits RunCanary
// needs) so we can drive RunCanary without standing up a real executor.
type canaryStub struct {
	Executor // embedded nil interface — RunCanary must not touch anything but Name/CanaryCheck
	name     string
	err      error
	calls    *int
}

func (c *canaryStub) Name() string { return c.name }
func (c *canaryStub) CanaryCheck(ctx context.Context) error {
	if c.calls != nil {
		*c.calls++
	}
	return c.err
}

// noCanaryStub implements Executor's Name only and deliberately does NOT
// implement CanaryChecker.
type noCanaryStub struct {
	Executor
	name string
}

func (n *noCanaryStub) Name() string { return n.name }

// TestRunCanary_NotImplemented_IsNoOp is the back-compat guarantee: the five
// executors that have not opted in (claude, codex, pi, opencode,
// managed_agents) must be completely unaffected by the gate.
func TestRunCanary_NotImplemented_IsNoOp(t *testing.T) {
	if err := RunCanary(context.Background(), &noCanaryStub{name: "claude"}); err != nil {
		t.Fatalf("executor without CanaryCheck must pass by default, got: %v", err)
	}
}

func TestRunCanary_Pass(t *testing.T) {
	calls := 0
	if err := RunCanary(context.Background(), &canaryStub{name: "motoko", err: nil, calls: &calls}); err != nil {
		t.Fatalf("healthy canary should pass, got: %v", err)
	}
	if calls != 1 {
		t.Errorf("canary must run exactly once, ran %d times", calls)
	}
}

// TestRunCanary_Fail_WrapsAsCanaryError ensures callers can distinguish a
// canary failure from any other error, so the suite can attribute the skip
// reason as canary_failed rather than a generic harness error.
func TestRunCanary_Fail_WrapsAsCanaryError(t *testing.T) {
	underlying := errors.New("effect checking failed in src/core/tool_runtime: Missing effects: Env")
	err := RunCanary(context.Background(), &canaryStub{name: "motoko", err: underlying})
	if err == nil {
		t.Fatal("failing canary must return an error")
	}

	var ce *CanaryError
	if !errors.As(err, &ce) {
		t.Fatalf("error must be a *CanaryError so callers can attribute the skip, got %T", err)
	}
	if ce.Executor != "motoko" {
		t.Errorf("CanaryError.Executor = %q, want %q", ce.Executor, "motoko")
	}
	if !errors.Is(err, underlying) {
		t.Error("CanaryError must wrap the underlying cause (errors.Is)")
	}
	// The operator has to be able to see WHY from the message alone.
	if !strings.Contains(err.Error(), "Missing effects: Env") {
		t.Errorf("CanaryError message must carry the underlying detail, got: %s", err.Error())
	}
}

// TestRunCanary_RespectsContextCancellation guards R4: a cancelled/timed-out
// context must surface rather than hang the suite.
func TestRunCanary_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := RunCanary(ctx, &canaryStub{name: "motoko", err: ctx.Err()})
	if err == nil {
		t.Fatal("cancelled context must produce an error")
	}
}
