package messaging

import (
	"errors"
	"os/exec"
	"testing"
	"time"
)

// M-MISSION-COMMS-P1 / M1 — closes quorum objection P-1 (design doc V18).
//
// GitHubClient shells out to `gh`. Until this change `defaultExecCommand` used
// exec.Command with no context, so a hung or wedged `gh` blocked the caller
// forever. That matters because the mission driver is being ported onto this
// client: today 5 of its 6 `gh issue comment` call sites are unbounded shell-outs
// and exactly one is wrapped in `_mc_bounded 30`. Porting to Go without a deadline
// would have carried the defect across instead of fixing it.
//
// The contract these tests pin:
//   1. a call that outlives the deadline returns promptly, not when the child does;
//   2. it returns a TYPED error (ErrExecTimeout) so a caller can tell "GitHub is
//      slow" from "gh is broken" — Critical Principle 2, no silent fallbacks;
//   3. the default deadline is 30s, matching the driver's existing _mc_bounded 30;
//   4. the happy path is unchanged.

func TestExecTimeout_ReturnsPromptlyWithTypedError(t *testing.T) {
	cfg := &GitHubConfig{ExecTimeout: 50 * time.Millisecond}
	c := NewGitHubClient(cfg)

	// A real child process that outlives the deadline. Using a real exec rather
	// than a sleeping stub is deliberate: a stub would prove the wrapper returns,
	// not that the deadline actually reaches the process.
	start := time.Now()
	_, err := c.execCommandCtx("sleep", "10")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected a timeout error, got nil — the deadline did not fire")
	}
	if !errors.Is(err, ErrExecTimeout) {
		t.Fatalf("expected ErrExecTimeout, got %T: %v", err, err)
	}
	// Generous bound: asserting promptness, not benchmarking. An unbounded
	// implementation takes ~10s here and fails this outright.
	if elapsed > 5*time.Second {
		t.Fatalf("took %v — the call was not bounded by the %v deadline", elapsed, cfg.ExecTimeout)
	}
}

func TestExecTimeout_HappyPathUnchanged(t *testing.T) {
	c := NewGitHubClient(&GitHubConfig{ExecTimeout: 10 * time.Second})
	out, err := c.execCommandCtx("echo", "ok")
	if err != nil {
		t.Fatalf("echo should succeed, got %v", err)
	}
	if got := string(out); got != "ok\n" {
		t.Fatalf("output = %q, want %q", got, "ok\n")
	}
}

func TestExecTimeout_DefaultIsThirtySeconds(t *testing.T) {
	// Matches the driver's `_mc_bounded 30`. If someone changes the default,
	// they must change it here too and say why in the diff.
	if got := (&GitHubClient{}).execTimeout(); got != defaultExecTimeout {
		t.Fatalf("zero-value client timeout = %v, want %v", got, defaultExecTimeout)
	}
	if defaultExecTimeout != 30*time.Second {
		t.Fatalf("defaultExecTimeout = %v, want 30s to match _mc_bounded 30", defaultExecTimeout)
	}
	// A nil config must not panic and must still yield the default.
	if got := NewGitHubClient(nil).execTimeout(); got != defaultExecTimeout {
		t.Fatalf("nil-config timeout = %v, want %v", got, defaultExecTimeout)
	}
}

func TestExecTimeout_NonTimeoutFailureIsNotMislabelled(t *testing.T) {
	// A command that fails for its own reasons must NOT come back as a timeout,
	// or the typed error is useless for telling the two apart.
	c := NewGitHubClient(&GitHubConfig{ExecTimeout: 10 * time.Second})
	_, err := c.execCommandCtx("false")
	if err == nil {
		t.Fatal("expected `false` to fail")
	}
	if errors.Is(err, ErrExecTimeout) {
		t.Fatalf("a plain non-zero exit was mislabelled as a timeout: %v", err)
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected an *exec.ExitError, got %T: %v", err, err)
	}
}

func TestExecTimeout_InjectedStubStillWins(t *testing.T) {
	// The execCommand field is the existing test seam used across this package.
	// Bounding must not break it, or every existing GitHubClient test breaks.
	called := false
	c := NewGitHubClient(nil)
	c.execCommand = func(name string, arg ...string) ([]byte, error) {
		called = true
		return []byte("stubbed"), nil
	}
	out, err := c.execCommandCtx("gh", "issue", "list")
	if err != nil {
		t.Fatalf("stub path returned error: %v", err)
	}
	if !called {
		t.Fatal("injected execCommand stub was bypassed — existing tests would break")
	}
	if string(out) != "stubbed" {
		t.Fatalf("output = %q, want %q", string(out), "stubbed")
	}
}
