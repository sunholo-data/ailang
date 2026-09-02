package coordinator

import (
	"testing"
	"time"
)

// M-COORDINATOR-EXECUTION-TRUST M3.
//
// task-c429f9e0 died on "pi idle for 3m0.006592232s mid-generation (no output)"
// with no second attempt: ResolveModel returned chain[0] and discarded the rest
// ("retry chains are a driver concern" — written when the driver WAS the only
// dispatcher). The coordinator dispatches unattended, so it needs this more.

// MU-11: a transport stall is retryable; a model refusing is not. Retrying a
// refusal burns a second lane to get the same answer — and a bare "429"
// substring match would retry into a spent quota bucket.
func TestOnlyTransportFailuresAdvanceTheChain(t *testing.T) {
	transport := []string{
		"executor failed: pi task failed: pi idle for 3m0.006592232s mid-generation (no output)",
		"provider returned status 429: rate_limit_exceeded",
		"provider returned status 503: upstream unavailable",
		"context deadline exceeded while streaming completion",
		"model returned zero bytes",
	}
	for _, msg := range transport {
		if got := ClassifyFailure(msg); got != FailureTransport {
			t.Errorf("ClassifyFailure(%q) = %q, want %q", msg, got, FailureTransport)
		}
	}

	modelClass := []string{
		"model declined: I cannot help with that",
		"the generated code does not compile",
		"tests failed after 3 attempts",
		// The trap: "429" appears, but as data, not as a provider status.
		"benchmark run 429 of 500 produced a wrong answer",
		"",
	}
	for _, msg := range modelClass {
		if got := ClassifyFailure(msg); got == FailureTransport {
			t.Errorf("ClassifyFailure(%q) = %q — a model-class outcome must not be retried", msg, got)
		}
	}
}

// MU-13: the cap is hard.
func TestTwoExecutionCapIsHard(t *testing.T) {
	for attempts, want := range map[int]bool{0: true, 1: true, 2: false, 3: false, 99: false} {
		task := &TaskRecord{Status: TaskStatusRunning, AttemptCount: attempts}
		if got := ShouldReDispatch(task, time.Hour, true); got != want {
			t.Errorf("AttemptCount=%d: ShouldReDispatch=%v, want %v", attempts, got, want)
		}
	}
}

// MU-13b: e0b12bf5f made getTaskAge refuse to act on an unknowable age
// (time.Since(zero) is ~292 years, which marked tasks timed-out ~57s after
// dispatch and fed a 591-message amplification loop). This asserts the refusal
// still holds through the new re-dispatch path.
func TestUnknownAgeNeverReDispatches(t *testing.T) {
	task := &TaskRecord{Status: TaskStatusRunning, AttemptCount: 0}
	if ShouldReDispatch(task, 0, false) {
		t.Fatal("an unknowable task age must never trigger a re-dispatch")
	}
}

// A task that already reached a terminal state is over — the completion handler
// owns it, and re-dispatching would duplicate work.
func TestTerminalTaskIsNeverReDispatched(t *testing.T) {
	for _, s := range TerminalStatuses() {
		task := &TaskRecord{Status: s, AttemptCount: 0}
		if ShouldReDispatch(task, time.Hour, true) {
			t.Errorf("a %q task must not be re-dispatched", s)
		}
	}
}

// MU-13d / V22: no attempt counter existed on the task, so the cap had nowhere
// to live. It must be a persisted field, not in-memory state that a
// scale-to-zero forgets.
func TestCapSurvivesCoordinatorRestart(t *testing.T) {
	task := &TaskRecord{Status: TaskStatusRunning, AttemptCount: 1, ChainLinkIndex: 1}
	// Simulate a restart: the record is all that survives.
	reloaded := &TaskRecord{
		Status:         task.Status,
		AttemptCount:   task.AttemptCount,
		ChainLinkIndex: task.ChainLinkIndex,
	}
	if !ShouldReDispatch(reloaded, time.Hour, true) {
		t.Fatal("attempt 1 should still be re-dispatchable after a restart")
	}
	reloaded.AttemptCount = 2
	if ShouldReDispatch(reloaded, time.Hour, true) {
		t.Fatal("the cap must be readable from the persisted record alone")
	}
}

// MU-10: the whole chain, not chain[0].
func TestResolveModelReturnsWholeChain(t *testing.T) {
	chain, err := ResolveModelChain(&AgentConfig{ID: "a", Role: "executor"})
	if err != nil {
		t.Fatalf("ResolveModelChain: %v", err)
	}
	if len(chain) < 2 {
		t.Fatalf("the executor role declares a multi-entry chain; got %d: %v", len(chain), chain)
	}
	// An explicit pin is a deliberate operator choice and stays a chain of one.
	pinned, err := ResolveModelChain(&AgentConfig{ID: "b", Model: "some/explicit-model", Role: "executor"})
	if err != nil {
		t.Fatalf("ResolveModelChain(pinned): %v", err)
	}
	if len(pinned) != 1 || pinned[0] != "some/explicit-model" {
		t.Fatalf("an explicit pin must win and stand alone, got %v", pinned)
	}
}

// The first link of the chain must remain byte-identical to what ResolveModel
// returns, or M3 silently re-routes every existing agent.
func TestChainHeadMatchesResolveModel(t *testing.T) {
	agent := &AgentConfig{ID: "a", Role: "executor"}
	single, err := ResolveModel(agent)
	if err != nil {
		t.Fatalf("ResolveModel: %v", err)
	}
	chain, err := ResolveModelChain(agent)
	if err != nil {
		t.Fatalf("ResolveModelChain: %v", err)
	}
	if chain[0] != single {
		t.Fatalf("chain head %q must equal ResolveModel %q", chain[0], single)
	}
}
