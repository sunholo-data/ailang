package pi

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

func TestNormalizePiFinishReason(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		// Observed in captured fixtures (pi 0.70.2).
		{"stop", executor.FinishStop},
		{"toolUse", executor.FinishToolCalls},
		// Plausible synonyms, mapped defensively.
		{"endTurn", executor.FinishStop},
		{"maxTokens", executor.FinishLength},
		{"refusal", executor.FinishContentFilter},
		{"aborted", executor.FinishError},
		// Unknown values pass through verbatim rather than being coerced to
		// "stop" — CategorizeAgentError ignores what it doesn't recognize, so
		// pass-through stays visible in banked JSON without misclassifying.
		{"someFuturePiReason", "someFuturePiReason"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := normalizePiFinishReason(tt.raw); got != tt.want {
			t.Errorf("normalizePiFinishReason(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

// runPiFixture executes a fixture through the fake-pi harness and returns the
// Result, so finish-reason assertions read as one line each.
func runPiFixture(t *testing.T, fixture string) *executor.Result {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	dir := t.TempDir()
	_ = writeFakePi(t, dir, loadFixtureLines(t, fixture))

	e, err := New(&executor.Config{
		PiPath:         filepath.Join(dir, "pi"),
		PiModel:        "anthropic/claude-haiku-4-5",
		TimeoutSeconds: 10,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := e.ExecuteStreaming(context.Background(), &executor.Task{
		ID:        "test-finish-reason",
		Directive: "test",
		Workspace: dir,
		Timeout:   10 * time.Second,
	}, &collectingHandler{})
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}
	return result
}

func TestExecuteStreaming_FinishReasonFromStopReason(t *testing.T) {
	if got := runPiFixture(t, "fizzbuzz.ndjson").FinishReason; got != executor.FinishStop {
		t.Errorf("FinishReason = %q, want %q", got, executor.FinishStop)
	}
}

// The tool_use fixture has an INTERMEDIATE message_end/turn_end with
// stopReason "toolUse" before the run's final "stop". Taking the last settled
// value (not the first, and not any streaming message_update partial) is what
// keeps a normal tool-using run from banking as "tool_calls".
func TestExecuteStreaming_FinishReasonIgnoresIntermediateToolUse(t *testing.T) {
	if got := runPiFixture(t, "tool_use.ndjson").FinishReason; got != executor.FinishStop {
		t.Errorf("FinishReason = %q, want %q (intermediate toolUse must not win)", got, executor.FinishStop)
	}
}

func TestPiCancelFinishReason(t *testing.T) {
	if got := piCancelFinishReason(context.DeadlineExceeded); got != executor.FinishTimeout {
		t.Errorf("DeadlineExceeded → %q, want %q", got, executor.FinishTimeout)
	}
	if got := piCancelFinishReason(context.Canceled); got != executor.FinishError {
		t.Errorf("Canceled → %q, want %q", got, executor.FinishError)
	}
}
