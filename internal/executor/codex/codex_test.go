package codex

import (
	"bufio"
	"context"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

func TestNewCodexExecutor(t *testing.T) {
	cfg := executor.DefaultConfig()
	exec, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if exec.Name() != "codex" {
		t.Errorf("expected name 'codex', got %q", exec.Name())
	}
	if exec.codexPath != "codex" {
		t.Errorf("expected default codexPath 'codex', got %q", exec.codexPath)
	}
	if exec.model != "gpt-5-codex" {
		t.Errorf("expected default model 'gpt-5-codex', got %q", exec.model)
	}
}

func TestNewCodexExecutor_EmptyConfigUsesFallbacks(t *testing.T) {
	// Empty config (no defaults) should still produce a working executor.
	exec, err := New(&executor.Config{})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if exec.codexPath != "codex" {
		t.Errorf("expected fallback path 'codex', got %q", exec.codexPath)
	}
	if exec.model != "gpt-5-codex" {
		t.Errorf("expected fallback model 'gpt-5-codex', got %q", exec.model)
	}
}

func TestNewCodexExecutor_CustomPathAndModel(t *testing.T) {
	cfg := &executor.Config{
		CodexPath:  "/custom/bin/codex",
		CodexModel: "gpt-5-codex-preview",
	}
	exec, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if exec.codexPath != "/custom/bin/codex" {
		t.Errorf("expected custom codexPath, got %q", exec.codexPath)
	}
	if exec.model != "gpt-5-codex-preview" {
		t.Errorf("expected custom model, got %q", exec.model)
	}
}

func TestCodexCapabilities(t *testing.T) {
	exec, _ := New(executor.DefaultConfig())
	caps := exec.Capabilities()
	if len(caps) == 0 {
		t.Fatal("expected at least one capability")
	}

	want := map[executor.Capability]bool{
		executor.CapStreaming:      false,
		executor.CapLocalWorkspace: false,
	}
	for _, c := range caps {
		if _, ok := want[c]; ok {
			want[c] = true
		}
	}
	for c, present := range want {
		if !present {
			t.Errorf("missing expected capability %q", c)
		}
	}
}

func TestCodexCostModel(t *testing.T) {
	exec, _ := New(executor.DefaultConfig())
	cm := exec.CostModel()

	if cm.ProviderName != "openai" {
		t.Errorf("expected provider 'openai', got %q", cm.ProviderName)
	}
	// gpt-5-codex: $1.25/$10.00 per 1M = $0.00125/$0.01 per 1K
	if cm.InputTokenCost != 0.00125 {
		t.Errorf("expected input cost 0.00125, got %v", cm.InputTokenCost)
	}
	if cm.OutputTokenCost != 0.01 {
		t.Errorf("expected output cost 0.01, got %v", cm.OutputTokenCost)
	}

	// Cost calculation sanity check: 1000 in + 500 out.
	got := cm.CalculateCost(executor.TokenUsage{InputTokens: 1000, OutputTokens: 500})
	want := 0.00125 + 0.005
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("expected cost %v, got %v", want, got)
	}
}

func TestCodexFactoryRegistration(t *testing.T) {
	available := executor.GlobalFactory().ListAvailable()
	for _, name := range available {
		if name == "codex" {
			return
		}
	}
	t.Errorf("expected 'codex' in factory.ListAvailable(), got %v", available)
}

// TestInit_RegistersCodex is the sprint M2 acceptance test: importing the codex
// package (via init()) must cause "codex" to appear in GlobalFactory().ListAvailable().
func TestInit_RegistersCodex(t *testing.T) {
	factory := executor.GlobalFactory()
	available := factory.ListAvailable()

	found := false
	for _, name := range available {
		if name == "codex" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("init() did not register 'codex'; factory.ListAvailable() = %v", available)
	}

	// Also verify the factory can actually build the executor.
	exec, err := factory.GetExecutor("codex")
	if err != nil {
		t.Fatalf("factory.GetExecutor(\"codex\") failed: %v", err)
	}
	if exec.Name() != "codex" {
		t.Errorf("built executor has wrong name: %q", exec.Name())
	}
}

func TestCodexClose(t *testing.T) {
	exec, _ := New(executor.DefaultConfig())
	if err := exec.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestGetModel_TaskOverride(t *testing.T) {
	exec, _ := New(executor.DefaultConfig())
	task := &executor.Task{Model: "override-model"}
	if got := exec.getModel(task); got != "override-model" {
		t.Errorf("expected task model override, got %q", got)
	}
	empty := &executor.Task{}
	if got := exec.getModel(empty); got != "gpt-5-codex" {
		t.Errorf("expected default model, got %q", got)
	}
}

func TestParseCodexEvent_Message(t *testing.T) {
	line := []byte(`{"type":"message","turn_number":1,"text":"hello","tokens_used":{"input":10,"output":5}}`)
	ev, err := parseCodexEvent(line)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if ev.Type != "message" {
		t.Errorf("expected type 'message', got %q", ev.Type)
	}
	if ev.TurnNumber != 1 {
		t.Errorf("expected turn_number 1, got %d", ev.TurnNumber)
	}
	if ev.Text != "hello" {
		t.Errorf("expected text 'hello', got %q", ev.Text)
	}
	if ev.Tokens.Input != 10 || ev.Tokens.Output != 5 {
		t.Errorf("expected tokens {10,5}, got %+v", ev.Tokens)
	}
	if ev.Raw == nil {
		t.Error("expected Raw map to be populated")
	}
}

func TestParseCodexEvent_Session(t *testing.T) {
	line := []byte(`{"type":"session","session_id":"codex-sess-abc123","model":"gpt-5-codex"}`)
	ev, err := parseCodexEvent(line)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if ev.Type != "session" {
		t.Errorf("expected type 'session', got %q", ev.Type)
	}
	if ev.SessionID != "codex-sess-abc123" {
		t.Errorf("expected session_id, got %q", ev.SessionID)
	}
	// Model is preserved in Raw even though the struct does not expose it.
	if m, ok := ev.Raw["model"]; !ok || m != "gpt-5-codex" {
		t.Errorf("expected model preserved in Raw, got %v", ev.Raw)
	}
}

func TestParseCodexEvent_ToolUse(t *testing.T) {
	line := []byte(`{"type":"tool_use","tool_name":"read","tool_id":"tool_1","parameters":{"path":"README.md"}}`)
	ev, err := parseCodexEvent(line)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if ev.Type != "tool_use" {
		t.Errorf("expected type 'tool_use', got %q", ev.Type)
	}
	if ev.ToolName != "read" {
		t.Errorf("expected tool_name 'read', got %q", ev.ToolName)
	}
	if !strings.Contains(string(ev.Parameters), `"path":"README.md"`) {
		t.Errorf("expected parameters JSON, got %s", string(ev.Parameters))
	}
}

func TestParseCodexEvent_ToolResult(t *testing.T) {
	line := []byte(`{"type":"tool_result","tool_name":"read","tool_id":"tool_1","output":"# readme"}`)
	ev, err := parseCodexEvent(line)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if ev.Type != "tool_result" {
		t.Errorf("expected type 'tool_result', got %q", ev.Type)
	}
	if ev.Output != "# readme" {
		t.Errorf("expected output '# readme', got %q", ev.Output)
	}
}

func TestParseCodexEvent_Result(t *testing.T) {
	line := []byte(`{"type":"result","tokens_used":{"input":160,"output":72},"status":"success"}`)
	ev, err := parseCodexEvent(line)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if ev.Type != "result" {
		t.Errorf("expected type 'result', got %q", ev.Type)
	}
	if ev.Tokens.Input != 160 || ev.Tokens.Output != 72 {
		t.Errorf("expected tokens {160,72}, got %+v", ev.Tokens)
	}
	if s, ok := ev.Raw["status"]; !ok || s != "success" {
		t.Errorf("expected status preserved in Raw, got %v", ev.Raw)
	}
}

func TestParseCodexEvent_SchemaDriftTolerance(t *testing.T) {
	// Unknown top-level fields should not fail parsing; they should be
	// preserved in Raw so downstream analysis can inspect them.
	line := []byte(`{"type":"message","text":"x","custom_field":"future_value","nested":{"a":1}}`)
	ev, err := parseCodexEvent(line)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if ev.Raw["custom_field"] != "future_value" {
		t.Errorf("expected custom_field preserved, got %v", ev.Raw["custom_field"])
	}
	if _, ok := ev.Raw["nested"]; !ok {
		t.Error("expected nested field preserved in Raw")
	}
}

func TestParseCodexEvent_NonJSONLine(t *testing.T) {
	cases := []string{
		"Creating GCP exporters...",
		"some preamble",
		"",
		"   ",
	}
	for _, line := range cases {
		if _, err := parseCodexEvent([]byte(line)); err == nil {
			t.Errorf("expected error for non-JSON line %q", line)
		}
	}
}

func TestParseCodexEvent_TruncatedJSON(t *testing.T) {
	// Partial JSON should fail with error, not panic.
	line := []byte(`{"type":"message","text":"hel`)
	_, err := parseCodexEvent(line)
	if err == nil {
		t.Error("expected error for truncated JSON")
	}
}

// TestParseCodexEvent_FromFixture validates parsing of the recorded Codex
// response fixture. Every non-empty line must be parseable and event counts
// should match the documented schema shape.
func TestParseCodexEvent_FromFixture(t *testing.T) {
	f, err := os.Open("testdata/codex_response.jsonl")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	var messages, toolUse, toolResult, result, session int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		ev, err := parseCodexEvent([]byte(line))
		if err != nil {
			t.Errorf("parse failed on line %q: %v", line, err)
			continue
		}
		switch ev.Type {
		case "message":
			messages++
		case "tool_use":
			toolUse++
		case "tool_result":
			toolResult++
		case "result":
			result++
		case "session":
			session++
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}

	if session != 1 {
		t.Errorf("expected 1 session event, got %d", session)
	}
	if messages != 3 {
		t.Errorf("expected 3 message events, got %d", messages)
	}
	if toolUse != 2 {
		t.Errorf("expected 2 tool_use events, got %d", toolUse)
	}
	if toolResult != 2 {
		t.Errorf("expected 2 tool_result events, got %d", toolResult)
	}
	if result != 1 {
		t.Errorf("expected 1 result event, got %d", result)
	}
}

func TestProviderData_Empty(t *testing.T) {
	if got := providerData(nil); got != nil {
		t.Errorf("expected nil for empty events, got %v", got)
	}
	if got := providerData([]map[string]any{}); got != nil {
		t.Errorf("expected nil for empty events, got %v", got)
	}
}

func TestProviderData_WrapsEvents(t *testing.T) {
	events := []map[string]any{
		{"type": "message", "text": "hi"},
		{"type": "result"},
	}
	got := providerData(events)
	if got == nil {
		t.Fatal("expected non-nil provider data")
	}
	raw, ok := got["codex_events"]
	if !ok {
		t.Fatal("expected 'codex_events' key")
	}
	list, ok := raw.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", raw)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 events, got %d", len(list))
	}
}

// recordingHandler captures every streaming callback for assertion.
type recordingHandler struct {
	mu          sync.Mutex
	turnStarts  []int
	turnEnds    []int
	texts       []string
	toolUses    []string
	toolResults []string
	errors      []error
}

func (h *recordingHandler) OnTurnStart(n int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.turnStarts = append(h.turnStarts, n)
}
func (h *recordingHandler) OnText(t string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.texts = append(h.texts, t)
}
func (h *recordingHandler) OnToolUse(name, input string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.toolUses = append(h.toolUses, name)
}
func (h *recordingHandler) OnToolResult(name, output string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.toolResults = append(h.toolResults, output)
}
func (h *recordingHandler) OnTurnEnd(n int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.turnEnds = append(h.turnEnds, n)
}
func (h *recordingHandler) OnError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errors = append(h.errors, err)
}

// writeFakeCodex writes a POSIX shell script to tmpDir that emits the fixture
// and exits 0. Returns the path to the fake binary.
func writeFakeCodex(t *testing.T, tmpDir, fixtureBody string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("mock binary tests use /bin/sh; skipping on windows")
	}
	fixturePath := filepath.Join(tmpDir, "fixture.jsonl")
	if err := os.WriteFile(fixturePath, []byte(fixtureBody), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	scriptPath := filepath.Join(tmpDir, "fake-codex")
	script := "#!/bin/sh\ncat " + fixturePath + "\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return scriptPath
}

func TestExecuteStreaming_ParsesFixture(t *testing.T) {
	fixture, err := os.ReadFile("testdata/codex_response.jsonl")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	tmpDir := t.TempDir()
	fake := writeFakeCodex(t, tmpDir, string(fixture))

	exec, _ := New(&executor.Config{
		CodexPath:      fake,
		CodexModel:     "gpt-5-codex",
		TimeoutSeconds: 30,
	})

	handler := &recordingHandler{}
	task := &executor.Task{
		ID:        "test-task-1",
		Directive: "write fizzbuzz",
		Workspace: tmpDir,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := exec.ExecuteStreaming(ctx, task, handler)
	if err != nil {
		t.Fatalf("ExecuteStreaming returned error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if result.NumTurns < 1 {
		t.Errorf("expected >= 1 turn, got %d", result.NumTurns)
	}
	if result.ToolCallCount != 2 {
		t.Errorf("expected 2 tool calls, got %d", result.ToolCallCount)
	}
	// Fixture's result event: tokens_used {160, 72}.
	if result.InputTokens != 160 {
		t.Errorf("expected 160 input tokens, got %d", result.InputTokens)
	}
	if result.OutputTokens != 72 {
		t.Errorf("expected 72 output tokens, got %d", result.OutputTokens)
	}
	if result.CostUSD <= 0 {
		t.Errorf("expected positive cost, got %v", result.CostUSD)
	}
	if result.SessionID != "codex-sess-abc123" {
		t.Errorf("expected session id from fixture, got %q", result.SessionID)
	}
	if !strings.Contains(result.Transcript, "Writing fizzbuzz implementation") {
		t.Errorf("expected fizzbuzz text in transcript, got: %s", result.Transcript)
	}
	if result.ProviderData == nil {
		t.Error("expected ProviderData to be populated")
	}
	if events, ok := result.ProviderData["codex_events"]; !ok {
		t.Error("expected codex_events in ProviderData")
	} else if list, _ := events.([]map[string]any); len(list) < 8 {
		t.Errorf("expected >= 8 raw events, got %d", len(list))
	}

	// Handler should have seen tool events and text.
	if len(handler.toolUses) != 2 {
		t.Errorf("expected 2 tool_use callbacks, got %d", len(handler.toolUses))
	}
	if len(handler.toolResults) != 2 {
		t.Errorf("expected 2 tool_result callbacks, got %d", len(handler.toolResults))
	}
	if len(handler.texts) == 0 {
		t.Error("expected at least one OnText callback")
	}
	if len(handler.errors) != 0 {
		t.Errorf("expected no errors, got %v", handler.errors)
	}
}

func TestExecuteStreaming_TolerantToNonJSONPreamble(t *testing.T) {
	body := "Loading configuration...\nConnecting to OpenAI...\n" +
		`{"type":"session","session_id":"s1"}` + "\n" +
		`{"type":"message","turn_number":1,"text":"ok","tokens_used":{"input":5,"output":3}}` + "\n" +
		`{"type":"result","tokens_used":{"input":5,"output":3},"status":"success"}` + "\n"

	tmpDir := t.TempDir()
	fake := writeFakeCodex(t, tmpDir, body)

	exec, _ := New(&executor.Config{CodexPath: fake, TimeoutSeconds: 10})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := exec.ExecuteStreaming(ctx, &executor.Task{ID: "t", Directive: "x"}, &executor.NoOpEventHandler{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success despite preamble, got error: %s", result.Error)
	}
	if result.SessionID != "s1" {
		t.Errorf("expected session from JSON, got %q", result.SessionID)
	}
}

func TestExecuteStreaming_NoResultMeansFailure(t *testing.T) {
	// Omit the final "result" event — Success should be false.
	body := `{"type":"session","session_id":"s1"}` + "\n" +
		`{"type":"message","turn_number":1,"text":"partial","tokens_used":{"input":1,"output":1}}` + "\n"

	tmpDir := t.TempDir()
	fake := writeFakeCodex(t, tmpDir, body)

	exec, _ := New(&executor.Config{CodexPath: fake, TimeoutSeconds: 10})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := exec.ExecuteStreaming(ctx, &executor.Task{ID: "t", Directive: "x"}, &executor.NoOpEventHandler{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected Success=false when no result event emitted")
	}
}

func TestExecuteStreaming_BinaryNotFound(t *testing.T) {
	exec, _ := New(&executor.Config{
		CodexPath:      "/nonexistent/path/to/codex-xyz-not-real",
		TimeoutSeconds: 5,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := exec.ExecuteStreaming(ctx, &executor.Task{ID: "t", Directive: "x"}, &executor.NoOpEventHandler{})
	if err == nil {
		t.Error("expected error for missing binary")
	}
}

func TestExecute_DelegatesToExecuteStreaming(t *testing.T) {
	body := `{"type":"result","tokens_used":{"input":1,"output":1},"status":"success"}` + "\n"

	tmpDir := t.TempDir()
	fake := writeFakeCodex(t, tmpDir, body)

	exec, _ := New(&executor.Config{CodexPath: fake, TimeoutSeconds: 10})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := exec.Execute(ctx, &executor.Task{ID: "t", Directive: "x"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.Success {
		t.Error("expected success from Execute")
	}
}

func TestHealthCheck_MissingBinary(t *testing.T) {
	exec, _ := New(&executor.Config{CodexPath: "/nonexistent/codex-not-here-xyz"})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := exec.HealthCheck(ctx); err == nil {
		t.Error("expected error for missing codex binary")
	}
}

func TestHealthCheck_SucceedsWhenBinaryRespondsToVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts unsupported on windows")
	}
	tmpDir := t.TempDir()
	// Fake binary that prints a version and exits 0 when invoked with --version.
	scriptPath := filepath.Join(tmpDir, "fake-codex")
	script := "#!/bin/sh\necho 'codex 0.0.1'\nexit 0\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	exec, _ := New(&executor.Config{CodexPath: scriptPath})
	// Generous: the fake-binary exec only needs ms, but CI runners under load have
	// blown the old 3s deadline (test-windows flake). A real hang still trips go test.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := exec.HealthCheck(ctx); err != nil {
		t.Errorf("expected HealthCheck to succeed, got %v", err)
	}
}

// TestLiveRun_Codex is a gated integration test. It runs the REAL codex
// binary end-to-end if one is available on PATH; otherwise skips cleanly.
// Run with: AILANG_CODEX_LIVE=1 go test ./internal/executor/codex/... -run TestLiveRun_Codex
func TestLiveRun_Codex(t *testing.T) {
	if os.Getenv("AILANG_CODEX_LIVE") == "" {
		t.Skip("set AILANG_CODEX_LIVE=1 to run; requires codex CLI + OPENAI_API_KEY")
	}
	if _, err := osexec.LookPath("codex"); err != nil {
		t.Skipf("codex binary not found on PATH: %v", err)
	}

	cfg := executor.DefaultConfig()
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := e.HealthCheck(ctx); err != nil {
		t.Skipf("codex HealthCheck failed (binary present but unusable): %v", err)
	}

	task := &executor.Task{
		ID:        "codex-live-smoke",
		Directive: "print the number 42 and nothing else",
		Workspace: t.TempDir(),
		Timeout:   90 * time.Second,
	}
	result, err := e.Execute(ctx, task)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if !result.Success {
		t.Errorf("live run reported failure: %s", result.Error)
	}
	if result.InputTokens == 0 || result.OutputTokens == 0 {
		t.Errorf("expected non-zero token counts, got input=%d output=%d", result.InputTokens, result.OutputTokens)
	}
	if result.CostUSD <= 0 {
		t.Errorf("expected positive cost, got %v", result.CostUSD)
	}
	if result.DurationMS <= 0 {
		t.Errorf("expected positive duration, got %d", result.DurationMS)
	}
}

func TestRegister_Idempotent(t *testing.T) {
	// init() already registered; calling Register() again should not panic
	// and should not duplicate the entry.
	Register()
	Register()

	available := executor.GlobalFactory().ListAvailable()
	count := 0
	for _, n := range available {
		if n == "codex" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 'codex' to appear exactly once, got %d", count)
	}
}
