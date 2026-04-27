package pi

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

const skipWindows = "mock binary tests use /bin/sh; skipping on windows"

func TestNewPiExecutor(t *testing.T) {
	cfg := &executor.Config{
		PiPath:         "/usr/local/bin/pi",
		PiModel:        "openai/gpt-5.4",
		TimeoutSeconds: 60,
	}
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if e.piPath != "/usr/local/bin/pi" {
		t.Errorf("piPath = %q, want /usr/local/bin/pi", e.piPath)
	}
	if e.model != "openai/gpt-5.4" {
		t.Errorf("model = %q, want openai/gpt-5.4", e.model)
	}
	if e.timeoutSeconds != 60 {
		t.Errorf("timeoutSeconds = %d, want 60", e.timeoutSeconds)
	}
}

func TestNewPiExecutor_EmptyConfigUsesFallbacks(t *testing.T) {
	cfg := &executor.Config{}
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if e.piPath != "pi" {
		t.Errorf("piPath fallback = %q, want \"pi\"", e.piPath)
	}
	if e.model != "anthropic/claude-haiku-4-5" {
		t.Errorf("model fallback = %q, want \"anthropic/claude-haiku-4-5\"", e.model)
	}
}

func TestPiCapabilities(t *testing.T) {
	e, _ := New(&executor.Config{})
	caps := e.Capabilities()
	want := map[executor.Capability]bool{
		executor.CapStreaming:      false,
		executor.CapLocalWorkspace: false,
		executor.CapToolControl:    false,
	}
	for _, c := range caps {
		want[c] = true
	}
	for cap, seen := range want {
		if !seen {
			t.Errorf("missing capability %q", cap)
		}
	}
}

func TestPiClose(t *testing.T) {
	e, _ := New(&executor.Config{})
	if err := e.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestGetModel_TaskOverride(t *testing.T) {
	e, _ := New(&executor.Config{PiModel: "anthropic/claude-haiku-4-5"})
	task := &executor.Task{Model: "openai/gpt-5.4"}
	if got := e.getModel(task); got != "openai/gpt-5.4" {
		t.Errorf("Task.Model override = %q, want \"openai/gpt-5.4\"", got)
	}
	taskNoModel := &executor.Task{}
	if got := e.getModel(taskNoModel); got != "anthropic/claude-haiku-4-5" {
		t.Errorf("default model = %q, want \"anthropic/claude-haiku-4-5\"", got)
	}
}

func TestBuildPiArgs_Defaults(t *testing.T) {
	args := buildPiArgs("anthropic/claude-haiku-4-5", &executor.Task{}, "fizz")
	want := []string{"--mode", "json", "--model", "anthropic/claude-haiku-4-5", "--no-session", "-p", "fizz"}
	if !equalSlices(args, want) {
		t.Errorf("args = %v, want %v", args, want)
	}
}

func TestBuildPiArgs_NoTools(t *testing.T) {
	args := buildPiArgs("openai/gpt-5.4", &executor.Task{AllowedTools: []string{}}, "do x")
	// AllowedTools = empty slice → --no-tools
	if !contains(args, "--no-tools") {
		t.Errorf("expected --no-tools flag for empty AllowedTools, got %v", args)
	}
}

func TestBuildPiArgs_AllowedTools(t *testing.T) {
	args := buildPiArgs("anthropic/claude-haiku-4-5",
		&executor.Task{AllowedTools: []string{"read", "grep"}},
		"summarize")
	idx := indexOf(args, "--tools")
	if idx < 0 || idx+1 >= len(args) {
		t.Fatalf("expected --tools <list> in args, got %v", args)
	}
	if args[idx+1] != "read,grep" {
		t.Errorf("--tools value = %q, want \"read,grep\"", args[idx+1])
	}
}

// TestParsePiEvent_Session verifies the session ID is parsed from the
// `session` event so it can be propagated to Result.SessionID.
func TestParsePiEvent_Session(t *testing.T) {
	line := []byte(`{"type":"session","version":3,"id":"sess-abc","timestamp":"2026-04-27T05:38:11.436Z","cwd":"/tmp"}`)
	ev, err := parsePiEvent(line)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ev.Type != "session" || ev.SessionID != "sess-abc" {
		t.Errorf("got type=%q sid=%q, want session/sess-abc", ev.Type, ev.SessionID)
	}
}

func TestParsePiEvent_TextDelta(t *testing.T) {
	line := []byte(`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","contentIndex":1,"delta":"hello"}}`)
	ev, err := parsePiEvent(line)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ev.AssistantMessageEvent == nil {
		t.Fatal("expected assistantMessageEvent populated")
	}
	if ev.AssistantMessageEvent.Type != "text_delta" || ev.AssistantMessageEvent.Delta != "hello" {
		t.Errorf("got type=%q delta=%q, want text_delta/hello",
			ev.AssistantMessageEvent.Type, ev.AssistantMessageEvent.Delta)
	}
}

func TestParsePiEvent_MessageEnd_AssistantUsage(t *testing.T) {
	line := []byte(`{"type":"message_end","message":{"role":"assistant","usage":{"input":480,"output":205,"cacheRead":0,"cacheWrite":0,"totalTokens":685,"cost":{"input":0.00048,"output":0.001025,"cacheRead":0,"cacheWrite":0,"total":0.001505}}}}`)
	ev, err := parsePiEvent(line)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ev.Message == nil || ev.Message.Usage == nil {
		t.Fatal("expected message.usage populated")
	}
	if ev.Message.Role != "assistant" {
		t.Errorf("role = %q, want assistant", ev.Message.Role)
	}
	if ev.Message.Usage.Input != 480 || ev.Message.Usage.Output != 205 {
		t.Errorf("tokens (input=%d output=%d), want (480, 205)",
			ev.Message.Usage.Input, ev.Message.Usage.Output)
	}
	if ev.Message.Usage.Cost.Total < 0.001504 || ev.Message.Usage.Cost.Total > 0.001506 {
		t.Errorf("cost.total = %f, want ~0.001505", ev.Message.Usage.Cost.Total)
	}
}

func TestParsePiEvent_ToolExecutionStart(t *testing.T) {
	line := []byte(`{"type":"tool_execution_start","toolCallId":"toolu_x","toolName":"write","args":{"path":"hello.txt","content":"hello world"}}`)
	ev, err := parsePiEvent(line)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ev.ToolName != "write" {
		t.Errorf("toolName = %q, want write", ev.ToolName)
	}
	if !strings.Contains(string(ev.Args), "hello.txt") {
		t.Errorf("args raw = %q, expected to contain \"hello.txt\"", string(ev.Args))
	}
}

func TestParsePiEvent_ToolExecutionEnd(t *testing.T) {
	line := []byte(`{"type":"tool_execution_end","toolCallId":"toolu_x","toolName":"write","result":{"content":[{"type":"text","text":"Successfully wrote 11 bytes"}]},"isError":false}`)
	ev, err := parsePiEvent(line)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ev.Result == nil || len(ev.Result.Content) == 0 {
		t.Fatal("expected result.content populated")
	}
	out := flattenPiToolResult(ev.Result)
	if !strings.Contains(out, "Successfully wrote") {
		t.Errorf("flattened output = %q, expected substring \"Successfully wrote\"", out)
	}
}

func TestParsePiEvent_NonJSON(t *testing.T) {
	if _, err := parsePiEvent([]byte("warning: starting up")); err == nil {
		t.Error("expected error for non-JSON line")
	}
}

func TestParsePiEvent_Empty(t *testing.T) {
	if _, err := parsePiEvent([]byte("")); err == nil {
		t.Error("expected error for empty line")
	}
}

func TestParsePiEvent_UnknownType(t *testing.T) {
	// Forward-compat: unknown type values must parse cleanly so the streaming
	// loop ignores them rather than aborting.
	line := []byte(`{"type":"future_event_kind","payload":{"x":1}}`)
	ev, err := parsePiEvent(line)
	if err != nil {
		t.Fatalf("unknown event type should not panic parser: %v", err)
	}
	if ev.Type != "future_event_kind" {
		t.Errorf("type = %q, want future_event_kind", ev.Type)
	}
	if ev.Raw == nil {
		t.Error("expected Raw populated for unknown type")
	}
}

func TestParsePiEvent_PreservesRaw(t *testing.T) {
	line := []byte(`{"type":"agent_start","custom_field":"vendor_specific"}`)
	ev, err := parsePiEvent(line)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if v, ok := ev.Raw["custom_field"]; !ok || v != "vendor_specific" {
		t.Errorf("Raw missing custom_field; Raw = %v", ev.Raw)
	}
}

// TestExecuteStreaming_FizzbuzzFixture replays the recorded no-tools fixture
// through a mock pi binary and asserts token/turn invariants.
func TestExecuteStreaming_FizzbuzzFixture(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	dir := t.TempDir()
	events := loadFixtureLines(t, "fizzbuzz.ndjson")
	_ = writeFakePi(t, dir, events)

	cfg := &executor.Config{
		PiPath:         filepath.Join(dir, "pi"),
		PiModel:        "anthropic/claude-haiku-4-5",
		TimeoutSeconds: 10,
	}
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var mu sync.Mutex
	var texts []string
	var turnStarts []int

	handler := &collectingHandler{
		onText:      func(s string) { mu.Lock(); texts = append(texts, s); mu.Unlock() },
		onTurnStart: func(n int) { mu.Lock(); turnStarts = append(turnStarts, n); mu.Unlock() },
	}

	task := &executor.Task{
		ID:        "test-fizzbuzz",
		Directive: "write fizzbuzz",
		Workspace: dir,
		Timeout:   10 * time.Second,
	}
	result, err := e.ExecuteStreaming(context.Background(), task, handler)
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}

	// Per-turn message_end usage for assistant: 480 input, 205 output.
	if result.InputTokens != 480 {
		t.Errorf("InputTokens = %d, want 480", result.InputTokens)
	}
	if result.OutputTokens != 205 {
		t.Errorf("OutputTokens = %d, want 205", result.OutputTokens)
	}

	// Cost: 0.001505 (single turn, taken from message_end.usage.cost.total).
	const wantCost = 0.001505
	const epsilon = 1e-7
	if diff := result.CostUSD - wantCost; diff > epsilon || diff < -epsilon {
		t.Errorf("CostUSD = %.8f, want %.8f", result.CostUSD, wantCost)
	}

	// Single turn (one turn_start event).
	if result.NumTurns != 1 {
		t.Errorf("NumTurns = %d, want 1", result.NumTurns)
	}

	// No tools invoked.
	if result.ToolCallCount != 0 {
		t.Errorf("ToolCallCount = %d, want 0", result.ToolCallCount)
	}

	// Text streamed via OnText (text_delta events accumulated).
	if len(texts) == 0 {
		t.Error("expected OnText to fire at least once")
	}
	if !strings.Contains(result.Output, "FizzBuzz") {
		t.Errorf("Output missing 'FizzBuzz'; got %q", result.Output)
	}

	// turn_start fired once with turn number 1.
	if len(turnStarts) != 1 || turnStarts[0] != 1 {
		t.Errorf("turnStarts = %v, want [1]", turnStarts)
	}

	// Session ID from session event.
	if result.SessionID == "" {
		t.Error("expected SessionID populated from session event")
	}

	if result.ProviderData == nil {
		t.Error("expected ProviderData populated")
	}
	if _, ok := result.ProviderData["pi_events"]; !ok {
		t.Error("expected 'pi_events' key in ProviderData")
	}
}

// TestExecuteStreaming_ToolUseFixture replays the recorded tool-call fixture
// and asserts tool events fire and per-turn usage is summed.
func TestExecuteStreaming_ToolUseFixture(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	dir := t.TempDir()
	events := loadFixtureLines(t, "tool_use.ndjson")
	_ = writeFakePi(t, dir, events)

	cfg := &executor.Config{
		PiPath:         filepath.Join(dir, "pi"),
		TimeoutSeconds: 10,
	}
	e, _ := New(cfg)

	var mu sync.Mutex
	var toolCalls []string
	var toolResults []string

	handler := &collectingHandler{
		onToolUse:    func(name, _ string) { mu.Lock(); toolCalls = append(toolCalls, name); mu.Unlock() },
		onToolResult: func(name, _ string) { mu.Lock(); toolResults = append(toolResults, name); mu.Unlock() },
	}

	task := &executor.Task{
		ID:        "test-tool-use",
		Directive: "create hello.txt",
		Workspace: dir,
		Timeout:   10 * time.Second,
	}
	result, err := e.ExecuteStreaming(context.Background(), task, handler)
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}

	// Per-turn deltas summed across two turns: 10+13=23 input, 128+84=212 output.
	if result.InputTokens != 23 {
		t.Errorf("InputTokens = %d, want 23", result.InputTokens)
	}
	if result.OutputTokens != 212 {
		t.Errorf("OutputTokens = %d, want 212", result.OutputTokens)
	}

	// Cost: 0.006343750000000001 + 0.006319250000000001 ≈ 0.012663
	const wantCost = 0.012663
	const epsilon = 1e-5
	if diff := result.CostUSD - wantCost; diff > epsilon || diff < -epsilon {
		t.Errorf("CostUSD = %.8f, want %.8f", result.CostUSD, wantCost)
	}

	// Two turns (two turn_start events).
	if result.NumTurns != 2 {
		t.Errorf("NumTurns = %d, want 2", result.NumTurns)
	}

	// One tool execution.
	if result.ToolCallCount != 1 {
		t.Errorf("ToolCallCount = %d, want 1", result.ToolCallCount)
	}
	if len(toolCalls) != 1 || toolCalls[0] != "write" {
		t.Errorf("toolCalls = %v, want [write]", toolCalls)
	}
	if len(toolResults) != 1 || toolResults[0] != "write" {
		t.Errorf("toolResults = %v, want [write]", toolResults)
	}
}

func TestExecuteStreaming_NonJSONPreambleTolerated(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	dir := t.TempDir()
	events := append(
		[]string{
			"warning: this is a non-json preamble",
			"info: connecting to provider",
		},
		loadFixtureLines(t, "fizzbuzz.ndjson")...,
	)
	_ = writeFakePi(t, dir, events)

	cfg := &executor.Config{
		PiPath:         filepath.Join(dir, "pi"),
		TimeoutSeconds: 10,
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
		PiPath:         "/nonexistent/pi-bin",
		TimeoutSeconds: 5,
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
	cfg := &executor.Config{PiPath: "/nonexistent/pi-does-not-exist"}
	e, _ := New(cfg)
	if err := e.HealthCheck(context.Background()); err == nil {
		t.Error("expected HealthCheck to fail for missing binary")
	}
}

func TestHealthCheck_WithFakeBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	dir := t.TempDir()
	_ = writeFakePi(t, dir, nil)
	cfg := &executor.Config{
		PiPath:         filepath.Join(dir, "pi"),
		TimeoutSeconds: 5,
	}
	e, _ := New(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.HealthCheck(ctx); err != nil {
		t.Errorf("expected HealthCheck to succeed, got %v", err)
	}
}

// TestInit_RegistersPi verifies that importing this package (via init())
// registers "pi" in the global executor factory.
func TestInit_RegistersPi(t *testing.T) {
	factory := executor.GlobalFactory()
	available := factory.ListAvailable()

	found := false
	for _, name := range available {
		if name == "pi" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("init() did not register 'pi'; factory.ListAvailable() = %v", available)
	}

	e, err := factory.GetExecutor("pi")
	if err != nil {
		t.Fatalf("factory.GetExecutor(\"pi\") failed: %v", err)
	}
	if e.Name() != "pi" {
		t.Errorf("built executor has wrong name: %q", e.Name())
	}
}

func TestRegister_Idempotent(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Register() panicked on double registration: %v", r)
		}
	}()
	Register()
	Register()
}

// TestLiveRun_Pi is a gated integration test. Requires AILANG_PI_LIVE=1
// and a working pi CLI with an Anthropic API key.
// Run with: AILANG_PI_LIVE=1 go test ./internal/executor/pi/... -run TestLiveRun_Pi
func TestLiveRun_Pi(t *testing.T) {
	if os.Getenv("AILANG_PI_LIVE") == "" {
		t.Skip("set AILANG_PI_LIVE=1 to run; requires pi CLI + provider key")
	}
	if _, err := osexec.LookPath("pi"); err != nil {
		t.Skipf("pi binary not found on PATH: %v", err)
	}

	cfg := executor.DefaultConfig()
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := e.HealthCheck(ctx); err != nil {
		t.Skipf("pi HealthCheck failed: %v", err)
	}

	task := &executor.Task{
		ID:           "pi-live-smoke",
		Directive:    "print the number 42 and nothing else",
		Workspace:    t.TempDir(),
		AllowedTools: []string{}, // disable tools for deterministic output
		Timeout:      90 * time.Second,
	}
	result, err := e.Execute(ctx, task)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}
	if !result.Success {
		t.Errorf("live run reported failure: %s", result.Error)
	}
	if result.InputTokens == 0 && result.OutputTokens == 0 {
		t.Error("expected non-zero token counts")
	}
	if result.DurationMS <= 0 {
		t.Errorf("expected positive duration, got %d", result.DurationMS)
	}
}

// --- helpers ---

// writeFakePi writes a POSIX shell script that emits a canned pi NDJSON event
// stream when invoked as `pi --mode json ...`. Used to test the streaming
// executor without a real pi binary.
func writeFakePi(t *testing.T, dir string, events []string) string {
	t.Helper()
	script := filepath.Join(dir, "pi")
	var body strings.Builder
	body.WriteString("#!/bin/sh\n")
	body.WriteString(`
case "$1" in
  --version)
    echo "0.70.2"
    exit 0 ;;
esac
`)
	for _, ev := range events {
		body.WriteString("printf '%s\\n' '")
		body.WriteString(strings.ReplaceAll(ev, "'", "'\\''"))
		body.WriteString("'\n")
	}
	body.WriteString("exit 0\n")

	if err := os.WriteFile(script, []byte(body.String()), 0755); err != nil {
		t.Fatalf("writeFakePi: %v", err)
	}
	return script
}

func loadFixtureLines(t *testing.T, name string) []string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(thisFile), "testdata", name)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture %s: %v", name, err)
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan fixture %s: %v", name, err)
	}
	return lines
}

func contains(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

func indexOf(slice []string, target string) int {
	for i, s := range slice {
		if s == target {
			return i
		}
	}
	return -1
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// collectingHandler implements executor.EventHandler recording events.
type collectingHandler struct {
	onText       func(string)
	onToolUse    func(string, string)
	onToolResult func(string, string)
	onTurnStart  func(int)
	onTurnEnd    func(int)
}

func (h *collectingHandler) OnTurnStart(n int) {
	if h.onTurnStart != nil {
		h.onTurnStart(n)
	}
}
func (h *collectingHandler) OnText(s string) {
	if h.onText != nil {
		h.onText(s)
	}
}
func (h *collectingHandler) OnToolUse(name, input string) {
	if h.onToolUse != nil {
		h.onToolUse(name, input)
	}
}
func (h *collectingHandler) OnToolResult(name, output string) {
	if h.onToolResult != nil {
		h.onToolResult(name, output)
	}
}
func (h *collectingHandler) OnTurnEnd(n int) {
	if h.onTurnEnd != nil {
		h.onTurnEnd(n)
	}
}
func (h *collectingHandler) OnError(err error) {}
