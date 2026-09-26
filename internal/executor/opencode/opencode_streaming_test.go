package opencode

// M-MODEL-REGISTRY-SINGLE-SOURCE M6: split out of opencode_test.go, which crossed
// the 800-line gate when D2(a)'s explicit model pins were added. A cohesive block
// (the streaming-parse cases) moved wholesale rather than shaving lines off the
// original, per .claude/rules/coding-standards.md.

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

func TestExecuteStreaming_CapturesReasoningAndTruncation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	dir := t.TempDir()
	events := []string{
		`{"type":"step_start","timestamp":1,"sessionID":"ses_r","part":{"id":"p1","messageID":"m1","sessionID":"ses_r","type":"step-start"}}`,
		`{"type":"step_finish","timestamp":2,"sessionID":"ses_r","part":{"id":"p2","reason":"tool-calls","messageID":"m1","sessionID":"ses_r","type":"step-finish","tokens":{"total":900,"input":10,"output":40,"reasoning":850,"cache":{"write":0,"read":0}},"cost":0.001}}`,
		`{"type":"step_start","timestamp":3,"sessionID":"ses_r","part":{"id":"p3","messageID":"m2","sessionID":"ses_r","type":"step-start"}}`,
		`{"type":"step_finish","timestamp":4,"sessionID":"ses_r","part":{"id":"p4","reason":"length","messageID":"m2","sessionID":"ses_r","type":"step-finish","tokens":{"total":700,"input":5,"output":15,"reasoning":680,"cache":{"write":0,"read":0}},"cost":0.002}}`,
	}
	_ = writeFakeOpenCode(t, dir, events)

	e, err := New(&executor.Config{
		OpenCodePath:   filepath.Join(dir, "opencode"),
		OpenCodeModel:  "anthropic/claude-haiku-4-5",
		TimeoutSeconds: 10,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	result, err := e.ExecuteStreaming(context.Background(), &executor.Task{
		ID:        "test-reasoning",
		Directive: "think hard",
		Workspace: dir,
		Timeout:   10 * time.Second,
	}, &collectingHandler{})
	if err != nil {
		t.Fatalf("ExecuteStreaming error: %v", err)
	}

	// Summed across step_finish deltas: 850 + 680.
	if result.ReasonTokens != 1530 {
		t.Errorf("ReasonTokens = %d, want 1530 (850+680 summed across steps)", result.ReasonTokens)
	}

	// Reasoning must stay DISJOINT from OutputTokens — the harness adds both
	// into TotalTokens, so folding them together would double-count.
	if result.OutputTokens != 55 {
		t.Errorf("OutputTokens = %d, want 55 (40+15, reasoning excluded)", result.OutputTokens)
	}

	if result.FinishReason != "length" {
		t.Errorf("FinishReason = %q, want \"length\" (truncated at the output cap)", result.FinishReason)
	}
}

func TestNormalizeOpencodeFinishReason(t *testing.T) {
	cases := map[string]string{
		"":               "",
		"stop":           "stop",
		"length":         "length",
		"tool-calls":     "tool_calls",
		"content-filter": "content_filter",
		"error":          "error",
		"OTHER":          "other",
	}
	for in, want := range cases {
		if got := normalizeOpencodeFinishReason(in); got != want {
			t.Errorf("normalizeOpencodeFinishReason(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExecuteStreaming_NonJSONPreambleTolerated(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	// Non-JSON lines (startup messages) must be skipped without failing.
	events := append(
		[]string{
			"Performing one time database migration, may take a few minutes...",
			"sqlite-migration:done",
			"Database migration complete.",
		},
		mockEvents()...,
	)
	dir := t.TempDir()
	_ = writeFakeOpenCode(t, dir, events)

	cfg := &executor.Config{
		OpenCodePath:   filepath.Join(dir, "opencode"),
		TimeoutSeconds: 10,
		OpenCodeModel:  "anthropic/claude-haiku-4-5", // D2(a): model is required
	}
	e, _ := New(cfg)
	task := &executor.Task{
		Directive: "test",
		Workspace: dir,
		Timeout:   10 * time.Second,
	}
	result, err := e.ExecuteStreaming(context.Background(), task, &executor.NoOpEventHandler{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.InputTokens == 0 {
		t.Error("expected tokens parsed despite preamble noise")
	}
}

func TestExecuteStreaming_BinaryNotFound(t *testing.T) {
	cfg := &executor.Config{
		OpenCodePath:   "/nonexistent/opencode-bin",
		TimeoutSeconds: 5,
		OpenCodeModel:  "anthropic/claude-haiku-4-5", // D2(a): model is required
	}
	e, _ := New(cfg)
	task := &executor.Task{
		Directive: "test",
		Workspace: t.TempDir(),
		Timeout:   5 * time.Second,
	}
	_, err := e.ExecuteStreaming(context.Background(), task, &executor.NoOpEventHandler{})
	if err == nil {
		t.Error("expected error when binary not found")
	}
}

func TestHealthCheck_MissingBinary(t *testing.T) {
	cfg := &executor.Config{OpenCodePath: "/nonexistent/opencode-does-not-exist", OpenCodeModel: "anthropic/claude-haiku-4-5"} // D2(a): model required; this test is about the BINARY
	e, _ := New(cfg)
	ctx := context.Background()
	if err := e.HealthCheck(ctx); err == nil {
		t.Error("expected HealthCheck to fail for missing binary")
	}
}

func TestHealthCheck_WithFakeBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	dir := t.TempDir()
	// writeFakeOpenCode handles --version → prints version and exits 0
	_ = writeFakeOpenCode(t, dir, nil)
	cfg := &executor.Config{
		OpenCodePath:   filepath.Join(dir, "opencode"),
		TimeoutSeconds: 5,
		OpenCodeModel:  "anthropic/claude-haiku-4-5", // D2(a): model is required
	}
	e, _ := New(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.HealthCheck(ctx); err != nil {
		t.Errorf("expected HealthCheck to succeed with fake binary, got: %v", err)
	}
}

// TestInit_RegistersOpencode verifies that importing this package (via init())
// registers "opencode" in the global executor factory.
func TestInit_RegistersOpencode(t *testing.T) {
	factory := executor.GlobalFactory()
	available := factory.ListAvailable()

	found := false
	for _, name := range available {
		if name == "opencode" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("init() did not register 'opencode'; factory.ListAvailable() = %v", available)
	}

	// M-MODEL-REGISTRY-SINGLE-SOURCE M6 (D2(a)): the factory still BUILDS the
	// executor with no model configured — construction is not where the check
	// lives, because the coordinator builds an executor before it knows the
	// task and supplies Task.Model per task. The fail-loud is at execution
	// entry; see TestNew ...EmptyConfig/EmptyModel FailsLoudly in this package.
	exec, err := factory.GetExecutor("opencode")
	if err != nil {
		t.Fatalf("factory.GetExecutor(%q) failed: %v", "opencode", err)
	}
	if exec.Name() != "opencode" {
		t.Errorf("built executor has wrong name: %q", exec.Name())
	}
}

// TestRegister_Idempotent verifies double-registration does not panic.
func TestRegister_Idempotent(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Register() panicked on double registration: %v", r)
		}
	}()
	Register()
	Register()
}

// TestExecuteStreaming_ParsesFixture_FromFile replays the recorded live fixture
// from testdata/ and asserts the same token/cost invariants.
