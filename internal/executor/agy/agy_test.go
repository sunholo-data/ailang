package agy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// mockAgy writes a POSIX script standing in for agy. It answers --version with
// version, records its argv and environment, replays a fixture (stdout and
// stderr) and exits with code. body, when set, replaces the replay.
type mockAgy struct {
	dir, path, argsFile, envFile string
}

func newMockAgy(t *testing.T, version, fixture string, code int, body string) *mockAgy {
	t.Helper()
	dir := t.TempDir()
	m := &mockAgy{dir: dir, path: filepath.Join(dir, "agy"),
		argsFile: filepath.Join(dir, "args"), envFile: filepath.Join(dir, "env")}
	replay := body
	if replay == "" && fixture != "" {
		abs, err := filepath.Abs(filepath.Join("testdata", fixture))
		if err != nil {
			t.Fatal(err)
		}
		replay = fmt.Sprintf("cat '%s.ndjson'\ncat '%s.stderr' >&2\nexit %d", abs, abs, code)
	}
	script := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "--version" ]; then echo %s; exit 0; fi
printf '%%s\n' "$@" > '%s'
env > '%s'
%s
`, version, m.argsFile, m.envFile, replay)
	if err := os.WriteFile(m.path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return m
}

func (m *mockAgy) args(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile(m.argsFile)
	if err != nil {
		t.Fatalf("mock agy never ran: %v", err)
	}
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n")
}

func (m *mockAgy) env(t *testing.T) map[string]string {
	t.Helper()
	b, err := os.ReadFile(m.envFile)
	if err != nil {
		t.Fatalf("mock agy never ran: %v", err)
	}
	out := map[string]string{}
	for _, l := range strings.Split(string(b), "\n") {
		if k, v, ok := strings.Cut(l, "="); ok {
			out[k] = v
		}
	}
	return out
}

func newExec(t *testing.T, m *mockAgy) *AgyExecutor {
	t.Helper()
	cfg := executor.DefaultConfig()
	cfg.AgyPath = m.path
	cfg.TimeoutSeconds = 60
	e, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func setupHomes(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("AILANG_AGY_HOME_ROOT", root)
	return root
}

func runTask(t *testing.T, e *AgyExecutor, model string, mutate func(*executor.Task)) *executor.Result {
	t.Helper()
	task := &executor.Task{ID: "t1", Directive: "do the thing", Workspace: t.TempDir(), Model: model, Timeout: 30 * time.Second}
	if mutate != nil {
		mutate(task)
	}
	res, err := e.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return res
}

func TestSplitRoute(t *testing.T) {
	for in, want := range map[string][2]string{
		"gemini-4-neon-high":        {routeSubscription, "gemini-4-neon-high"},
		"api/gemini-3.8-flash-high": {routeAPIKey, "gemini-3.8-flash-high"},
		"api/gemini-4-neon-xhigh":   {routeAPIKey, "gemini-4-neon-xhigh"},
		"apix/gemini-4-neon-xhigh":  {routeSubscription, "apix/gemini-4-neon-xhigh"},
		"gemini-4-carbon-high":      {routeSubscription, "gemini-4-carbon-high"},
	} {
		r, m := splitRoute(in)
		if r != want[0] || m != want[1] {
			t.Errorf("splitRoute(%q) = %q,%q want %q,%q", in, r, m, want[0], want[1])
		}
	}
}

// The edit fixture is a real agy 1.5.15 run (API-key route, 2026-11-23).
func TestFixture_Edit_APIKeyRoute(t *testing.T) {
	root := setupHomes(t)
	t.Setenv("GEMINI_AGENTS_API_KEY", "test-agents-key")
	t.Setenv("GEMINI_API_KEY", "parent-key-must-not-leak")
	m := newMockAgy(t, ExpectedVersion, "edit", 0, "")
	res := runTask(t, newExec(t, m), "api/gemini-3.8-flash-low", nil)

	if !res.Success || res.FinishReason != executor.FinishStop {
		t.Fatalf("want success/stop, got %v/%s: %s", res.Success, res.FinishReason, res.Error)
	}
	if res.ToolCallCount != 1 || res.ToolCalls["write_file"] != 1 {
		t.Errorf("tool calls = %d %v, want 1 write_file (ACTIVE+DONE is one step)", res.ToolCallCount, res.ToolCalls)
	}
	if res.InputTokens != 11461 || res.OutputTokens != 187 || res.CacheReadInputTokens != 7202 || res.ReasonTokens != 0 {
		t.Errorf("tokens in=%d out=%d cache=%d think=%d, want result.usage totals 11461/187/7202/0",
			res.InputTokens, res.OutputTokens, res.CacheReadInputTokens, res.ReasonTokens)
	}
	if res.NumTurns != 2 {
		t.Errorf("NumTurns = %d, want 2 model steps", res.NumTurns)
	}
	if res.SessionID == "" || !strings.Contains(res.Output, "hello.txt") {
		t.Errorf("session=%q output=%q", res.SessionID, res.Output)
	}
	if res.ExecutorVersion != "agy@"+ExpectedVersion {
		t.Errorf("ExecutorVersion = %q", res.ExecutorVersion)
	}
	if res.ProviderData["agy_route"] != routeAPIKey || res.ProviderData["agy_init_model"] != "gemini-3.8-flash-low" {
		t.Errorf("ProviderData = %v", res.ProviderData)
	}

	args := strings.Join(m.args(t), " ")
	for _, want := range []string{"--model gemini-3.8-flash-low", "--mode accept-edits", "--output-format stream-json", "--disable-slash-commands", "-p do the thing"} {
		if !strings.Contains(args, want) {
			t.Errorf("argv missing %q: %s", want, args)
		}
	}
	if strings.Contains(args, "dangerously") {
		t.Errorf("argv must never carry --dangerously-skip-permissions: %s", args)
	}

	env := m.env(t)
	if env["HOME"] != filepath.Join(root, routeAPIKey) {
		t.Errorf("child HOME = %q, want the isolated api-key HOME", env["HOME"])
	}
	if env["GEMINI_API_KEY"] != "test-agents-key" {
		t.Errorf("child GEMINI_API_KEY = %q, want the value of GEMINI_AGENTS_API_KEY", env["GEMINI_API_KEY"])
	}
	if env["AGY_CLI_DISABLE_AUTO_UPDATE"] != "true" {
		t.Error("child must run with AGY_CLI_DISABLE_AUTO_UPDATE=true")
	}
	if _, ok := env["GEMINI_AGENTS_API_KEY"]; ok {
		t.Error("the locked source key must not reach the child under its own name")
	}

	var settings map[string]any
	b, err := os.ReadFile(settingsPath(filepath.Join(root, routeAPIKey)))
	if err != nil || json.Unmarshal(b, &settings) != nil {
		t.Fatalf("settings.json: %v", err)
	}
	if settings["modelProvider"] != "gemini" {
		t.Errorf("api-key settings need modelProvider=gemini, got %v", settings["modelProvider"])
	}
}

func TestSubscriptionRoute_NoCredentialEnv(t *testing.T) {
	root := setupHomes(t)
	t.Setenv("GEMINI_API_KEY", "parent-key")
	t.Setenv("GEMINI_AGENTS_API_KEY", "agents-key")
	m := newMockAgy(t, ExpectedVersion, "edit", 0, "")
	res := runTask(t, newExec(t, m), "gemini-4-neon-high", func(task *executor.Task) {
		task.Pricing = &executor.CostModel{InputTokenCost: 0.003, OutputTokenCost: 0.015, CacheReadCost: 0.00015}
	})
	if !res.Success {
		t.Fatalf("want success: %s", res.Error)
	}
	env := m.env(t)
	for _, k := range []string{"GEMINI_API_KEY", "GOOGLE_API_KEY", "GEMINI_AGENTS_API_KEY", "AGY_ADC_AUTH"} {
		if _, ok := env[k]; ok {
			t.Errorf("subscription child must not see %s", k)
		}
	}
	if env["HOME"] != filepath.Join(root, routeSubscription) {
		t.Errorf("child HOME = %q", env["HOME"])
	}
	if res.CostProvenance != executor.CostListPriceEquivalent {
		t.Errorf("CostProvenance = %q, want list-price-equivalent on the subscription route", res.CostProvenance)
	}
	if res.CostUSD <= 0 {
		t.Errorf("CostUSD = %v, want list-price arithmetic over real tokens", res.CostUSD)
	}
}

// A soft-denied tool exits 0 with status SUCCESS. It must bank as a failure.
func TestFixture_SoftDenial_IsFailure(t *testing.T) {
	setupHomes(t)
	t.Setenv("GEMINI_AGENTS_API_KEY", "k")
	m := newMockAgy(t, ExpectedVersion, "shell_denied", 0, "")
	res := runTask(t, newExec(t, m), "api/gemini-3.8-flash-low", nil)
	if res.Success {
		t.Fatal("a soft-denied run must not bank as success")
	}
	if !strings.Contains(res.Error, "soft-denied 1") || !strings.Contains(res.Error, "command(Bash)") {
		t.Errorf("Error = %q", res.Error)
	}
	if res.ProviderData["agy_denied_actions"] != "command(Bash)" {
		t.Errorf("ProviderData = %v", res.ProviderData)
	}
}

func TestFixture_AllowedShell(t *testing.T) {
	setupHomes(t)
	t.Setenv("GEMINI_AGENTS_API_KEY", "k")
	m := newMockAgy(t, ExpectedVersion, "shell_git_allowed", 0, "")
	res := runTask(t, newExec(t, m), "api/gemini-3.8-flash-low", nil)
	if !res.Success || res.ToolCalls["run_command"] != 1 {
		t.Fatalf("success=%v tools=%v err=%s", res.Success, res.ToolCalls, res.Error)
	}
}

func TestFixture_UnknownModel(t *testing.T) {
	setupHomes(t)
	t.Setenv("GEMINI_AGENTS_API_KEY", "k")
	m := newMockAgy(t, ExpectedVersion, "unknown_model", 1, "")
	res := runTask(t, newExec(t, m), "api/gemini-9-nope", nil)
	if res.Success || !strings.Contains(res.Error, "not recognized") {
		t.Fatalf("success=%v err=%q", res.Success, res.Error)
	}
}

func TestFixture_NoAuth_NamesTheSignin(t *testing.T) {
	setupHomes(t)
	m := newMockAgy(t, ExpectedVersion, "no_auth", 1, "")
	res := runTask(t, newExec(t, m), "gemini-4-neon-high", nil)
	if res.Success || !strings.Contains(res.Error, "make agy-signin") {
		t.Fatalf("success=%v err=%q", res.Success, res.Error)
	}
}

func TestNoResultEvent_IsWireDrift(t *testing.T) {
	setupHomes(t)
	m := newMockAgy(t, ExpectedVersion, "", 0, `echo '{"event":"init","init":{"model":"x"}}'`)
	res := runTask(t, newExec(t, m), "gemini-4-neon-high", nil)
	if res.Success || res.FinishReason != executor.FinishWireDrift {
		t.Fatalf("success=%v finish=%s err=%s", res.Success, res.FinishReason, res.Error)
	}
}

func TestPrintTimeoutExit3_IsTimeout(t *testing.T) {
	setupHomes(t)
	m := newMockAgy(t, ExpectedVersion, "", 0,
		`echo '{"event":"result","result":{"status":"SUCCESS","response":"partial"}}'; exit 3`)
	res := runTask(t, newExec(t, m), "gemini-4-neon-high", nil)
	if res.Success || res.FinishReason != executor.FinishTimeout {
		t.Fatalf("success=%v finish=%s", res.Success, res.FinishReason)
	}
}

// A hung agy (and its children) is killed at Task.Timeout.
func TestHang_KilledAtTimeout_WithChildren(t *testing.T) {
	setupHomes(t)
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "child.pid")
	m := newMockAgy(t, ExpectedVersion, "", 0, fmt.Sprintf("sleep 60 &\necho $! > '%s'\nwait", pidFile))
	start := time.Now()
	res := runTask(t, newExec(t, m), "gemini-4-neon-high", func(task *executor.Task) { task.Timeout = 1500 * time.Millisecond })
	if el := time.Since(start); el > 10*time.Second {
		t.Fatalf("returned after %v; the hard deadline did not hold", el)
	}
	if res.Success || res.FinishReason != executor.FinishTimeout {
		t.Fatalf("success=%v finish=%s", res.Success, res.FinishReason)
	}
	b, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("child pid not recorded: %v", err)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if syscall.Kill(pid, 0) != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	t.Fatalf("grandchild %d survived the timeout kill", pid)
}

func TestAPIKeyRoute_RequiresAgentsKey(t *testing.T) {
	setupHomes(t)
	t.Setenv("GEMINI_AGENTS_API_KEY", "")
	m := newMockAgy(t, ExpectedVersion, "edit", 0, "")
	e := newExec(t, m)
	_, err := e.Execute(context.Background(), &executor.Task{Directive: "x", Workspace: t.TempDir(), Model: "api/gemini-3.8-flash-low", Timeout: time.Minute})
	if err == nil || !strings.Contains(err.Error(), "GEMINI_AGENTS_API_KEY") {
		t.Fatalf("err = %v", err)
	}
}

func TestEmptyModel_FailsLoudly(t *testing.T) {
	setupHomes(t)
	m := newMockAgy(t, ExpectedVersion, "edit", 0, "")
	if _, err := newExec(t, m).Execute(context.Background(), &executor.Task{Directive: "x", Timeout: time.Minute}); err == nil {
		t.Fatal("an empty model must fail loudly")
	}
}

func TestEnsureSettings_KeepsForeignKeys_StripsBillingKeysOnSubscription(t *testing.T) {
	home := t.TempDir()
	p := settingsPath(home)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	seed := `{"model":"Gemini 4 Neon (High)","gcp":{"project":"metered-project"},"modelProvider":"gemini","trustedWorkspaces":["/w"]}`
	if err := os.WriteFile(p, []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureSettings(home, routeSubscription, []string{"command(git)"}, []string{"command(sudo)"}); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	b, _ := os.ReadFile(p)
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["gcp"]; ok {
		t.Error("subscription settings must drop gcp (it moves billing to metered Vertex)")
	}
	if _, ok := got["modelProvider"]; ok {
		t.Error("subscription settings must drop modelProvider")
	}
	if got["model"] != "Gemini 4 Neon (High)" || got["trustedWorkspaces"] == nil {
		t.Errorf("foreign keys lost: %v", got)
	}
	perms, _ := got["permissions"].(map[string]any)
	if perms == nil || fmt.Sprint(perms["allow"]) != "[command(git)]" || fmt.Sprint(perms["deny"]) != "[command(sudo)]" {
		t.Errorf("permissions = %v", got["permissions"])
	}
	// Unchanged policy: no rewrite.
	st1, _ := os.Stat(p)
	time.Sleep(10 * time.Millisecond)
	if err := ensureSettings(home, routeSubscription, []string{"command(git)"}, []string{"command(sudo)"}); err != nil {
		t.Fatal(err)
	}
	st2, _ := os.Stat(p)
	if !st1.ModTime().Equal(st2.ModTime()) {
		t.Error("an unchanged policy rewrote settings.json")
	}
}

func TestPermissionRules(t *testing.T) {
	allow, deny := permissionRules(&executor.Task{})
	if len(allow) == 0 || !contains(allow, "command(ailang)") || !contains(allow, "command(git)") {
		t.Errorf("default allow = %v", allow)
	}
	if !contains(deny, "command(git push)") || !contains(deny, "write_file(.git/)") {
		t.Errorf("deny = %v", deny)
	}
	if a, _ := permissionRules(&executor.Task{AllowedTools: []string{"Read", "Write"}}); len(a) != 0 {
		t.Errorf("no Bash in AllowedTools must grant no commands, got %v", a)
	}
	if a, _ := permissionRules(&executor.Task{AllowedTools: []string{"bash"}}); len(a) == 0 {
		t.Error("Bash in AllowedTools must grant the command allowlist")
	}
}

func TestHealthCheck_VersionPin(t *testing.T) {
	ok := newMockAgy(t, ExpectedVersion, "", 0, "exit 0")
	if err := newExec(t, ok).HealthCheck(context.Background()); err != nil {
		t.Errorf("pinned version: %v", err)
	}
	bad := newMockAgy(t, "1.3.5", "", 0, "exit 0")
	err := newExec(t, bad).HealthCheck(context.Background())
	if err == nil || !strings.Contains(err.Error(), "1.3.5") || !strings.Contains(err.Error(), ExpectedVersion) {
		t.Errorf("mismatch error = %v", err)
	}
}

func TestRegistered(t *testing.T) {
	ex, err := executor.GlobalFactory().GetExecutor("agy")
	if err != nil || ex.Name() != "agy" {
		t.Fatalf("agy not registered: %v", err)
	}
}

// Gated live run: AILANG_AGY_LIVE=1 (API-key route; needs GEMINI_AGENTS_API_KEY
// and agy installed). AILANG_AGY_LIVE_MODEL overrides the model, e.g.
// "gemini-4-neon-high" to exercise the subscription route after sign-in.
func TestLive_OneLineEdit(t *testing.T) {
	if os.Getenv("AILANG_AGY_LIVE") != "1" {
		t.Skip("set AILANG_AGY_LIVE=1 to run a real agy edit")
	}
	model := os.Getenv("AILANG_AGY_LIVE_MODEL")
	if model == "" {
		model = "api/gemini-3.8-flash-low"
	}
	cfg := executor.DefaultConfig()
	e, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.HealthCheck(context.Background()); err != nil {
		t.Fatal(err)
	}
	ws := t.TempDir()
	res, err := e.Execute(context.Background(), &executor.Task{
		Directive: "Create a file hello.txt containing exactly the text: hi",
		Workspace: ws, Model: model, Timeout: 3 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(ws, "hello.txt"))
	t.Logf("success=%v finish=%s in=%d out=%d think=%d cost=$%.4f (%s) model=%v err=%s",
		res.Success, res.FinishReason, res.InputTokens, res.OutputTokens, res.ReasonTokens, res.CostUSD, res.CostProvenance, res.ProviderData["agy_init_model"], res.Error)
	if !res.Success || strings.TrimSpace(string(b)) != "hi" {
		t.Fatalf("live edit failed: file=%q", b)
	}
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

// agy judges workspace writes against the resolved cwd; PWD must agree with it.
func TestWorkspace_PWDIsResolvedPath(t *testing.T) {
	setupHomes(t)
	m := newMockAgy(t, ExpectedVersion, "edit", 0, "")
	ws := t.TempDir()
	runTask(t, newExec(t, m), "gemini-4-neon-high", func(task *executor.Task) { task.Workspace = ws })
	want, err := filepath.EvalSymlinks(ws)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.env(t)["PWD"]; got != want {
		t.Errorf("child PWD = %q, want the resolved workspace %q", got, want)
	}
}
