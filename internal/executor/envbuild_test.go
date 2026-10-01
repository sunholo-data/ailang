package executor

import (
	"context"
	"strings"
	"testing"
)

// envMap parses a built env, failing the test on a duplicate key: the builder
// must never rely on os/exec's last-wins de-duplication (AC2).
func envMap(t *testing.T, env []string) map[string]string {
	t.Helper()
	m := make(map[string]string, len(env))
	for _, kv := range env {
		name, value, ok := splitEnvEntry(kv)
		if !ok {
			t.Fatalf("malformed env entry %q", kv)
		}
		k := foldEnvKey(name)
		if _, dup := m[k]; dup {
			t.Fatalf("duplicate env key %q in built environment", name)
		}
		m[k] = value
	}
	return m
}

func mustBuild(t *testing.T, opts EnvironmentOptions) map[string]string {
	t.Helper()
	env, err := BuildEnvironment(opts)
	if err != nil {
		t.Fatalf("BuildEnvironment: %v", err)
	}
	return envMap(t, env)
}

// The keys the harness injects are typically ALSO inherited (a cloud job
// runs inside a coordinator-spawned process). Each must appear once, holding
// the harness value — this is the E3 fixture.
func TestBuildEnvironment_NoDuplicateKeys_InjectedBeatsInherited(t *testing.T) {
	t.Setenv("AILANG_PARENT_TASK_ID", "inherited-parent")
	t.Setenv("AILANG_TASK_ID", "inherited-task")
	t.Setenv("AILANG_CHAIN_ID", "inherited-chain")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
	t.Setenv("PWD", "/inherited")
	t.Setenv("CLAUDECODE", "1")

	task := &Task{
		ID:        "task-1",
		Workspace: t.TempDir(),
		Metadata:  map[string]string{"chain_id": "chain-1", "stage_id": "stage-1", "message_id": "msg-1"},
	}
	m := mustBuild(t, EnvironmentOptions{Task: task, SessionID: "sess-1", Context: context.Background(), EnableClaudeTelemetry: true})

	want := map[string]string{
		"AILANG_TASK_ID":        "task-1",
		"AILANG_SESSION_ID":     "sess-1",
		"AILANG_PARENT_TASK_ID": "task-1",
		"AILANG_CHAIN_ID":       "chain-1",
		"AILANG_STAGE_ID":       "stage-1",
		"AILANG_MESSAGE_ID":     "msg-1",
		"PWD":                   task.Workspace,
		"OTEL_METRICS_EXPORTER": "otlp",
	}
	for k, v := range want {
		if got := m[foldEnvKey(k)]; got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
	if _, ok := m["CLAUDECODE"]; ok {
		t.Error("CLAUDECODE must be stripped from every agent child")
	}
	if _, ok := m["AILANG_RIG_LEASE"]; !ok {
		t.Error("AILANG_RIG_LEASE must always be defined")
	}
}

// Precedence: ExtraEnv > harness-injected > inherited; the executor's own
// required set is applied last (D4).
func TestBuildEnvironment_Precedence(t *testing.T) {
	t.Setenv("MOTOKO_AST_AUTOREAD", "inherited")
	t.Setenv("MOTOKO_CONFIG", "dogfood")
	task := &Task{ExtraEnv: map[string]string{
		"MOTOKO_AST_AUTOREAD": "from-task",
		"MOTOKO_CONFIG":       "from-task",
	}}
	m := mustBuild(t, EnvironmentOptions{Task: task, ExecutorEnv: []string{"MOTOKO_CONFIG=from-executor"}})
	if got := m["MOTOKO_AST_AUTOREAD"]; got != "from-task" {
		t.Errorf("ExtraEnv must beat inherited: got %q", got)
	}
	if got := m["MOTOKO_CONFIG"]; got != "from-executor" {
		t.Errorf("executor-required set must be applied last: got %q", got)
	}
}

// The program policy is harness-injected from Task.PolicyPath, so it survives
// an agent_env block and cannot be overridden through ExtraEnv.
func TestBuildEnvironment_AgentPolicyFromPolicyPath(t *testing.T) {
	t.Setenv("AILANG_AGENT_POLICY", "/inherited/policy.toml")
	task := &Task{PolicyPath: "/run/policy/agent-policy.toml", ExtraEnv: map[string]string{"MOTOKO_AST_AUTOREAD": "1"}}
	m := mustBuild(t, EnvironmentOptions{Task: task})
	if got := m["AILANG_AGENT_POLICY"]; got != task.PolicyPath {
		t.Errorf("AILANG_AGENT_POLICY = %q, want Task.PolicyPath %q", got, task.PolicyPath)
	}
	task.ExtraEnv["AILANG_AGENT_POLICY"] = "/tmp/evil.toml"
	if _, err := BuildEnvironment(EnvironmentOptions{Task: task}); err == nil {
		t.Fatal("ExtraEnv AILANG_AGENT_POLICY must be refused")
	}
}

// Determinism (A1): the same task builds the same env, independent of map
// iteration order.
func TestBuildEnvironment_Deterministic(t *testing.T) {
	task := &Task{ID: "t", ExtraEnv: map[string]string{"A_ONE": "1", "B_TWO": "2", "C_THREE": "3", "D_FOUR": "4"}}
	first, err := BuildEnvironment(EnvironmentOptions{Task: task, SessionID: "s"})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		again, _ := BuildEnvironment(EnvironmentOptions{Task: task, SessionID: "s"})
		if strings.Join(first, "\x00") != strings.Join(again, "\x00") {
			t.Fatal("BuildEnvironment is not deterministic for an identical task")
		}
	}
}

// A refused ExtraEnv name fails the build loudly — on every dispatch path,
// not only the ones that call ValidateTaskCapabilities (E6).
func TestBuildEnvironment_RefusesDeniedExtraEnv(t *testing.T) {
	_, err := BuildEnvironment(EnvironmentOptions{Task: &Task{ExtraEnv: map[string]string{"LD_PRELOAD": "/tmp/x.so"}}})
	if err == nil || !strings.Contains(err.Error(), `"LD_PRELOAD"`) {
		t.Fatalf("BuildEnvironment with LD_PRELOAD = %v, want an error naming the variable", err)
	}
}

func TestEnvSet_UnsetAndWindowsDriveEntries(t *testing.T) {
	e := newEnvSet()
	e.setEntries([]string{"A=1", "B=2", "=C:=C:\\dir", "noequals", "A=3"})
	e.unset("B")
	got := strings.Join(e.environ(), ",")
	if got != "A=3,=C:=C:\\dir" {
		t.Fatalf("environ() = %q", got)
	}
}
