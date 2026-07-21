package opencode

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

// writeCapturingOpenCode writes a fake opencode that, on `run`, records the
// final positional arg (the message) to <cwd>/captured_msg and a copy of any
// AGENTS.md found in cwd to <cwd>/captured_agents (or "NONE"), then emits a
// minimal valid event stream. Lets tests assert how SystemPrompt was delivered.
func writeCapturingOpenCode(t *testing.T, dir string) string {
	t.Helper()
	script := filepath.Join(dir, "opencode")
	body := `#!/bin/sh
case "$1" in
  run)
    for a in "$@"; do last="$a"; done
    printf '%s' "$last" > "$PWD/captured_msg"
    if [ -f "$PWD/AGENTS.md" ]; then cp "$PWD/AGENTS.md" "$PWD/captured_agents"; else printf 'NONE' > "$PWD/captured_agents"; fi
    printf '%s\n' '{"type":"step_start","sessionID":"ses_t","part":{"id":"p1","messageID":"m1","sessionID":"ses_t","type":"step-start"}}'
    printf '%s\n' '{"type":"step_finish","sessionID":"ses_t","part":{"id":"p2","reason":"stop","messageID":"m1","sessionID":"ses_t","type":"step-finish","tokens":{"total":10,"input":1,"output":9,"reasoning":0,"cache":{"write":0,"read":0}},"cost":0.0}}'
    exit 0 ;;
  --version) echo "1.15.7"; exit 0 ;;
  *) exit 0 ;;
esac
`
	if err := os.WriteFile(script, []byte(body), 0755); err != nil {
		t.Fatalf("writeCapturingOpenCode: %v", err)
	}
	return script
}

func TestExecuteStreaming_PersistentSystemPrompt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	const sysPrompt = "TEACHING-PROMPT-SENTINEL: AILANG rules go here."
	const directive = "Solve the benchmark task."

	cases := []struct {
		name       string
		persistent bool
		wantAgents string // expected captured_agents content
		wantMsg    string // expected captured_msg content
	}{
		{
			name:       "persistent_writes_AGENTS_md_and_message_is_task_only",
			persistent: true,
			wantAgents: sysPrompt,
			wantMsg:    directive,
		},
		{
			name:       "default_no_AGENTS_md_message_is_concatenated",
			persistent: false,
			wantAgents: "NONE",
			wantMsg:    sysPrompt + "\n\n" + directive,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			_ = writeCapturingOpenCode(t, dir)
			e, err := New(&executor.Config{
				OpenCodePath:   filepath.Join(dir, "opencode"),
				OpenCodeModel:  "ollama/qwen3.5:35b-a3b-mxfp8",
				TimeoutSeconds: 10,
			})
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			task := &executor.Task{
				ID:                     "test-persist",
				Directive:              directive,
				SystemPrompt:           sysPrompt,
				PersistentSystemPrompt: tc.persistent,
				Workspace:              dir,
				Timeout:                10 * time.Second,
			}
			if _, err := e.ExecuteStreaming(context.Background(), task, &executor.NoOpEventHandler{}); err != nil {
				t.Fatalf("ExecuteStreaming: %v", err)
			}

			gotAgents, err := os.ReadFile(filepath.Join(dir, "captured_agents"))
			if err != nil {
				t.Fatalf("read captured_agents: %v", err)
			}
			if string(gotAgents) != tc.wantAgents {
				t.Errorf("AGENTS.md content:\n  got  %q\n  want %q", gotAgents, tc.wantAgents)
			}

			gotMsg, err := os.ReadFile(filepath.Join(dir, "captured_msg"))
			if err != nil {
				t.Fatalf("read captured_msg: %v", err)
			}
			if string(gotMsg) != tc.wantMsg {
				t.Errorf("opencode message arg:\n  got  %q\n  want %q", gotMsg, tc.wantMsg)
			}

			// Persistent mode must NOT also bury the prompt in the message.
			if tc.persistent && strings.Contains(string(gotMsg), sysPrompt) {
				t.Errorf("persistent mode leaked SystemPrompt into the user message: %q", gotMsg)
			}
		})
	}
}

func TestNewOpenCodeExecutor(t *testing.T) {
	cfg := executor.DefaultConfig()
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if e.Name() != "opencode" {
		t.Errorf("expected name 'opencode', got %q", e.Name())
	}
	if e.opencodePath != "opencode" {
		t.Errorf("expected default opencodePath 'opencode', got %q", e.opencodePath)
	}
	if e.model != "anthropic/claude-haiku-4-5" {
		t.Errorf("expected default model 'anthropic/claude-haiku-4-5', got %q", e.model)
	}
}

func TestNewOpenCodeExecutor_EmptyConfigUsesFallbacks(t *testing.T) {
	e, err := New(&executor.Config{})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if e.opencodePath != "opencode" {
		t.Errorf("expected fallback path 'opencode', got %q", e.opencodePath)
	}
	if e.model != "anthropic/claude-haiku-4-5" {
		t.Errorf("expected fallback model, got %q", e.model)
	}
}

func TestNewOpenCodeExecutor_CustomPathAndModel(t *testing.T) {
	cfg := &executor.Config{
		OpenCodePath:  "/custom/bin/opencode",
		OpenCodeModel: "ollama/gemma4:latest",
	}
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if e.opencodePath != "/custom/bin/opencode" {
		t.Errorf("expected custom path, got %q", e.opencodePath)
	}
	if e.model != "ollama/gemma4:latest" {
		t.Errorf("expected custom model, got %q", e.model)
	}
}

func TestOpenCodeCapabilities(t *testing.T) {
	e, _ := New(executor.DefaultConfig())
	caps := e.Capabilities()
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
	for cap, found := range want {
		if !found {
			t.Errorf("expected capability %v not found", cap)
		}
	}
}

func TestOpenCodeClose(t *testing.T) {
	e, _ := New(executor.DefaultConfig())
	if err := e.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestGetModel_TaskOverride(t *testing.T) {
	e, _ := New(executor.DefaultConfig())
	task := &executor.Task{Model: "ollama/gemma4:latest"}
	if got := e.getModel(task); got != "ollama/gemma4:latest" {
		t.Errorf("expected task model override, got %q", got)
	}
	empty := &executor.Task{}
	if got := e.getModel(empty); got != "anthropic/claude-haiku-4-5" {
		t.Errorf("expected default model, got %q", got)
	}
}

func TestParseOpenCodeEvent_StepStart(t *testing.T) {
	line := []byte(`{"type":"step_start","timestamp":1776860240950,"sessionID":"ses_abc","part":{"id":"prt_1","messageID":"msg_1","sessionID":"ses_abc","type":"step-start"}}`)
	ev, err := parseOpenCodeEvent(line)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if ev.Type != "step_start" {
		t.Errorf("expected type 'step_start', got %q", ev.Type)
	}
	if ev.SessionID != "ses_abc" {
		t.Errorf("expected sessionID 'ses_abc', got %q", ev.SessionID)
	}
	if ev.Timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}
	if ev.Raw == nil {
		t.Error("expected Raw map populated for schema-drift tolerance")
	}
}

func TestParseOpenCodeEvent_Text(t *testing.T) {
	line := []byte(`{"type":"text","timestamp":1776860241181,"sessionID":"ses_abc","part":{"id":"prt_2","messageID":"msg_1","sessionID":"ses_abc","type":"text","text":"Hello!","time":{"start":1776860240949,"end":1776860241181}}}`)
	ev, err := parseOpenCodeEvent(line)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if ev.Type != "text" {
		t.Errorf("expected type 'text', got %q", ev.Type)
	}
	if ev.Part.Text != "Hello!" {
		t.Errorf("expected text 'Hello!', got %q", ev.Part.Text)
	}
}

func TestParseOpenCodeEvent_ToolUse(t *testing.T) {
	line := []byte(`{"type":"tool_use","timestamp":1776860359892,"sessionID":"ses_abc","part":{"type":"tool","tool":"write","callID":"toolu_abc","state":{"status":"completed","input":{"filePath":"/tmp/x.txt","content":"hello"},"output":"Wrote file successfully.","title":"x.txt"},"id":"prt_3","sessionID":"ses_abc","messageID":"msg_1"}}`)
	ev, err := parseOpenCodeEvent(line)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if ev.Type != "tool_use" {
		t.Errorf("expected type 'tool_use', got %q", ev.Type)
	}
	if ev.Part.Tool != "write" {
		t.Errorf("expected tool 'write', got %q", ev.Part.Tool)
	}
	if ev.Part.CallID != "toolu_abc" {
		t.Errorf("expected callID 'toolu_abc', got %q", ev.Part.CallID)
	}
	if ev.Part.State.Status != "completed" {
		t.Errorf("expected status 'completed', got %q", ev.Part.State.Status)
	}
	if ev.Part.State.Output != "Wrote file successfully." {
		t.Errorf("expected output, got %q", ev.Part.State.Output)
	}
}

func TestParseOpenCodeEvent_StepFinish(t *testing.T) {
	line := []byte(`{"type":"step_finish","timestamp":1776860241203,"sessionID":"ses_abc","part":{"id":"prt_4","reason":"stop","messageID":"msg_1","sessionID":"ses_abc","type":"step-finish","tokens":{"total":17240,"input":1,"output":26,"reasoning":0,"cache":{"write":17213,"read":0}},"cost":0.02164725}}`)
	ev, err := parseOpenCodeEvent(line)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if ev.Type != "step_finish" {
		t.Errorf("expected type 'step_finish', got %q", ev.Type)
	}
	if ev.Part.Reason != "stop" {
		t.Errorf("expected reason 'stop', got %q", ev.Part.Reason)
	}
	if ev.Part.Tokens.Input != 1 {
		t.Errorf("expected input tokens 1, got %d", ev.Part.Tokens.Input)
	}
	if ev.Part.Tokens.Output != 26 {
		t.Errorf("expected output tokens 26, got %d", ev.Part.Tokens.Output)
	}
	if ev.Part.Cost <= 0 {
		t.Errorf("expected positive cost, got %v", ev.Part.Cost)
	}
}

func TestParseOpenCodeEvent_NonJSON(t *testing.T) {
	_, err := parseOpenCodeEvent([]byte("Performing one time database migration..."))
	if err == nil {
		t.Error("expected error for non-JSON line")
	}
}

func TestParseOpenCodeEvent_Empty(t *testing.T) {
	_, err := parseOpenCodeEvent([]byte(""))
	if err == nil {
		t.Error("expected error for empty line")
	}
}

func TestParseOpenCodeEvent_UnknownType(t *testing.T) {
	// Unknown event types must parse without error (forward-compat).
	line := []byte(`{"type":"future_event_type","timestamp":1234567890,"sessionID":"ses_x","part":{}}`)
	ev, err := parseOpenCodeEvent(line)
	if err != nil {
		t.Fatalf("expected unknown event type to parse cleanly, got: %v", err)
	}
	if ev.Type != "future_event_type" {
		t.Errorf("expected type 'future_event_type', got %q", ev.Type)
	}
	if ev.Raw == nil {
		t.Error("expected Raw preserved for unknown events")
	}
}

func TestParseOpenCodeEvent_SchemaGift_ProviderData(t *testing.T) {
	// Provider-specific metadata (e.g. anthropic.caller.type) must be tolerated
	// and preserved in Raw without causing parse errors.
	line := []byte(`{"type":"tool_use","timestamp":1234,"sessionID":"ses_y","part":{"type":"tool","tool":"read","callID":"c1","state":{"status":"running","input":{},"output":"","title":""},"metadata":{"anthropic":{"caller":{"type":"direct"}}},"id":"prt_x","sessionID":"ses_y","messageID":"msg_y"}}`)
	ev, err := parseOpenCodeEvent(line)
	if err != nil {
		t.Fatalf("schema drift should not cause parse error: %v", err)
	}
	if ev.Raw == nil {
		t.Error("expected Raw populated")
	}
}

// writeFakeOpenCode writes a POSIX shell script that emits a canned opencode
// JSON event stream when invoked as `opencode run ... --format json`.
// Used to test the streaming executor without a real opencode binary.
func writeFakeOpenCode(t *testing.T, dir string, events []string) string {
	t.Helper()
	script := filepath.Join(dir, "opencode")
	// Build the script body: print each event line then exit.
	var body strings.Builder
	body.WriteString("#!/bin/sh\n")
	// Only emit events when called with 'run' and '--format' args.
	body.WriteString(`
case "$1" in
  run)
    true ;;  # fall through to event emission
  --version)
    echo "1.14.20"
    exit 0 ;;
  *)
    exit 0 ;;
esac
`)
	for _, ev := range events {
		// Use printf to avoid issues with special chars in echo
		body.WriteString("printf '%s\\n' '")
		body.WriteString(strings.ReplaceAll(ev, "'", "'\\''"))
		body.WriteString("'\n")
	}
	body.WriteString("exit 0\n")

	if err := os.WriteFile(script, []byte(body.String()), 0755); err != nil {
		t.Fatalf("writeFakeOpenCode: %v", err)
	}
	return script
}

// mockEvents returns the canonical 3-step fixture (1 text + 1 tool_use step).
func mockEvents() []string {
	return []string{
		`{"type":"step_start","timestamp":1776860240950,"sessionID":"ses_test","part":{"id":"prt_1","messageID":"msg_1","sessionID":"ses_test","type":"step-start"}}`,
		`{"type":"text","timestamp":1776860241181,"sessionID":"ses_test","part":{"id":"prt_2","messageID":"msg_1","sessionID":"ses_test","type":"text","text":"I will create the file.","time":{"start":1776860240949,"end":1776860241181}}}`,
		`{"type":"step_finish","timestamp":1776860241203,"sessionID":"ses_test","part":{"id":"prt_3","reason":"tool-calls","messageID":"msg_1","sessionID":"ses_test","type":"step-finish","tokens":{"total":17240,"input":1,"output":26,"reasoning":0,"cache":{"write":17213,"read":0}},"cost":0.02164725}}`,
		`{"type":"step_start","timestamp":1776860358605,"sessionID":"ses_test","part":{"id":"prt_4","messageID":"msg_2","sessionID":"ses_test","type":"step-start"}}`,
		`{"type":"tool_use","timestamp":1776860359892,"sessionID":"ses_test","part":{"type":"tool","tool":"write","callID":"toolu_1","state":{"status":"completed","input":{"filePath":"/tmp/x.txt","content":"hi"},"output":"Wrote file successfully.","title":"x.txt","time":{"start":1776860359882,"end":1776860359891}},"id":"prt_5","sessionID":"ses_test","messageID":"msg_2"}}`,
		`{"type":"step_finish","timestamp":1776860359982,"sessionID":"ses_test","part":{"id":"prt_6","reason":"stop","messageID":"msg_2","sessionID":"ses_test","type":"step-finish","tokens":{"total":17357,"input":1,"output":133,"reasoning":0,"cache":{"write":17223,"read":0}},"cost":0.02219475}}`,
	}
}

func TestExecuteStreaming_ParsesFixture(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	dir := t.TempDir()
	_ = writeFakeOpenCode(t, dir, mockEvents())

	cfg := &executor.Config{
		OpenCodePath:   filepath.Join(dir, "opencode"),
		OpenCodeModel:  "anthropic/claude-haiku-4-5",
		TimeoutSeconds: 10,
	}
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var mu sync.Mutex
	var texts []string
	var toolCalls []string
	var toolResults []string
	var turnStarts []int

	handler := &collectingHandler{
		onText:       func(s string) { mu.Lock(); texts = append(texts, s); mu.Unlock() },
		onToolUse:    func(name, _ string) { mu.Lock(); toolCalls = append(toolCalls, name); mu.Unlock() },
		onToolResult: func(name, _ string) { mu.Lock(); toolResults = append(toolResults, name); mu.Unlock() },
		onTurnStart:  func(n int) { mu.Lock(); turnStarts = append(turnStarts, n); mu.Unlock() },
	}

	task := &executor.Task{
		ID:        "test-stream",
		Directive: "create a file",
		Workspace: dir,
		Timeout:   10 * time.Second,
	}

	result, err := e.ExecuteStreaming(context.Background(), task, handler)
	if err != nil {
		t.Fatalf("ExecuteStreaming error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}

	// Text accumulation
	if !strings.Contains(result.Output, "I will create the file.") {
		t.Errorf("output missing expected text, got: %q", result.Output)
	}

	// Per-step delta token summation: input=1+1=2, output=26+133=159
	if result.InputTokens != 2 {
		t.Errorf("expected input tokens 2 (per-step deltas summed), got %d", result.InputTokens)
	}
	if result.OutputTokens != 159 {
		t.Errorf("expected output tokens 159 (per-step deltas summed), got %d", result.OutputTokens)
	}

	// Cost summed across steps
	const wantCost = 0.02164725 + 0.02219475
	const epsilon = 1e-7
	if diff := result.CostUSD - wantCost; diff > epsilon || diff < -epsilon {
		t.Errorf("cost = %.8f, want %.8f", result.CostUSD, wantCost)
	}

	// NumTurns = step_start count
	if result.NumTurns != 2 {
		t.Errorf("NumTurns = %d, want 2", result.NumTurns)
	}

	// Tool calls: fixture has one completed tool_use → OnToolResult called
	if len(toolResults) != 1 || toolResults[0] != "write" {
		t.Errorf("expected 1 tool result 'write', got %v", toolResults)
	}

	// Session ID propagated for --session resume
	if result.SessionID != "ses_test" {
		t.Errorf("SessionID = %q, want 'ses_test'", result.SessionID)
	}

	// ProviderData populated
	if result.ProviderData == nil {
		t.Error("expected ProviderData to be set")
	}
	if _, ok := result.ProviderData["opencode_events"]; !ok {
		t.Error("expected 'opencode_events' key in ProviderData")
	}

	// FinishReason comes from the LAST step_finish. The fixture's first step
	// reports "tool-calls" (mid-run handoff) and the second "stop" — if an
	// intermediate step won, every tool-using run would look like it ended on a
	// tool call.
	if result.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want \"stop\" (last step_finish, not the intermediate \"tool-calls\")", result.FinishReason)
	}

	// Fixture reports reasoning:0 on both steps.
	if result.ReasonTokens != 0 {
		t.Errorf("ReasonTokens = %d, want 0 (fixture reports reasoning:0)", result.ReasonTokens)
	}
}

// TestExecuteStreaming_CapturesReasoningAndTruncation guards the call site that
// silently dropped this data: opencode parsed tokens.reasoning and the
// step_finish reason for a long time, but neither was ever copied into
// executor.Result, so agent-mode rows banked reasoning as 0 and finish_reason as
// empty 100% of the time (see eval_results/baselines/v0.30.0/CAVEATS.md).
//
// "length" is the case that matters: it is how a reasoning model that exhausts
// its budget mid-thought reports itself. Without it, truncation is
// indistinguishable from a capability gap.
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
	cfg := &executor.Config{OpenCodePath: "/nonexistent/opencode-does-not-exist"}
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

	e, err := factory.GetExecutor("opencode")
	if err != nil {
		t.Fatalf("factory.GetExecutor(\"opencode\") failed: %v", err)
	}
	if e.Name() != "opencode" {
		t.Errorf("built executor has wrong name: %q", e.Name())
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
func TestExecuteStreaming_ParsesFixture_FromFile(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	fixture := filepath.Join(filepath.Dir(thisFile), "testdata", "opencode_response.jsonl")

	f, err := os.Open(fixture)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	var totalInput, totalOutput int
	var totalCost float64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Bytes()
		ev, err := parseOpenCodeEvent(line)
		if err != nil {
			continue
		}
		if ev.Type == "step_finish" {
			totalInput += ev.Part.Tokens.Input
			totalOutput += ev.Part.Tokens.Output
			totalCost += ev.Part.Cost
		}
	}

	// Fixture: 3 step_finish events, input=1+1+3=5, output=26+133+25=184
	if totalInput != 5 {
		t.Errorf("summed input = %d, want 5", totalInput)
	}
	if totalOutput != 184 {
		t.Errorf("summed output = %d, want 184", totalOutput)
	}
	const wantCost = 0.04587605
	const epsilon = 1e-6
	if diff := totalCost - wantCost; diff > epsilon || diff < -epsilon {
		t.Errorf("cost = %.8f, want %.8f", totalCost, wantCost)
	}
}

// TestLiveRun_OpenCode is a gated integration test. Requires AILANG_OPENCODE_LIVE=1
// and a working opencode CLI with a configured provider.
// Run with: AILANG_OPENCODE_LIVE=1 go test ./internal/executor/opencode/... -run TestLiveRun_OpenCode
func TestLiveRun_OpenCode(t *testing.T) {
	if os.Getenv("AILANG_OPENCODE_LIVE") == "" {
		t.Skip("set AILANG_OPENCODE_LIVE=1 to run; requires opencode CLI + provider config")
	}
	if _, err := osexec.LookPath("opencode"); err != nil {
		t.Skipf("opencode binary not found on PATH: %v", err)
	}

	cfg := executor.DefaultConfig()
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := e.HealthCheck(ctx); err != nil {
		t.Skipf("opencode HealthCheck failed: %v", err)
	}

	task := &executor.Task{
		ID:        "opencode-live-smoke",
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
	if result.InputTokens == 0 && result.OutputTokens == 0 {
		t.Error("expected non-zero token counts")
	}
	if result.DurationMS <= 0 {
		t.Errorf("expected positive duration, got %d", result.DurationMS)
	}
}

// collectingHandler implements executor.EventHandler recording events for assertions.
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
